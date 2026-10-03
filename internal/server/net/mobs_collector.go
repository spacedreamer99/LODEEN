package net

import (
	"math"

	"github.com/spacedreamer99/lodeen/internal/shared/protocol"
)

// tickCollector — серый моб идёт к ближайшему ресурсу и подбирает его.
// Вызывается, когда держится mobsMu.Lock() в tickMobs.
func (s *Server) tickCollector(m *Mob, dt float32) {
	// Найти ближайший ресурс.
	var target *protocol.Resource
	bestD2 := float32(collectorSearchD2)

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

	// Подобрать, если рядом.
	if bestD2 < float32(collectorPickD2) {
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
	m.Pos.X += (dx / d) * collectorSpeed * dt
	m.Pos.Y += (dy / d) * collectorSpeed * dt
	m.Pos.Z += (dz / d) * collectorSpeed * dt
	m.Pos = protocol.ClampToSurface(m.Pos)
}
