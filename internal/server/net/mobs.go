package net

import (
	"crypto/rand"
	"encoding/binary"
	"math"
	"time"

	"github.com/spacedreamer99/lodeen/internal/shared/protocol"
)

type Mob struct {
	ID           string
	Pos          protocol.Vector3
	HP           int
	MaxHP        int
	Kind         string // "hostile" | "collector" | "pink"
	LastAttackAt time.Time
	Angered      bool
	Inventory    map[string]int
	Contract     string // "" | "gather4" | "guard"
	OwnerID      string
	DeliveredAt  time.Time
}

func (s *Server) spawnMobs(n int) {
	for i := 0; i < n; i++ {
		var b [16]byte
		_, _ = rand.Read(b[:])
		u := float64(binary.BigEndian.Uint64(b[0:8])) / float64(^uint64(0))
		v := float64(binary.BigEndian.Uint64(b[8:16])) / float64(^uint64(0))
		theta := 2 * math.Pi * u
		phi := math.Acos(2*v - 1)
		x := float32(math.Sin(phi) * math.Cos(theta))
		y := float32(math.Cos(phi))
		z := float32(math.Sin(phi) * math.Sin(theta))
		pos := protocol.ClampToSurface(protocol.Vector3{X: x, Y: y, Z: z})
		id := newID()
		kind := "hostile"
		hp := 1
		switch i % 3 {
		case 1:
			kind = "collector"
			hp = 10
		case 2:
			kind = "pink"
			hp = 10
		}
		s.mobs.Map()[id] = &Mob{
			ID:        id,
			Pos:       pos,
			HP:        hp,
			MaxHP:     hp,
			Kind:      kind,
			Inventory: make(map[string]int),
		}
	}
	s.log.Info("spawned mobs", "count", n)
}

const mobAggroD2 = 25.0 * 25.0
const mobAttackRangeD2 = 12.0 * 12.0
const mobKeepMinD2 = 6.0 * 6.0
const mobSpeed = 5.0
const mobAttackDamage = 2
const mobAttackCooldown = 1500 * time.Millisecond

const collectorSearchD2 = 40.0 * 40.0
const collectorPickD2 = 2.5 * 2.5
const collectorSpeed = 4.0

func (s *Server) findClient(id string) *Client {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return s.clients[id]
}

// findPlayerState находит PlayerState по ID.
func (s *Server) findPlayerState(id string) *protocol.PlayerState {
	c := s.findClient(id)
	if c == nil {
		return nil
	}
	st := c.State()
	return &st
}

// mobPlayerInfo — снимок клиента для AI мобов.
type mobPlayerInfo struct {
	client *Client
	state  protocol.PlayerState
}

// tickMobs — оркестратор тика мобов: снимок игроков + диспетчер по виду моба.
func (s *Server) tickMobs(dt float32) {
	s.mobs.Lock()
	defer s.mobs.Unlock()

	players := s.snapshotMobPlayers()
	now := time.Now()

	for _, m := range s.mobs.Map() {
		s.tickMob(m, players, now, dt)
	}
}

// snapshotMobPlayers копирует текущих клиентов и их состояния.
// Делается до s.mobs.Lock() — иначе deadlock с per-client lock'ами.
func (s *Server) snapshotMobPlayers() []mobPlayerInfo {
	s.mu.RLock()
	defer s.mu.RUnlock()
	out := make([]mobPlayerInfo, 0, len(s.clients))
	for _, c := range s.clients {
		out = append(out, mobPlayerInfo{client: c, state: c.State()})
	}
	return out
}

// tickMob — диспетчер по виду моба.
//
//	pink        — убегает от врагов, выполняет контракт
//	collector   — собирает ресурсы (пока не Angered)
//	hostile     — преследует и атакует ближайшего игрока
//	Angered     — collector в ярости тоже атакует
func (s *Server) tickMob(m *Mob, players []mobPlayerInfo, now time.Time, dt float32) {
	switch {
	case m.Kind == "pink":
		s.tickPink(m, dt)
	case m.Kind == "collector" && !m.Angered:
		s.tickCollector(m, dt)
	case m.Kind == "hostile" || m.Angered:
		s.tickHostileMob(m, players, now, dt)
	}
}

// tickHostileMob — поведение враждебного моба по дистанции до игрока:
// слишком близко — отойти, далеко — подойти, в зоне стрельбы — бросок копья.
func (s *Server) tickHostileMob(m *Mob, players []mobPlayerInfo, now time.Time, dt float32) {
	nearest, minD2 := findNearestPlayer(players, m.Pos)
	if nearest == nil {
		return
	}
	dx := nearest.state.X - m.Pos.X
	dy := nearest.state.Y - m.Pos.Y
	dz := nearest.state.Z - m.Pos.Z
	d := float32(math.Sqrt(float64(dx*dx + dy*dy + dz*dz)))
	if d < 0.01 {
		return
	}
	ux, uy, uz := dx/d, dy/d, dz/d

	switch {
	case minD2 < float32(mobKeepMinD2):
		m.moveByDir(-ux, -uy, -uz, mobSpeed*dt)

	case minD2 > float32(mobAttackRangeD2):
		m.moveByDir(ux, uy, uz, mobSpeed*dt)

	default:
		if now.Sub(m.LastAttackAt) >= mobAttackCooldown {
			m.LastAttackAt = now
			s.mobThrowSpear(m, nearest, ux, uy, uz, now)
		}
	}
}

// findNearestPlayer ищет ближайшего игрока в радиусе mobAggroD2.
// Возвращает nil, если никого нет в радиусе.
func findNearestPlayer(players []mobPlayerInfo, pos protocol.Vector3) (*mobPlayerInfo, float32) {
	var nearest *mobPlayerInfo
	minD2 := float32(mobAggroD2)
	for i := range players {
		dx := players[i].state.X - pos.X
		dy := players[i].state.Y - pos.Y
		dz := players[i].state.Z - pos.Z
		d2 := dx*dx + dy*dy + dz*dz
		if d2 < minD2 {
			minD2 = d2
			nearest = &players[i]
		}
	}
	return nearest, minD2
}

// mobThrowSpear создаёт снаряд от моба в сторону цели.
func (s *Server) mobThrowSpear(m *Mob, target *mobPlayerInfo, ux, uy, uz float32, now time.Time) {
	pid := newID()
	s.projectiles.Lock()
	s.projectiles.Map()[pid] = &Projectile{
		ID:         pid,
		Pos:        m.Pos,
		Dir:        protocol.Vector3{X: ux, Y: uy, Z: uz},
		Speed:      12.0,
		OwnerMobID: m.ID,
		TargetID:   target.client.ID,
		SpawnAt:    now,
	}
	s.projectiles.Unlock()
	s.log.Info("mob throws spear", "mob", m.ID, "target", target.client.ID)
}

// moveByDir смещает моба вдоль единичного направления на dist и прижимает к поверхности.
func (m *Mob) moveByDir(ux, uy, uz, dist float32) {
	m.Pos.X += ux * dist
	m.Pos.Y += uy * dist
	m.Pos.Z += uz * dist
	m.Pos = protocol.ClampToSurface(m.Pos)
}
