package net

import (
	"math"

	"github.com/spacedreamer99/lodeen/internal/shared/protocol"
)

// WorldState — состояние двойной звёздной системы.
// Звёзды статичны. Вокруг каждой крутится планета.
type WorldState struct {
	// Система 1: Солнце + Земля
	EarthPos protocol.Vector3
	EarthVel protocol.Vector3

	// Система 2: Star2 + Planet2
	Planet2Pos protocol.Vector3
	Planet2Vel protocol.Vector3
}

// NewWorldState — обе планеты на круговых орбитах вокруг своих звёзд.
func NewWorldState() *WorldState {
	// Земля: круговая орбита вокруг Солнца.
	rE := float64(protocol.SunDistance)
	vCircE := math.Sqrt(float64(protocol.MuSun) / rE)

	// Planet2: круговая орбита вокруг Star2.
	rP := float64(protocol.Planet2OrbitRadius)
	vCircP := math.Sqrt(float64(protocol.MuStar2) / rP)

	return &WorldState{
		EarthPos: protocol.Vector3{
			X: protocol.SunPos.X + protocol.SunDistance,
			Y: protocol.SunPos.Y,
			Z: protocol.SunPos.Z,
		},
		EarthVel: protocol.Vector3{X: 0, Y: 0, Z: float32(vCircE)},

		Planet2Pos: protocol.Vector3{
			X: protocol.Star2Pos.X + protocol.Planet2OrbitRadius,
			Y: protocol.Star2Pos.Y,
			Z: protocol.Star2Pos.Z,
		},
		Planet2Vel: protocol.Vector3{X: 0, Y: 0, Z: float32(vCircP)},
	}
}

// Tick — обе планеты двигаются под гравитацией своих звёзд.
func (w *WorldState) Tick(dt float32) {
	// Земля вокруг Солнца.
	tickBody(&w.EarthPos, &w.EarthVel, protocol.SunPos, protocol.MuSun, dt)
	// Planet2 вокруг Star2.
	tickBody(&w.Planet2Pos, &w.Planet2Vel, protocol.Star2Pos, protocol.MuStar2, dt)
}

// tickBody — интегрирует движение тела в поле центральной массы.
func tickBody(pos, vel *protocol.Vector3, center protocol.Vector3, mu float32, dt float32) {
	dx := center.X - pos.X
	dy := center.Y - pos.Y
	dz := center.Z - pos.Z
	dist := float32(math.Sqrt(float64(dx*dx + dy*dy + dz*dz)))
	if dist < 1 {
		return
	}
	g := mu / (dist * dist)
	ax := dx / dist * g
	ay := dy / dist * g
	az := dz / dist * g
	vel.X += ax * dt
	vel.Y += ay * dt
	vel.Z += az * dt
	pos.X += vel.X * dt
	pos.Y += vel.Y * dt
	pos.Z += vel.Z * dt
}
