package net

import (
	"crypto/rand"
	"encoding/binary"
	"math"

	"github.com/spacedreamer99/lodeen/internal/shared/protocol"
)

// Файл вынесен при рефакторинге server.go.

func (s *Server) spawnWells(n int) {
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
		s.wells.Map()[id] = &protocol.Well{
			ID: id,
			X:  pos.X,
			Y:  pos.Y,
			Z:  pos.Z,
		}
	}
	s.log.Info("spawned wells", "count", n)
}

func (s *Server) handleTakeWater(c *Client, wellID string) {
	s.wells.RLock()
	w, ok := s.wells.Map()[wellID]
	s.wells.RUnlock()
	if !ok {
		c.log.Warn("take water: well not found", "well", wellID)
		return
	}
	ps := c.State()
	dx := float64(w.X - ps.X)
	dy := float64(w.Y - ps.Y)
	dz := float64(w.Z - ps.Z)
	if dx*dx+dy*dy+dz*dz > pickupRadius*pickupRadius {
		c.log.Warn("take water: too far")
		return
	}
	inv := c.addItem("water")
	c.sendEnvelope(protocol.TypeInventoryUpdate, protocol.InventoryUpdate{Items: inv})
	c.log.Info("water taken", "well", wellID)
}
