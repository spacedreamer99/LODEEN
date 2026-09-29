package net

import (
	"github.com/spacedreamer99/lodeen/internal/shared/protocol"
)

// Файл вынесен при рефакторинге server.go.

type House struct {
	ID       string
	Pos      protocol.Vector3
	Yaw      float32
	DoorOpen bool
}


func (s *Server) handlePlaceHouse(c *Client, p protocol.PlaceHouse) {
	if !c.consumeItem("house") {
		c.log.Warn("house: no house item in inventory")
		return
	}
	c.mu.Lock()
	inv := make(map[string]int, len(c.inventory))
	for k, v := range c.inventory {
		inv[k] = v
	}
	c.mu.Unlock()

	id := newID()
	s.housesMu.Lock()
	s.houses[id] = &House{
		ID:  id,
		Pos: protocol.Vector3{X: p.X, Y: p.Y, Z: p.Z},
		Yaw: p.Yaw,
	}
	s.housesMu.Unlock()

	c.sendEnvelope(protocol.TypeInventoryUpdate, protocol.InventoryUpdate{Items: inv})
	c.log.Info("house placed", "id", id)
}


func (s *Server) handleToggleDoor(c *Client, houseID string) {
	ps := c.State()

	s.housesMu.Lock()
	h, ok := s.houses[houseID]
	if !ok {
		s.housesMu.Unlock()
		c.log.Warn("door: house not found")
		return
	}
	dx := float64(h.Pos.X - ps.X)
	dy := float64(h.Pos.Y - ps.Y)
	dz := float64(h.Pos.Z - ps.Z)
	if dx*dx+dy*dy+dz*dz > 8.0*8.0 {
		s.housesMu.Unlock()
		c.log.Warn("door: too far")
		return
	}
	h.DoorOpen = !h.DoorOpen
	open := h.DoorOpen
	s.housesMu.Unlock()

	c.log.Info("door toggled", "id", houseID, "open", open)
}

