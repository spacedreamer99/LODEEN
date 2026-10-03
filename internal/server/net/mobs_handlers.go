package net

import (
	"math"
	mrand "math/rand"

	"github.com/spacedreamer99/lodeen/internal/shared/protocol"
)

func (s *Server) handleHitMob(c *Client, mobID string) {
	s.mobs.Lock()
	m, ok := s.mobs.Map()[mobID]
	if !ok {
		s.mobs.Unlock()
		c.log.Warn("hit mob: not found")
		return
	}
	// Валидация: игрок рядом.
	ps := c.State()
	dx := float64(m.Pos.X - ps.X)
	dy := float64(m.Pos.Y - ps.Y)
	dz := float64(m.Pos.Z - ps.Z)
	if dx*dx+dy*dy+dz*dz > 100.0*100.0 {
		s.mobs.Unlock()
		c.log.Warn("hit mob: too far")
		return
	}
	if !c.consumeItem("spear") {
		s.mobs.Unlock()
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
		delete(s.mobs.Map(), mobID)
	}
	s.mobs.Unlock()

	if killed {
		s.resources.Lock()
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
					s.resources.Map()[rid] = protocol.Resource{
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
				s.resources.Map()[rid] = protocol.Resource{
					ID:   rid,
					Type: "spear",
					X:    pp.X,
					Y:    pp.Y,
					Z:    pp.Z,
				}
			}
		}
		s.resources.Unlock()
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
	s.mobs.Lock()
	m, ok := s.mobs.Map()[mobID]
	if !ok {
		s.mobs.Unlock()
		c.log.Warn("contract: mob not found")
		return
	}
	if m.Kind != "pink" {
		s.mobs.Unlock()
		c.log.Warn("contract: not a pink mob")
		return
	}
	if m.Contract != "" {
		s.mobs.Unlock()
		c.log.Warn("contract: mob already busy", "existing", m.Contract)
		return
	}
	if contractID != "gather4" && contractID != "guard" {
		s.mobs.Unlock()
		c.log.Warn("contract: unknown id", "id", contractID)
		return
	}

	// Проверяем цену.
	switch contractID {
	case "gather4":
		if !c.consumeItem("fruit") {
			s.mobs.Unlock()
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
			s.mobs.Unlock()
			c.log.Warn("contract: not enough spears")
			return
		}
	}

	m.Contract = contractID
	m.OwnerID = c.ID
	if contractID == "gather4" {
		m.Inventory = make(map[string]int)
	}
	s.mobs.Unlock()

	c.log.Info("contract accepted", "mob", mobID, "contract", contractID)

	c.mu.Lock()
	inv := make(map[string]int, len(c.inventory))
	for k, v := range c.inventory {
		inv[k] = v
	}
	c.mu.Unlock()
	c.sendEnvelope(protocol.TypeInventoryUpdate, protocol.InventoryUpdate{Items: inv})
}
