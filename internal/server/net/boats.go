package net

import (
	"math"

	"github.com/spacedreamer99/lodeen/internal/shared/protocol"
)

// Файл вынесен при рефакторинге server.go.

type Boat struct {
	ID      string
	Pos     protocol.Vector3
	Yaw     float32
	RiderID string
}

func (s *Server) handlePlaceBoat(c *Client, p protocol.PlaceBoat) {
	// Проверка: точка на воде.
	dir := protocol.Vector3{X: p.X, Y: p.Y, Z: p.Z}
	l := float32(math.Sqrt(float64(dir.X*dir.X + dir.Y*dir.Y + dir.Z*dir.Z)))
	if l < 0.01 {
		return
	}
	nx, ny, nz := dir.X/l, dir.Y/l, dir.Z/l
	th := protocol.TerrainHeight(nx, ny, nz)
	if th >= protocol.SeaLevel {
		c.log.Warn("boat: not on water")
		return
	}
	if !c.consumeItem("boat") {
		c.log.Warn("boat: no boat in inventory")
		return
	}
	id := newID()
	s.boatsMu.Lock()
	s.boats[id] = &Boat{
		ID: id,
		Pos: protocol.Vector3{
			X: nx * protocol.SeaLevel,
			Y: ny * protocol.SeaLevel,
			Z: nz * protocol.SeaLevel,
		},
		Yaw: p.Yaw,
	}
	s.boatsMu.Unlock()
	c.log.Info("boat placed", "id", id)

	c.mu.Lock()
	inv := make(map[string]int, len(c.inventory))
	for k, v := range c.inventory {
		inv[k] = v
	}
	c.mu.Unlock()
	c.sendEnvelope(protocol.TypeInventoryUpdate, protocol.InventoryUpdate{Items: inv})
}

func (s *Server) handleEnterBoat(c *Client, boatID string) {
	ps := c.State()

	// Если уже в лодке/на мамонте — сойти.
	s.boatsMu.Lock()
	for _, b := range s.boats {
		if b.RiderID == c.ID {
			b.RiderID = ""
		}
	}
	s.mammothsMu.Lock()
	for _, m := range s.mammoths {
		if m.RiderID == c.ID {
			m.RiderID = ""
		}
	}
	s.mammothsMu.Unlock()

	if boatID == "" {
		s.boatsMu.Unlock()
		c.log.Info("dismounted boat")
		return
	}

	b, ok := s.boats[boatID]
	if !ok {
		s.boatsMu.Unlock()
		c.log.Warn("boat: not found")
		return
	}
	dx := float64(b.Pos.X - ps.X)
	dy := float64(b.Pos.Y - ps.Y)
	dz := float64(b.Pos.Z - ps.Z)
	if dx*dx+dy*dy+dz*dz > 8.0*8.0 {
		s.boatsMu.Unlock()
		c.log.Warn("boat: too far")
		return
	}
	if b.RiderID != "" {
		s.boatsMu.Unlock()
		c.log.Warn("boat: occupied")
		return
	}
	b.RiderID = c.ID
	s.boatsMu.Unlock()
	c.log.Info("boat entered", "id", boatID)
}

func (s *Server) tickBoats() {
	s.boatsMu.Lock()
	defer s.boatsMu.Unlock()
	for _, b := range s.boats {
		if b.RiderID == "" {
			continue
		}
		s.mu.RLock()
		var rider *protocol.PlayerState
		for _, c := range s.clients {
			if c.ID == b.RiderID {
				st := c.State()
				rider = &st
				break
			}
		}
		s.mu.RUnlock()
		if rider == nil {
			b.RiderID = ""
			continue
		}
		// Лодка плывёт точно под всадником, на уровне моря.
		dir := protocol.Vector3{X: rider.X, Y: rider.Y, Z: rider.Z}
		l := float32(math.Sqrt(float64(dir.X*dir.X + dir.Y*dir.Y + dir.Z*dir.Z)))
		if l < 0.01 {
			continue
		}
		nx, ny, nz := dir.X/l, dir.Y/l, dir.Z/l
		b.Pos = protocol.Vector3{
			X: nx * protocol.SeaLevel,
			Y: ny * protocol.SeaLevel,
			Z: nz * protocol.SeaLevel,
		}
	}
}
