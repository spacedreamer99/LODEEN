package net

import (
	"crypto/rand"
	"encoding/binary"
	"math"
	mrand "math/rand"
	"time"

	"github.com/spacedreamer99/lodeen/internal/shared/protocol"
)

// Этот файл содержит всё про мобов: struct Mob, spawnMobs,
// поведение (hostile / collector / pink), контракты, попадания.

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

// tickPink — розовый мирный моб. Убегает от серых и красных, но не от игрока.
// Вызывается при удержании mobsMu.Lock() в tickMobs.
const pinkFleeD2 = 15.0 * 15.0
const pinkPanicD2 = 8.0 * 8.0
const pinkSpeed = 6.0
const pinkSearchD2 = 40.0 * 40.0
const pinkPickD2 = 2.5 * 2.5
const pinkDeliverD2 = 4.0 * 4.0
const pinkGuardRangeD2 = 12.0 * 12.0
const pinkGuardCloseD2 = 5.0 * 5.0
const pinkThrowDamage = 2
const pinkThrowCooldown = 1500 * time.Millisecond
const pinkShootRangeD2 = 15.0 * 15.0

// tickPink — розовый мирный моб. Убегает от серых и красных, но не от игрока.
// Выполняет контракты: gather4 (собрать 4 ресурса и принести) или guard (охранять игрока).
func (s *Server) tickPink(m *Mob, dt float32) {
	// Все розовые автостреляют в угрозы (hostile или Angered collector).
	s.pinkAutoShoot(m)

	// Контракт gather4.
	if m.Contract == "gather4" {
		s.tickPinkGather(m, dt)
		return
	}
	// Контракт guard.
	if m.Contract == "guard" {
		s.tickPinkGuard(m, dt)
		return
	}
	// Без контракта — убегает от угроз.
	s.tickPinkFlee(m, dt)
}

// pinkAutoShoot — розовый стреляет в ближайшую угрозу.
// Угроза: hostile (всегда) или collector с Angered=true.
func (s *Server) pinkAutoShoot(m *Mob) {
	var threat *Mob
	bestD2 := float32(pinkShootRangeD2)
	for _, other := range s.mobs {
		if other.ID == m.ID {
			continue
		}
		isThreat := other.Kind == "hostile" ||
			(other.Kind == "collector" && other.Angered)
		if !isThreat {
			continue
		}
		dx := other.Pos.X - m.Pos.X
		dy := other.Pos.Y - m.Pos.Y
		dz := other.Pos.Z - m.Pos.Z
		d2 := dx*dx + dy*dy + dz*dz
		if d2 < bestD2 {
			threat = other
			bestD2 = d2
		}
	}
	if threat == nil {
		return
	}
	now := time.Now()
	if now.Sub(m.LastAttackAt) < pinkThrowCooldown {
		return
	}
	d := float32(math.Sqrt(float64(bestD2)))
	if d < 0.01 {
		return
	}
	m.LastAttackAt = now
	pid := newID()
	s.projMu.Lock()
	s.projectiles[pid] = &Projectile{
		ID:  pid,
		Pos: m.Pos,
		Dir: protocol.Vector3{
			X: (threat.Pos.X - m.Pos.X) / d,
			Y: (threat.Pos.Y - m.Pos.Y) / d,
			Z: (threat.Pos.Z - m.Pos.Z) / d,
		},
		Speed:      12.0,
		OwnerMobID: m.ID,
		TargetID:   threat.ID,
		SpawnAt:    now,
	}
	s.projMu.Unlock()
	s.log.Info("pink shoots hostile", "mob", m.ID, "target", threat.ID, "dist", d)
}

// tickPinkFlee — убегает от серых и красных.
func (s *Server) tickPinkFlee(m *Mob, dt float32) {
	var threat *Mob
	bestD2 := float32(pinkFleeD2)
	for _, other := range s.mobs {
		if other.ID == m.ID {
			continue
		}
		if other.Kind != "hostile" && other.Kind != "collector" {
			continue
		}
		dx := other.Pos.X - m.Pos.X
		dy := other.Pos.Y - m.Pos.Y
		dz := other.Pos.Z - m.Pos.Z
		d2 := dx*dx + dy*dy + dz*dz
		if d2 < bestD2 {
			threat = other
			bestD2 = d2
		}
	}
	if threat == nil {
		return
	}
	dx := m.Pos.X - threat.Pos.X
	dy := m.Pos.Y - threat.Pos.Y
	dz := m.Pos.Z - threat.Pos.Z
	d := float32(math.Sqrt(float64(dx*dx + dy*dy + dz*dz)))
	if d < 0.01 {
		return
	}
	speed := float32(pinkSpeed)
	if bestD2 < float32(pinkPanicD2) {
		speed *= 1.5
	}
	m.Pos.X += (dx / d) * speed * dt
	m.Pos.Y += (dy / d) * speed * dt
	m.Pos.Z += (dz / d) * speed * dt
	m.Pos = protocol.ClampToSurface(m.Pos)
}

