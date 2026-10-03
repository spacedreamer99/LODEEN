package net

import (
	"math"
	"time"

	"github.com/spacedreamer99/lodeen/internal/shared/protocol"
)

// tickPink — розовый мирный моб. Убегает от серых и красных, но не от игрока.
// Вызывается при удержании s.mobs.Lock() в tickMobs.
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
	for _, other := range s.mobs.Map() {
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
	s.projectiles.Lock()
	s.projectiles.Map()[pid] = &Projectile{
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
	s.projectiles.Unlock()
	s.log.Info("pink shoots hostile", "mob", m.ID, "target", threat.ID, "dist", d)
}

// tickPinkFlee — убегает от серых и красных.
func (s *Server) tickPinkFlee(m *Mob, dt float32) {
	var threat *Mob
	bestD2 := float32(pinkFleeD2)
	for _, other := range s.mobs.Map() {
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
	s.resources.RLock()
	for _, res := range s.resources.Map() {
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
	s.resources.RUnlock()

	if target == nil {
		return
	}

	// Подобрать если рядом.
	if bestD2 < float32(pinkPickD2) {
		s.resources.Lock()
		if _, ok := s.resources.Map()[target.ID]; ok {
			delete(s.resources.Map(), target.ID)
			m.Inventory[target.Type]++
		}
		s.resources.Unlock()
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
	for _, other := range s.mobs.Map() {
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
