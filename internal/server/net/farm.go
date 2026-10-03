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
		s.resources.Map()[res.ID] = res
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
	// Прижимаем seed к поверхности — иначе висит в воздухе на высоте камеры.
	pos := protocol.ClampToSurface(protocol.Vector3{X: p.X, Y: p.Y, Z: p.Z})
	id := newID()
	s.resources.Lock()
	s.resources.Map()[id] = protocol.Resource{
		ID:   id,
		Type: "seed",
		X:    pos.X,
		Y:    pos.Y,
		Z:    pos.Z,
	}
	s.resources.Unlock()

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

	s.resources.Lock()
	r, ok := s.resources.Map()[resourceID]
	if !ok || r.Type != "seed" || r.Watered {
		s.resources.Unlock()
		return
	}
	dx := float64(r.X - ps.X)
	dy := float64(r.Y - ps.Y)
	dz := float64(r.Z - ps.Z)
	if dx*dx+dy*dy+dz*dz > pickupRadius*pickupRadius {
		s.resources.Unlock()
		return
	}
	s.resources.Unlock()

	if !c.consumeItem("water") {
		return
	}

	s.resources.Lock()
	r, ok = s.resources.Map()[resourceID]
	if ok && r.Type == "seed" && !r.Watered {
		r.Watered = true
		r.GrowAt = time.Now().Add(growDelay).UnixMilli()
		s.resources.Map()[resourceID] = r
	}
	s.resources.Unlock()

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
	s.resources.Lock()
	defer s.resources.Unlock()
	for id, r := range s.resources.Map() {
		if r.Type != "seed" || !r.Watered || r.GrowAt > now {
			continue
		}
		delete(s.resources.Map(), id)
		const harvestFruits = 2
		for i := 0; i < harvestFruits; i++ {
			fid := newID()
			s.resources.Map()[fid] = protocol.Resource{
				ID:   fid,
				Type: "fruit",
				X:    r.X + float32(i)*1.5,
				Y:    r.Y,
				Z:    r.Z,
			}
		}
		s.log.Info("seed grew", "id", id, "fruits", harvestFruits)
	}
}
