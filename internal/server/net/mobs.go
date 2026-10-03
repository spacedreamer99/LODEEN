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
		s.mobs[id] = &Mob{
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

func (s *Server) tickMobs(dt float32) {
	s.mobsMu.Lock()
	defer s.mobsMu.Unlock()

	// Снимок игроков
	type pInfo struct {
		client *Client
		state  protocol.PlayerState
	}
	s.mu.RLock()
	players := make([]pInfo, 0, len(s.clients))
	for _, c := range s.clients {
		players = append(players, pInfo{client: c, state: c.State()})
	}
	s.mu.RUnlock()

	now := time.Now()
	for _, m := range s.mobs {
		// Розовый — убегает от серых и красных.
		if m.Kind == "pink" {
			s.tickPink(m, dt)
			continue
		}
		// Коллектор и не разозлён — собирает ресурсы и не атакует.
		if m.Kind == "collector" && !m.Angered {
			s.tickCollector(m, dt)
			continue
		}
		// Только враги (красные или разозлённые серые) идут сюда.
		if m.Kind != "hostile" && !m.Angered {
			continue
		}
		var nearest *pInfo
		minD2 := float32(mobAggroD2)
		for i := range players {
			dx := players[i].state.X - m.Pos.X
			dy := players[i].state.Y - m.Pos.Y
			dz := players[i].state.Z - m.Pos.Z
			d2 := dx*dx + dy*dy + dz*dz
			if d2 < minD2 {
				minD2 = d2
				nearest = &players[i]
			}
		}
		if nearest == nil {
			continue
		}

		// Направление и расстояние до игрока.
		dx := nearest.state.X - m.Pos.X
		dy := nearest.state.Y - m.Pos.Y
		dz := nearest.state.Z - m.Pos.Z
		d := float32(math.Sqrt(float64(dx*dx + dy*dy + dz*dz)))
		if d < 0.01 {
			continue
		}

		// Выбор поведения по дистанции.
		switch {
		case minD2 < float32(mobKeepMinD2):
			// Слишком близко — отойти от игрока.
			m.Pos.X -= (dx / d) * mobSpeed * dt
			m.Pos.Y -= (dy / d) * mobSpeed * dt
			m.Pos.Z -= (dz / d) * mobSpeed * dt
			m.Pos = protocol.ClampToSurface(m.Pos)

		case minD2 > float32(mobAttackRangeD2):
			// Далеко — подойти к игроку.
			m.Pos.X += (dx / d) * mobSpeed * dt
			m.Pos.Y += (dy / d) * mobSpeed * dt
			m.Pos.Z += (dz / d) * mobSpeed * dt
			m.Pos = protocol.ClampToSurface(m.Pos)

		default:
			// В зоне стрельбы (6–12 юнитов) — держим позицию и бросаем копьё.
			if now.Sub(m.LastAttackAt) >= mobAttackCooldown {
				m.LastAttackAt = now
				pid := newID()
				s.projMu.Lock()
				s.projectiles[pid] = &Projectile{
					ID:         pid,
					Pos:        m.Pos,
					Dir:        protocol.Vector3{X: dx / d, Y: dy / d, Z: dz / d},
					Speed:      12.0,
					OwnerMobID: m.ID,
					TargetID:   nearest.client.ID,
					SpawnAt:    now,
				}
				s.projMu.Unlock()
				s.log.Info("mob throws spear", "mob", m.ID, "target", nearest.client.ID, "dist", d)
			}
		}
	}
}
