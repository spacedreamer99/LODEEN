package net

import (
	"math"

	"github.com/spacedreamer99/lodeen/internal/shared/protocol"
)

// WorldState — состояние системы. Сейчас: Земля на круговой орбите вокруг Солнца.
type WorldState struct {
	EarthPos protocol.Vector3
	EarthVel protocol.Vector3
}

// NewWorldState — Земля на круговой орбите вокруг Солнца радиусом SunDistance.
// Позиция — относительно Солнца (а не относительно центра мира).
func NewWorldState() *WorldState {
	r := float64(protocol.SunDistance)
	vCirc := math.Sqrt(float64(protocol.MuSun) / r)
	// Радиус от Солнца к Земле — по +X от SunPos. Скорость перпендикулярна (по +Z).
	return &WorldState{
		EarthPos: protocol.Vector3{
			X: protocol.SunPos.X + protocol.SunDistance,
			Y: protocol.SunPos.Y,
			Z: protocol.SunPos.Z,
		},
		EarthVel: protocol.Vector3{X: 0, Y: 0, Z: float32(vCirc)},
	}
}

// Tick — гравитация от Солнца двигает Землю.
func (w *WorldState) Tick(dt float32) {
	dx := protocol.SunPos.X - w.EarthPos.X
	dy := protocol.SunPos.Y - w.EarthPos.Y
	dz := protocol.SunPos.Z - w.EarthPos.Z
	dist := float32(math.Sqrt(float64(dx*dx + dy*dy + dz*dz)))
	if dist < 1 {
		return
	}
	g := protocol.MuSun / (dist * dist)
	ax := dx / dist * g
	ay := dy / dist * g
	az := dz / dist * g
	w.EarthVel.X += ax * dt
	w.EarthVel.Y += ay * dt
	w.EarthVel.Z += az * dt
	w.EarthPos.X += w.EarthVel.X * dt
	w.EarthPos.Y += w.EarthVel.Y * dt
	w.EarthPos.Z += w.EarthVel.Z * dt
}
