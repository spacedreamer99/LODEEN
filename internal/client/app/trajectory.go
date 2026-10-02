package app

import (
	"math"

	"github.com/spacedreamer99/lodeen/internal/shared/protocol"
)

func predictTrajectory(helioPos, helioVel, earthPos0, earthVel protocol.Vector3, dt float32, steps int) []protocol.Vector3 {
	p := helioPos
	v := helioVel
	ep := earthPos0
	ev := earthVel // Земля тоже ускоряется — не const!
	pts := make([]protocol.Vector3, 0, steps)
	for i := 0; i < steps; i++ {
		// ── Гравитация Солнца на Землю ──
		ex := ep.X - protocol.SunPos.X
		ey := ep.Y - protocol.SunPos.Y
		ez := ep.Z - protocol.SunPos.Z
		distE := float32(math.Sqrt(float64(ex*ex + ey*ey + ez*ez)))
		var axE, ayE, azE float32
		if distE > 1 {
			gE := protocol.MuSun / (distE * distE)
			axE = -ex / distE * gE
			ayE = -ey / distE * gE
			azE = -ez / distE * gE
		}
		ev.X += axE * dt
		ev.Y += ayE * dt
		ev.Z += azE * dt
		ep.X += ev.X * dt
		ep.Y += ev.Y * dt
		ep.Z += ev.Z * dt

		// ── Гравитация Солнца на ракету ──
		sx := p.X - protocol.SunPos.X
		sy := p.Y - protocol.SunPos.Y
		sz := p.Z - protocol.SunPos.Z
		distSun := float32(math.Sqrt(float64(sx*sx + sy*sy + sz*sz)))

		// ── Гравитация Земли на ракету ──
		relPos := protocol.Vector3{X: p.X - ep.X, Y: p.Y - ep.Y, Z: p.Z - ep.Z}
		distEarth := float32(math.Sqrt(float64(
			relPos.X*relPos.X + relPos.Y*relPos.Y + relPos.Z*relPos.Z)))

		var gx, gy, gz float32

		if distSun > 1 {
			g := protocol.MuSun / (distSun * distSun)
			gx += -sx / distSun * g
			gy += -sy / distSun * g
			gz += -sz / distSun * g
		}
		if distEarth < protocol.SOIEarth && distEarth > 1 {
			g := protocol.MuEarth / (distEarth * distEarth)
			gx += -relPos.X / distEarth * g
			gy += -relPos.Y / distEarth * g
			gz += -relPos.Z / distEarth * g
		}

		// ── Интеграция ракеты ──
		v.X += gx * dt
		v.Y += gy * dt
		v.Z += gz * dt
		p.X += v.X * dt
		p.Y += v.Y * dt
		p.Z += v.Z * dt

		pts = append(pts, p)

		// Early exit: возврат к точке старта (замкнутая орбита вокруг Солнца).
		if i > 200 {
			dxs := p.X - helioPos.X
			dys := p.Y - helioPos.Y
			dzs := p.Z - helioPos.Z
			if dxs*dxs+dys*dys+dzs*dzs < 25 {
				break
			}
		}
	}
	return pts
}
