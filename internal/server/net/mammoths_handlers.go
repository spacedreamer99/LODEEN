package net

import (
	"time"

	"github.com/spacedreamer99/lodeen/internal/shared/protocol"
)

func (s *Server) handleTameMammoth(c *Client, mammothID string) {
	ps := c.State()

	s.mammoths.Lock()
	m, ok := s.mammoths.Map()[mammothID]
	if !ok {
		s.mammoths.Unlock()
		c.log.Warn("tame: mammoth not found")
		return
	}
	dx := float64(m.Pos.X - ps.X)
	dy := float64(m.Pos.Y - ps.Y)
	dz := float64(m.Pos.Z - ps.Z)
	if dx*dx+dy*dy+dz*dz > 5.0*5.0 {
		s.mammoths.Unlock()
		c.log.Warn("tame: too far")
		return
	}

	if !m.Tamed {
		m.Tamed = true
		s.mammoths.Unlock()
		if !c.consumeItem("fruit") {
			// откатываем
			s.mammoths.Lock()
			m.Tamed = false
			s.mammoths.Unlock()
			return
		}
		c.log.Info("mammoth tamed", "id", mammothID)
	} else {
		if !c.consumeItem("fruit") {
			s.mammoths.Unlock()
			return
		}
		m.FedCount++
		fed := m.FedCount
		s.mammoths.Unlock()
		c.log.Info("mammoth fed", "id", mammothID, "count", fed)
	}

	c.mu.Lock()
	inv := make(map[string]int, len(c.inventory))
	for k, v := range c.inventory {
		inv[k] = v
	}
	c.mu.Unlock()
	c.sendEnvelope(protocol.TypeInventoryUpdate, protocol.InventoryUpdate{Items: inv})
}

func (s *Server) handleLeashMammoth(c *Client, mammothID string) {
	ps := c.State()

	s.mammoths.Lock()
	m, ok := s.mammoths.Map()[mammothID]
	if !ok {
		s.mammoths.Unlock()
		c.log.Warn("leash: not found")
		return
	}
	dx := float64(m.Pos.X - ps.X)
	dy := float64(m.Pos.Y - ps.Y)
	dz := float64(m.Pos.Z - ps.Z)
	if dx*dx+dy*dy+dz*dz > 5.0*5.0 {
		s.mammoths.Unlock()
		c.log.Warn("leash: too far")
		return
	}
	if !m.Tamed {
		s.mammoths.Unlock()
		c.log.Warn("leash: not tamed")
		return
	}
	myID := c.ID
	if m.LeashedTo == myID {
		m.LeashedTo = ""
		s.mammoths.Unlock()
		c.log.Info("mammoth unleashed", "id", mammothID)
		return
	}
	m.LeashedTo = myID
	s.mammoths.Unlock()

	c.log.Info("mammoth leashed", "id", mammothID)
}

func (s *Server) handleSaddleMammoth(c *Client, mammothID string) {
	ps := c.State()

	s.mammoths.Lock()
	m, ok := s.mammoths.Map()[mammothID]
	if !ok {
		s.mammoths.Unlock()
		c.log.Warn("saddle: not found")
		return
	}
	dx := float64(m.Pos.X - ps.X)
	dy := float64(m.Pos.Y - ps.Y)
	dz := float64(m.Pos.Z - ps.Z)
	if dx*dx+dy*dy+dz*dz > 5.0*5.0 {
		s.mammoths.Unlock()
		c.log.Warn("saddle: too far")
		return
	}
	if !m.Tamed {
		s.mammoths.Unlock()
		c.log.Warn("saddle: not tamed")
		return
	}
	if m.Baby {
		s.mammoths.Unlock()
		c.log.Warn("saddle: baby")
		return
	}
	if !m.Saddle {
		// ставим: нужен предмет
		if !c.consumeItem("saddle") {
			s.mammoths.Unlock()
			return
		}
		m.Saddle = true
		s.mammoths.Unlock()
		c.log.Info("saddle installed", "id", mammothID)
	} else {
		// снимаем
		if m.RiderID != "" {
			s.mammoths.Unlock()
			c.log.Warn("saddle: rider on it")
			return
		}
		m.Saddle = false
		s.mammoths.Unlock()
		c.addItem("saddle")
		c.log.Info("saddle removed", "id", mammothID)
	}

	c.mu.Lock()
	inv := make(map[string]int, len(c.inventory))
	for k, v := range c.inventory {
		inv[k] = v
	}
	c.mu.Unlock()
	c.sendEnvelope(protocol.TypeInventoryUpdate, protocol.InventoryUpdate{Items: inv})
}

