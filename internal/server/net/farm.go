package net

import (
	"crypto/rand"
	"encoding/binary"
	"math"
	"time"

	"github.com/spacedreamer99/lodeen/internal/shared/protocol"
)

// Этот файл содержит фермерство: спавн ресурсов на поверхности,
// посадку/полив семечка, рост в фрукты.

var resourceTypes = []string{"stone", "wood", "ore", "fruit", "fruit", "fruit", "fruit", "fruit", "water", "liana"}


func (s *Server) spawnResources(n int) {
	for i := 0; i < n; i++ {
		// Случайная точка на сфере (равномерно)
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

		res := protocol.Resource{
			ID:   newID(),
			Type: resourceTypes[i%len(resourceTypes)],
			X:    pos.X,
			Y:    pos.Y,
			Z:    pos.Z,
		}
		s.resources[res.ID] = res
	}
	s.log.Info("spawned resources on surface", "count", n)
}


const plantRadius = 5.0

const growDelay = 30 * time.Second


func (s *Server) handlePlantSeed(c *Client, p protocol.PlantSeed) {
	ps := c.State()
	dx := float64(p.X - ps.X)
	dy := float64(p.Y - ps.Y)
	dz := float64(p.Z - ps.Z)
	if dx*dx+dy*dy+dz*dz > plantRadius*plantRadius {
		c.log.Warn("plant rejected: too far")
		return
	}
	if !c.consumeItem("fruit") {
		return
	}
	id := newID()
	s.resourcesMu.Lock()
	s.resources[id] = protocol.Resource{
		ID:   id,
		Type: "seed",
		X:    p.X,
		Y:    p.Y,
		Z:    p.Z,
	}
	s.resourcesMu.Unlock()

	c.mu.Lock()
	inv := make(map[string]int, len(c.inventory))
	for k, v := range c.inventory {
		inv[k] = v
	}
	c.mu.Unlock()
	c.sendEnvelope(protocol.TypeInventoryUpdate, protocol.InventoryUpdate{Items: inv})
	c.log.Info("seed planted", "id", id)
}


func (s *Server) handleWaterPlant(c *Client, resourceID string) {
	ps := c.State()

	s.resourcesMu.Lock()
	r, ok := s.resources[resourceID]
	if !ok || r.Type != "seed" || r.Watered {
		s.resourcesMu.Unlock()
		return
	}
	dx := float64(r.X - ps.X)
	dy := float64(r.Y - ps.Y)
	dz := float64(r.Z - ps.Z)
	if dx*dx+dy*dy+dz*dz > pickupRadius*pickupRadius {
		s.resourcesMu.Unlock()
		return
	}
	s.resourcesMu.Unlock()

	if !c.consumeItem("water") {
		return
	}

	s.resourcesMu.Lock()
	r, ok = s.resources[resourceID]
	if ok && r.Type == "seed" && !r.Watered {
		r.Watered = true
		r.GrowAt = time.Now().Add(growDelay).UnixMilli()
		s.resources[resourceID] = r
	}
	s.resourcesMu.Unlock()

	c.mu.Lock()
	inv := make(map[string]int, len(c.inventory))
	for k, v := range c.inventory {
		inv[k] = v
	}
	c.mu.Unlock()
	c.sendEnvelope(protocol.TypeInventoryUpdate, protocol.InventoryUpdate{Items: inv})
	c.log.Info("seed watered", "id", resourceID)
}


func (s *Server) tickResources() {
	now := time.Now().UnixMilli()
	s.resourcesMu.Lock()
	defer s.resourcesMu.Unlock()
	for id, r := range s.resources {
		if r.Type != "seed" || !r.Watered || r.GrowAt > now {
			continue
		}
		delete(s.resources, id)
		for i := 0; i < 10; i++ {
			fid := newID()
			s.resources[fid] = protocol.Resource{
				ID:   fid,
				Type: "fruit",
				X:    r.X + float32(i)*1.5,
				Y:    r.Y,
				Z:    r.Z,
			}
		}
		s.log.Info("seed grew", "id", id, "fruits", 2)
	}
}