// tickPinkGather — собирает 4 ресурса и несёт владельцу.
func (s *Server) tickPinkGather(m *Mob, dt float32) {
	// Сколько уже собрал.
	total := 0
	for _, q := range m.Inventory {
		total += q
	}

	// Если собрал 4+ — идём к владельцу и отдаём.
	if total >= 4 {
		if s.pinkDeliver(m) {
			return
		}
		// Идти к владельцу.
		owner := s.findPlayerState(m.OwnerID)
		if owner == nil {
			return
		}
		dx := owner.X - m.Pos.X
		dy := owner.Y - m.Pos.Y
		dz := owner.Z - m.Pos.Z
		d := float32(math.Sqrt(float64(dx*dx + dy*dy + dz*dz)))
		if d < 0.01 {
			return
		}
		m.Pos.X += (dx / d) * pinkSpeed * dt
		m.Pos.Y += (dy / d) * pinkSpeed * dt
		m.Pos.Z += (dz / d) * pinkSpeed * dt
		m.Pos = protocol.ClampToSurface(m.Pos)
		return
	}

	// Ищем ближайший ресурс.
	var target *protocol.Resource
	bestD2 := float32(pinkSearchD2)
	s.resourcesMu.RLock()
	for _, res := range s.resources {
		dx := res.X - m.Pos.X
		dy := res.Y - m.Pos.Y
		dz := res.Z - m.Pos.Z
		d2 := dx*dx + dy*dy + dz*dz
		if d2 < bestD2 {
			r := res
			target = &r
			bestD2 = d2
		}
	}
	s.resourcesMu.RUnlock()

	if target == nil {
		return
	}

	// Подобрать если рядом.
	if bestD2 < float32(pinkPickD2) {
		s.resourcesMu.Lock()
		if _, ok := s.resources[target.ID]; ok {
			delete(s.resources, target.ID)
			m.Inventory[target.Type]++
		}
		s.resourcesMu.Unlock()
		return
	}

	// Идти к ресурсу.
	dx := target.X - m.Pos.X
	dy := target.Y - m.Pos.Y
	dz := target.Z - m.Pos.Z
	d := float32(math.Sqrt(float64(dx*dx + dy*dy + dz*dz)))
	if d < 0.01 {
		return
	}
	m.Pos.X += (dx / d) * pinkSpeed * dt
	m.Pos.Y += (dy / d) * pinkSpeed * dt
	m.Pos.Z += (dz / d) * pinkSpeed * dt
	m.Pos = protocol.ClampToSurface(m.Pos)
}

// pinkDeliver — если рядом с владельцем, передаёт ему все ресурсы. Возвращает true если отдал.
func (s *Server) pinkDeliver(m *Mob) bool {
	owner := s.findClient(m.OwnerID)
	if owner == nil {
		return false
	}
	ps := owner.State()
	dx := ps.X - m.Pos.X
	dy := ps.Y - m.Pos.Y
	dz := ps.Z - m.Pos.Z
	if dx*dx+dy*dy+dz*dz > float32(pinkDeliverD2) {
		return false
	}
	// Передаём всё в инвентарь игрока.
	owner.mu.Lock()
	for item, q := range m.Inventory {
		owner.inventory[item] += q
	}
	inv := make(map[string]int, len(owner.inventory))
	for k, v := range owner.inventory {
		inv[k] = v
	}
	owner.mu.Unlock()
	owner.sendEnvelope(protocol.TypeInventoryUpdate, protocol.InventoryUpdate{Items: inv})
	s.log.Info("pink delivered", "mob", m.ID, "items", len(m.Inventory))
	m.Inventory = make(map[string]int)
	m.Contract = ""
	m.OwnerID = ""
	return true
}