func (s *Server) handleRideMammoth(c *Client, mammothID string) {
	ps := c.State()

	s.mammoths.Lock()
	for _, m := range s.mammoths.Map() {
		if m.RiderID == c.ID {
			m.RiderID = ""
			s.mammoths.Unlock()
			c.log.Info("dismounted")
			return
		}
	}
	if mammothID == "" {
		s.mammoths.Unlock()
		return
	}
	m, ok := s.mammoths.Map()[mammothID]
	if !ok {
		s.mammoths.Unlock()
		c.log.Warn("ride: not found")
		return
	}
	dx := float64(m.Pos.X - ps.X)
	dy := float64(m.Pos.Y - ps.Y)
	dz := float64(m.Pos.Z - ps.Z)
	if dx*dx+dy*dy+dz*dz > 6.0*6.0 {
		s.mammoths.Unlock()
		c.log.Warn("ride: too far")
		return
	}
	if !m.Tamed || !m.Saddle || m.RiderID != "" {
		s.mammoths.Unlock()
		c.log.Warn("ride: not ready")
		return
	}
	m.RiderID = c.ID
	s.mammoths.Unlock()
	c.log.Info("mounted", "id", mammothID)
}

// mammothHitResult — итог удара по мамонту.
type mammothHitResult struct {
	killed  bool
	hpAfter int
	pos     protocol.Vector3 // позиция для дропа (валидна только если killed)
}

// handleHitMammoth — оркестратор удара по мамонту:
// дебаунс → валидация → применение урона → дроп (если убит).
func (s *Server) handleHitMammoth(c *Client, mammothID string) {
	now := time.Now()

	// Тихо гасим дубли HitMammoth в пределах 200 мс — норма для клиента.
	if now.Sub(c.lastHitAt) < 200*time.Millisecond {
		return
	}

	// Валидация: игрок недавно бросил копьё.
	if now.Sub(c.lastThrowAt) > 2*time.Second {
		c.log.Warn("hit rejected: no recent throw")
		return
	}

	c.lastHitAt = now

	res, ok := s.applyHitMammoth(c, mammothID)
	if !ok {
		return
	}

	if res.killed {
		s.spawnMammothDrops(res.pos)
		c.log.Info("mammoth KILLED", "id", mammothID)
	} else {
		c.log.Info("mammoth hit", "id", mammothID, "hp", res.hpAfter)
	}
}

// applyHitMammoth под одним lock'ом: находит мамонта, проверяет дистанцию,
// наносит урон, удаляет убитого. Возвращает ok=false, если удар отклонён.
func (s *Server) applyHitMammoth(c *Client, mammothID string) (mammothHitResult, bool) {
	s.mammoths.Lock()
	defer s.mammoths.Unlock()

	m, ok := s.mammoths.Map()[mammothID]
	if !ok {
		c.log.Warn("hit rejected: mammoth not found")
		return mammothHitResult{}, false
	}

	ps := c.State()
	dx := m.Pos.X - ps.X
	dy := m.Pos.Y - ps.Y
	dz := m.Pos.Z - ps.Z
	const maxD2 float32 = 80.0 * 80.0
	if dx*dx+dy*dy+dz*dz > maxD2 {
		c.log.Warn("hit rejected: too far")
		return mammothHitResult{}, false
	}

	m.HP--
	res := mammothHitResult{
		killed:  m.HP <= 0,
		hpAfter: m.HP,
	}
	if res.killed {
		res.pos = m.Pos
		delete(s.mammoths.Map(), mammothID)
	}
	return res, true
}

// spawnMammothDrops кладёт на землю мясо и копьё рядом с тушей.
// Копьё сдвинуто — чтобы не совпадало с мясом и подбиралось отдельно.
func (s *Server) spawnMammothDrops(pos protocol.Vector3) {
	s.resources.Lock()
	defer s.resources.Unlock()

	meatID := newID()
	s.resources.Map()[meatID] = protocol.Resource{
		ID:   meatID,
		Type: "meat",
		X:    pos.X,
		Y:    pos.Y,
		Z:    pos.Z,
	}

	spearID := newID()
	s.resources.Map()[spearID] = protocol.Resource{
		ID:   spearID,
		Type: "spear",
		X:    pos.X + 1.5,
		Y:    pos.Y,
		Z:    pos.Z + 1.5,
	}
}
