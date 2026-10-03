package net

import (
	"time"

	"github.com/spacedreamer99/lodeen/internal/shared/protocol"
)

func (s *Server) handleTameMammoth(c *Client, mammothID string) {
	ps := c.State()

	s.mammothsMu.Lock()
	m, ok := s.mammoths[mammothID]
	if !ok {
		s.mammothsMu.Unlock()
		c.log.Warn("tame: mammoth not found")
		return
	}
	dx := float64(m.Pos.X - ps.X)
	dy := float64(m.Pos.Y - ps.Y)
	dz := float64(m.Pos.Z - ps.Z)
	if dx*dx+dy*dy+dz*dz > 5.0*5.0 {
		s.mammothsMu.Unlock()
		c.log.Warn("tame: too far")
		return
	}

	if !m.Tamed {
		m.Tamed = true
		s.mammothsMu.Unlock()
		if !c.consumeItem("fruit") {
			// откатываем
			s.mammothsMu.Lock()
			m.Tamed = false
			s.mammothsMu.Unlock()
			return
		}
		c.log.Info("mammoth tamed", "id", mammothID)
	} else {
		if !c.consumeItem("fruit") {
			s.mammothsMu.Unlock()
			return
		}
		m.FedCount++
		fed := m.FedCount
		s.mammothsMu.Unlock()
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

	s.mammothsMu.Lock()
	m, ok := s.mammoths[mammothID]
	if !ok {
		s.mammothsMu.Unlock()
		c.log.Warn("leash: not found")
		return
	}
	dx := float64(m.Pos.X - ps.X)
	dy := float64(m.Pos.Y - ps.Y)
	dz := float64(m.Pos.Z - ps.Z)
	if dx*dx+dy*dy+dz*dz > 5.0*5.0 {
		s.mammothsMu.Unlock()
		c.log.Warn("leash: too far")
		return
	}
	if !m.Tamed {
		s.mammothsMu.Unlock()
		c.log.Warn("leash: not tamed")
		return
	}
	myID := c.ID
	if m.LeashedTo == myID {
		m.LeashedTo = ""
		s.mammothsMu.Unlock()
		c.log.Info("mammoth unleashed", "id", mammothID)
		return
	}
	m.LeashedTo = myID
	s.mammothsMu.Unlock()

	c.log.Info("mammoth leashed", "id", mammothID)
}

func (s *Server) handleSaddleMammoth(c *Client, mammothID string) {
	ps := c.State()

	s.mammothsMu.Lock()
	m, ok := s.mammoths[mammothID]
	if !ok {
		s.mammothsMu.Unlock()
		c.log.Warn("saddle: not found")
		return
	}
	dx := float64(m.Pos.X - ps.X)
	dy := float64(m.Pos.Y - ps.Y)
	dz := float64(m.Pos.Z - ps.Z)
	if dx*dx+dy*dy+dz*dz > 5.0*5.0 {
		s.mammothsMu.Unlock()
		c.log.Warn("saddle: too far")
		return
	}
	if !m.Tamed {
		s.mammothsMu.Unlock()
		c.log.Warn("saddle: not tamed")
		return
	}
	if m.Baby {
		s.mammothsMu.Unlock()
		c.log.Warn("saddle: baby")
		return
	}
	if !m.Saddle {
		// ставим: нужен предмет
		if !c.consumeItem("saddle") {
			s.mammothsMu.Unlock()
			return
		}
		m.Saddle = true
		s.mammothsMu.Unlock()
		c.log.Info("saddle installed", "id", mammothID)
	} else {
		// снимаем
		if m.RiderID != "" {
			s.mammothsMu.Unlock()
			c.log.Warn("saddle: rider on it")
			return
		}
		m.Saddle = false
		s.mammothsMu.Unlock()
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

	s.mammothsMu.Lock()
	for _, m := range s.mammoths {
		if m.RiderID == c.ID {
			m.RiderID = ""
			s.mammothsMu.Unlock()
			c.log.Info("dismounted")
			return
		}
	}
	if mammothID == "" {
		s.mammothsMu.Unlock()
		return
	}
	m, ok := s.mammoths[mammothID]
	if !ok {
		s.mammothsMu.Unlock()
		c.log.Warn("ride: not found")
		return
	}
	dx := float64(m.Pos.X - ps.X)
	dy := float64(m.Pos.Y - ps.Y)
	dz := float64(m.Pos.Z - ps.Z)
	if dx*dx+dy*dy+dz*dz > 6.0*6.0 {
		s.mammothsMu.Unlock()
		c.log.Warn("ride: too far")
		return
	}
	if !m.Tamed || !m.Saddle || m.RiderID != "" {
		s.mammothsMu.Unlock()
		c.log.Warn("ride: not ready")
		return
	}
	m.RiderID = c.ID
	s.mammothsMu.Unlock()
	c.log.Info("mounted", "id", mammothID)
}

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

	// Валидация: мамонт существует.
	s.mammothsMu.Lock()
	m, ok := s.mammoths[mammothID]
	if !ok {
		s.mammothsMu.Unlock()
		c.log.Warn("hit rejected: mammoth not found")
		return
	}

	// Валидация: мамонт в разумной близости от игрока.
	ps := c.State()
	dx := m.Pos.X - ps.X
	dy := m.Pos.Y - ps.Y
	dz := m.Pos.Z - ps.Z
	dist2 := dx*dx + dy*dy + dz*dz
	const maxD2 float32 = 80.0 * 80.0
	if dist2 > maxD2 {
		s.mammothsMu.Unlock()
		c.log.Warn("hit rejected: too far", "d2", dist2)
		return
	}

	m.HP--
	var meatPos protocol.Vector3
	killed := m.HP <= 0
	if killed {
		meatPos = m.Pos
		delete(s.mammoths, mammothID)
	}
	s.mammothsMu.Unlock()

	if killed {
		s.resourcesMu.Lock()
		meatID := newID()
		s.resources[meatID] = protocol.Resource{
			ID:   meatID,
			Type: "meat",
			X:    meatPos.X,
			Y:    meatPos.Y,
			Z:    meatPos.Z,
		}
		// Небольшой сдвиг, чтобы копьё не совпадало с мясом и его можно было подобрать отдельно.
		spearID := newID()
		s.resources[spearID] = protocol.Resource{
			ID:   spearID,
			Type: "spear",
			X:    meatPos.X + 1.5,
			Y:    meatPos.Y,
			Z:    meatPos.Z + 1.5,
		}
		s.resourcesMu.Unlock()
		c.log.Info("mammoth KILLED", "id", mammothID)
	} else {
		c.log.Info("mammoth hit", "id", mammothID, "hp", m.HP)
	}
}