// tickPinkGuard — охраняет владельца, атакует ближайшего красного.
func (s *Server) tickPinkGuard(m *Mob, dt float32) {
	owner := s.findPlayerState(m.OwnerID)
	if owner == nil {
		m.Contract = ""
		m.OwnerID = ""
		return
	}

	// Ищем ближайшего красного в радиусе.
	var target *Mob
	bestD2 := float32(pinkGuardRangeD2)
	for _, other := range s.mobs {
		if other.Kind != "hostile" {
			continue
		}
		dx := other.Pos.X - m.Pos.X
		dy := other.Pos.Y - m.Pos.Y
		dz := other.Pos.Z - m.Pos.Z
		d2 := dx*dx + dy*dy + dz*dz
		if d2 < bestD2 {
			target = other
			bestD2 = d2
		}
	}

	if target != nil {
		return
	}

	// Красных рядом нет — следуем за владельцем на дистанции 5.
	dx := owner.X - m.Pos.X
	dy := owner.Y - m.Pos.Y
	dz := owner.Z - m.Pos.Z
	d2 := dx*dx + dy*dy + dz*dz
	if d2 < float32(pinkGuardCloseD2) {
		return
	}
	d := float32(math.Sqrt(float64(d2)))
	if d < 0.01 {
		return
	}
	m.Pos.X += (dx / d) * pinkSpeed * dt
	m.Pos.Y += (dy / d) * pinkSpeed * dt
	m.Pos.Z += (dz / d) * pinkSpeed * dt
	m.Pos = protocol.ClampToSurface(m.Pos)
}

// findClient находит клиента по ID.
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

// tickCollector — серый моб идёт к ближайшему ресурсу и подбирает его.
// Вызывается, когда держится mobsMu.Lock() в tickMobs.
func (s *Server) tickCollector(m *Mob, dt float32) {
	// Найти ближайший ресурс.
	var target *protocol.Resource
	bestD2 := float32(collectorSearchD2)

	s.resourcesMu.RLock()
	for _, res := range s.resources {
		dx := res.X - m.Pos.X
		dy := res.Y - m.Pos.Y
		dz := res.Z - m.Pos.Z
		d2 := dx*dx + dy*dy + dz*dz
		if d2 < bestD2 {
			r := res
			target = &r
			bestD2 = d2
		}
	}
	s.resourcesMu.RUnlock()

	if target == nil {
		return
	}

	// Подобрать, если рядом.
	if bestD2 < float32(collectorPickD2) {
		s.resourcesMu.Lock()
		if _, ok := s.resources[target.ID]; ok {
			delete(s.resources, target.ID)
			m.Inventory[target.Type]++
		}
		s.resourcesMu.Unlock()
		return
	}

	// Идти к ресурсу.
	dx := target.X - m.Pos.X
	dy := target.Y - m.Pos.Y
	dz := target.Z - m.Pos.Z
	d := float32(math.Sqrt(float64(dx*dx + dy*dy + dz*dz)))
	if d < 0.01 {
		return
	}
	m.Pos.X += (dx / d) * collectorSpeed * dt
	m.Pos.Y += (dy / d) * collectorSpeed * dt
	m.Pos.Z += (dz / d) * collectorSpeed * dt
	m.Pos = protocol.ClampToSurface(m.Pos)
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

func (s *Server) tickProjectiles(dt float32) {
	now := time.Now()
	const projTTL = 3 * time.Second
	const projHitD2 = 1.5 * 1.5

	// Снимок мобов ДО projMu — иначе deadlock с tickMobs.
	s.mobsMu.RLock()
	mobPos := make(map[string]protocol.Vector3, len(s.mobs))
	for id, mo := range s.mobs {
		mobPos[id] = mo.Pos
	}
	s.mobsMu.RUnlock()

	type mobHit struct {
		id string
	}
	var mobHits []mobHit

	s.projMu.Lock()
	for id, p := range s.projectiles {
		if now.Sub(p.SpawnAt) > projTTL {
			delete(s.projectiles, id)
			continue
		}
		p.Pos.X += p.Dir.X * p.Speed * dt
		p.Pos.Y += p.Dir.Y * p.Speed * dt
		p.Pos.Z += p.Dir.Z * p.Speed * dt

		// 1. Проверка попадания в целевого игрока.
		s.mu.RLock()
		var target *Client
		for _, c := range s.clients {
			if c.ID == p.TargetID {
				target = c
				break
			}
		}
		s.mu.RUnlock()
		if target != nil {
			ps := target.State()
			dx := ps.X - p.Pos.X
			dy := ps.Y - p.Pos.Y
			dz := ps.Z - p.Pos.Z
			if dx*dx+dy*dy+dz*dz < projHitD2 {
				target.damage(mobAttackDamage)
				s.log.Info("mob spear hit", "target", p.TargetID, "dmg", mobAttackDamage)
				delete(s.projectiles, id)
			}
			continue
		}

		// 2. Проверка попадания в моба-цель.
		if mp, ok := mobPos[p.TargetID]; ok {
			dx := mp.X - p.Pos.X
			dy := mp.Y - p.Pos.Y
			dz := mp.Z - p.Pos.Z
			if dx*dx+dy*dy+dz*dz < projHitD2 {
				mobHits = append(mobHits, mobHit{id: p.TargetID})
				delete(s.projectiles, id)
			}
		}
		// Если цель исчезла — снаряд летит дальше до TTL.
	}
	s.projMu.Unlock()

	// Применяем попадания в мобов.
	if len(mobHits) > 0 {
		type killDrop struct {
			pos    protocol.Vector3
			kind   string
			mobInv map[string]int
		}
		var kills []killDrop

		s.mobsMu.Lock()
		for _, h := range mobHits {
			mo, ok := s.mobs[h.id]
			if !ok {
				continue
			}
			mo.HP -= mobAttackDamage
			if mo.HP <= 0 {
				kills = append(kills, killDrop{
					pos:    mo.Pos,
					kind:   mo.Kind,
					mobInv: mo.Inventory,
				})
				delete(s.mobs, h.id)
				s.log.Info("mob killed by projectile", "id", h.id, "kind", mo.Kind)
			} else {
				s.log.Info("mob hit by projectile", "id", h.id, "hp", mo.HP)
			}
		}
		s.mobsMu.Unlock()

		// Спавним дроп после unlock.
		if len(kills) > 0 {
			s.resourcesMu.Lock()
			for _, k := range kills {
				switch k.kind {
				case "hostile":
					for i := 0; i < 10; i++ {
						rid := newID()
						theta := mrand.Float32() * 2 * math.Pi
						rr := mrand.Float32() * 2.0
						dx := float32(math.Cos(float64(theta))) * rr
						dz := float32(math.Sin(float64(theta))) * rr
						pp := protocol.ClampToSurface(protocol.Vector3{
							X: k.pos.X + dx, Y: k.pos.Y, Z: k.pos.Z + dz,
						})
						s.resources[rid] = protocol.Resource{
							ID: rid, Type: "spear",
							X: pp.X, Y: pp.Y, Z: pp.Z,
						}
					}
				case "collector":
					for itemType, qty := range k.mobInv {
						for i := 0; i < qty; i++ {
							rid := newID()
							theta := mrand.Float32() * 2 * math.Pi
							rr := mrand.Float32() * 2.5
							dx := float32(math.Cos(float64(theta))) * rr
							dz := float32(math.Sin(float64(theta))) * rr
							pp := protocol.ClampToSurface(protocol.Vector3{
								X: k.pos.X + dx, Y: k.pos.Y, Z: k.pos.Z + dz,
							})
							s.resources[rid] = protocol.Resource{
								ID: rid, Type: itemType,
								X: pp.X, Y: pp.Y, Z: pp.Z,
							}
						}
					}
				}
			}
			s.resourcesMu.Unlock()
		}
	}
}

func (s *Server) handleHitMob(c *Client, mobID string) {
	s.mobsMu.Lock()
	m, ok := s.mobs[mobID]
	if !ok {
		s.mobsMu.Unlock()
		c.log.Warn("hit mob: not found")
		return
	}
	// Валидация: игрок рядом.
	ps := c.State()
	dx := float64(m.Pos.X - ps.X)
	dy := float64(m.Pos.Y - ps.Y)
	dz := float64(m.Pos.Z - ps.Z)
	if dx*dx+dy*dy+dz*dz > 100.0*100.0 {
		s.mobsMu.Unlock()
		c.log.Warn("hit mob: too far")
		return
	}
	if !c.consumeItem("spear") {
		s.mobsMu.Unlock()
		return
	}
	m.HP--
	if m.Kind == "collector" {
		m.Angered = true
	}
	killed := m.HP <= 0
	pos := m.Pos
	kind := m.Kind
	mobInv := m.Inventory
	if killed {
		delete(s.mobs, mobID)
	}
	s.mobsMu.Unlock()

	if killed {
		s.resourcesMu.Lock()
		if kind == "pink" {
			// Розовый — без дропа.
		} else if kind == "collector" {
			// Выпадают все собранные ресурсы компактной кучей.
			for itemType, qty := range mobInv {
				for i := 0; i < qty; i++ {
					rid := newID()
					theta := mrand.Float32() * 2 * math.Pi
					rr := mrand.Float32() * 2.5
					dx := float32(math.Cos(float64(theta))) * rr
					dz := float32(math.Sin(float64(theta))) * rr
					pp := protocol.ClampToSurface(protocol.Vector3{
						X: pos.X + dx,
						Y: pos.Y,
						Z: pos.Z + dz,
					})
					s.resources[rid] = protocol.Resource{
						ID:   rid,
						Type: itemType,
						X:    pp.X,
						Y:    pp.Y,
						Z:    pp.Z,
					}
				}
			}
		} else {
			// Красный моб — 10 копий.
			for i := 0; i < 10; i++ {
				rid := newID()
				theta := mrand.Float32() * 2 * math.Pi
				r := mrand.Float32() * 2.0
				dx := float32(math.Cos(float64(theta))) * r
				dz := float32(math.Sin(float64(theta))) * r
				pp := protocol.ClampToSurface(protocol.Vector3{
					X: pos.X + dx,
					Y: pos.Y,
					Z: pos.Z + dz,
				})
				s.resources[rid] = protocol.Resource{
					ID:   rid,
					Type: "spear",
					X:    pp.X,
					Y:    pp.Y,
					Z:    pp.Z,
				}
			}
		}
		s.resourcesMu.Unlock()
		c.log.Info("mob killed", "id", mobID, "kind", kind)
	} else {
		c.log.Info("mob hit", "id", mobID, "hp", m.HP, "kind", kind)
	}

	// Inventory update
	c.mu.Lock()
	inv := make(map[string]int, len(c.inventory))
	for k, v := range c.inventory {
		inv[k] = v
	}
	c.mu.Unlock()
	c.sendEnvelope(protocol.TypeInventoryUpdate, protocol.InventoryUpdate{Items: inv})
}

func (s *Server) handleAcceptContract(c *Client, mobID, contractID string) {
	s.mobsMu.Lock()
	m, ok := s.mobs[mobID]
	if !ok {
		s.mobsMu.Unlock()
		c.log.Warn("contract: mob not found")
		return
	}
	if m.Kind != "pink" {
		s.mobsMu.Unlock()
		c.log.Warn("contract: not a pink mob")
		return
	}
	if m.Contract != "" {
		s.mobsMu.Unlock()
		c.log.Warn("contract: mob already busy", "existing", m.Contract)
		return
	}
	if contractID != "gather4" && contractID != "guard" {
		s.mobsMu.Unlock()
		c.log.Warn("contract: unknown id", "id", contractID)
		return
	}

	// Проверяем цену.
	switch contractID {
	case "gather4":
		if !c.consumeItem("fruit") {
			s.mobsMu.Unlock()
			c.log.Warn("contract: no fruit")
			return
		}
	case "guard":
		c.mu.Lock()
		hasSpear := c.inventory["spear"] >= 10
		if hasSpear {
			c.inventory["spear"] -= 10
		}
		c.mu.Unlock()
		if !hasSpear {
			s.mobsMu.Unlock()
			c.log.Warn("contract: not enough spears")
			return
		}
	}

	m.Contract = contractID
	m.OwnerID = c.ID
	if contractID == "gather4" {
		m.Inventory = make(map[string]int)
	}
	s.mobsMu.Unlock()

	c.log.Info("contract accepted", "mob", mobID, "contract", contractID)

	c.mu.Lock()
	inv := make(map[string]int, len(c.inventory))
	for k, v := range c.inventory {
		inv[k] = v
	}
	c.mu.Unlock()
	c.sendEnvelope(protocol.TypeInventoryUpdate, protocol.InventoryUpdate{Items: inv})
}
