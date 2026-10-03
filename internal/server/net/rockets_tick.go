package net

import (
	"math"
	"time"

	"github.com/spacedreamer99/lodeen/internal/shared/protocol"
)

// tickPilotedRocket — оркестратор тика пилотируемой ракеты.
func (s *Server) tickPilotedRocket(r *Rocket, ep, ev protocol.Vector3, dt float32) {
	relPos := protocol.Vector3{X: r.Pos.X - ep.X, Y: r.Pos.Y - ep.Y, Z: r.Pos.Z - ep.Z}
	relVel := protocol.Vector3{X: r.Vel.X - ev.X, Y: r.Vel.Y - ev.Y, Z: r.Vel.Z - ev.Z}
	distEarth := float32(math.Sqrt(float64(
		relPos.X*relPos.X + relPos.Y*relPos.Y + relPos.Z*relPos.Z)))

	sx := r.Pos.X - protocol.SunPos.X
	sy := r.Pos.Y - protocol.SunPos.Y
	sz := r.Pos.Z - protocol.SunPos.Z
	distSun := float32(math.Sqrt(float64(sx*sx + sy*sy + sz*sz)))

	isEarth := distEarth < protocol.SOIEarth

	// Автопилот — выбирает TargetUp.
	applyAutopilot(r, relPos, relVel, distEarth)

	// Руль — плавно разворачивает Up к TargetUp.
	steerUpTowardTarget(r, dt)

	// Гравитация: Солнце всегда, Земля если в её SOI.
	gx, gy, gz := gravityAccel(relPos, distEarth, distSun, isEarth, sx, sy, sz)

	// Тяга + расход топлива.
	tx, ty, tz := consumeThrust(r, dt)

	// Интеграция скорости.
	r.Vel.X += (gx + tx) * dt
	r.Vel.Y += (gy + ty) * dt
	r.Vel.Z += (gz + tz) * dt

	// Атмосферное сопротивление (только Земля).
	if isEarth {
		applyAtmosphericDrag(r, ev, distEarth, dt)
	}

	// Мягкий лимит скорости.
	applySpeedLimit(r, ev, isEarth)

	// Интеграция позиции.
	r.Pos.X += r.Vel.X * dt
	r.Pos.Y += r.Vel.Y * dt
	r.Pos.Z += r.Vel.Z * dt

	// Коллизия с Землёй.
	if isEarth {
		collideWithEarth(r, ep, ev)
	}

	// Обновляем фрейм (LocalPos, PrimaryBody, DistSun, SOIRadius).
	updateRocketFrame(r, ep, isEarth, distSun)

	// Коллизия с Солнцем.
	collideWithSun(s, r)

	// Орбитальные параметры и флаг стабильной орбиты.
	updateOrbitalParams(s, r, ep, isEarth)

	// Периодический лог.
	logRocketTick(s, r, distEarth, distSun)
}

// applyAutopilot — выбирает TargetUp по режиму автопилота.
func applyAutopilot(r *Rocket, relPos, relVel protocol.Vector3, distEarth float32) {
	if r.AutoPilot == "" {
		return
	}
	vLen := float32(math.Sqrt(float64(
		relVel.X*relVel.X + relVel.Y*relVel.Y + relVel.Z*relVel.Z)))
	rLen := distEarth

	var target protocol.Vector3
	switch r.AutoPilot {
	case "prograde":
		if vLen > 0.1 {
			target = protocol.Vector3{X: relVel.X / vLen, Y: relVel.Y / vLen, Z: relVel.Z / vLen}
		}
	case "retrograde":
		if vLen > 0.1 {
			target = protocol.Vector3{X: -relVel.X / vLen, Y: -relVel.Y / vLen, Z: -relVel.Z / vLen}
		}
	case "radial_out":
		if rLen > 0.1 {
			target = protocol.Vector3{X: relPos.X / rLen, Y: relPos.Y / rLen, Z: relPos.Z / rLen}
		}
	case "radial_in":
		if rLen > 0.1 {
			target = protocol.Vector3{X: -relPos.X / rLen, Y: -relPos.Y / rLen, Z: -relPos.Z / rLen}
		}
	case "normal", "antinormal":
		nx := relPos.Y*relVel.Z - relPos.Z*relVel.Y
		ny := relPos.Z*relVel.X - relPos.X*relVel.Z
		nz := relPos.X*relVel.Y - relPos.Y*relVel.X
		nl := float32(math.Sqrt(float64(nx*nx + ny*ny + nz*nz)))
		if nl > 0.01 {
			k := float32(1.0)
			if r.AutoPilot == "antinormal" {
				k = -1.0
			}
			target = protocol.Vector3{X: nx / nl * k, Y: ny / nl * k, Z: nz / nl * k}
		}
	}

	if target.X != 0 || target.Y != 0 || target.Z != 0 {
		r.TargetUp = target
	}
}

// steerUpTowardTarget — плавно разворачивает Up к TargetUp с угловой скоростью steerRate.
func steerUpTowardTarget(r *Rocket, dt float32) {
	if r.TargetUp.X == 0 && r.TargetUp.Y == 0 && r.TargetUp.Z == 0 {
		return
	}
	tLen := float32(math.Sqrt(float64(
		r.TargetUp.X*r.TargetUp.X + r.TargetUp.Y*r.TargetUp.Y + r.TargetUp.Z*r.TargetUp.Z)))
	if tLen <= 0.01 {
		return
	}
	tx := r.TargetUp.X / tLen
	ty := r.TargetUp.Y / tLen
	tz := r.TargetUp.Z / tLen

	dot := r.Up.X*tx + r.Up.Y*ty + r.Up.Z*tz
	if dot > 1 {
		dot = 1
	}
	if dot < -1 {
		dot = -1
	}
	angle := float32(math.Acos(float64(dot)))
	maxAngle := steerRate * dt

	if angle > maxAngle && angle > 0.001 {
		k := maxAngle / angle
		r.Up.X += (tx - r.Up.X) * k
		r.Up.Y += (ty - r.Up.Y) * k
		r.Up.Z += (tz - r.Up.Z) * k
		upLen := float32(math.Sqrt(float64(
			r.Up.X*r.Up.X + r.Up.Y*r.Up.Y + r.Up.Z*r.Up.Z)))
		if upLen > 0.01 {
			r.Up.X /= upLen
			r.Up.Y /= upLen
			r.Up.Z /= upLen
		}
	} else {
		r.Up.X = tx
		r.Up.Y = ty
		r.Up.Z = tz
	}
}

// gravityAccel — ускорение от Солнца (всегда) и от Земли (если в её SOI).
func gravityAccel(relPos protocol.Vector3, distEarth, distSun float32, isEarth bool, sx, sy, sz float32) (gx, gy, gz float32) {
	if distSun > 1 {
		g := protocol.MuSun / (distSun * distSun)
		gx += -sx / distSun * g
		gy += -sy / distSun * g
		gz += -sz / distSun * g
	}
	if isEarth && distEarth > 1 {
		g := protocol.MuEarth / (distEarth * distEarth)
		if g > maxGravity {
			g = maxGravity
		}
		gx += -relPos.X / distEarth * g
		gy += -relPos.Y / distEarth * g
		gz += -relPos.Z / distEarth * g
	}
	return
}

// consumeThrust — расход топлива и вектор ускорения от тяги.
func consumeThrust(r *Rocket, dt float32) (tx, ty, tz float32) {
	if r.Thrust == 0 || r.Fuel <= 0 {
		return
	}
	absThrust := r.Thrust
	if absThrust < 0 {
		absThrust = -absThrust
	}
	if !r.InfiniteFuel {
		r.fuelAccum += rocketFuelBurn * dt * absThrust
		for r.fuelAccum >= 1 && r.Fuel > 0 {
			r.Fuel--
			r.fuelAccum--
		}
	}
	accel := rocketThrustAccel * r.Thrust
	tx = r.Up.X * accel
	ty = r.Up.Y * accel
	tz = r.Up.Z * accel
	return
}

// applyAtmosphericDrag — торможение в атмосфере Земли.
func applyAtmosphericDrag(r *Rocket, ev protocol.Vector3, distEarth float32, dt float32) {
	alt := distEarth - float32(protocol.PlanetRadius)
	if alt <= 0 || alt >= atmosphereHeight {
		return
	}
	relVel := protocol.Vector3{X: r.Vel.X - ev.X, Y: r.Vel.Y - ev.Y, Z: r.Vel.Z - ev.Z}
	speedSq := relVel.X*relVel.X + relVel.Y*relVel.Y + relVel.Z*relVel.Z
	if speedSq <= 0.01 {
		return
	}
	density := float32(math.Exp(-float64(alt) / atmoScaleHeight))
	speed := float32(math.Sqrt(float64(speedSq)))
	dragMag := atmoDensityCoef * density * speedSq
	dv := dragMag / speed * dt
	if dv > speed {
		dv = speed
	}
	relVel.X -= relVel.X / speed * dv
	relVel.Y -= relVel.Y / speed * dv
	relVel.Z -= relVel.Z / speed * dv
	r.Vel.X = ev.X + relVel.X
	r.Vel.Y = ev.Y + relVel.Y
	r.Vel.Z = ev.Z + relVel.Z
}

// applySpeedLimit — мягкий потолок скорости относительно Земли.
func applySpeedLimit(r *Rocket, ev protocol.Vector3, isEarth bool) {
	if !isEarth {
		return
	}
	relVel := protocol.Vector3{X: r.Vel.X - ev.X, Y: r.Vel.Y - ev.Y, Z: r.Vel.Z - ev.Z}
	speedSq := relVel.X*relVel.X + relVel.Y*relVel.Y + relVel.Z*relVel.Z
	if speedSq <= maxSpeedSoft*maxSpeedSoft {
		return
	}
	speed := float32(math.Sqrt(float64(speedSq)))
	k := maxSpeedSoft / speed
	relVel.X *= k
	relVel.Y *= k
	relVel.Z *= k
	r.Vel.X = ev.X + relVel.X
	r.Vel.Y = ev.Y + relVel.Y
	r.Vel.Z = ev.Z + relVel.Z
}

// collideWithEarth — если ракета воткнулась в Землю, ставим на поверхность и гасим скорость.
func collideWithEarth(r *Rocket, ep, ev protocol.Vector3) {
	relVel := protocol.Vector3{X: r.Vel.X - ev.X, Y: r.Vel.Y - ev.Y, Z: r.Vel.Z - ev.Z}
	newRel := protocol.Vector3{X: r.Pos.X - ep.X, Y: r.Pos.Y - ep.Y, Z: r.Pos.Z - ep.Z}
	newDist := float32(math.Sqrt(float64(
		newRel.X*newRel.X + newRel.Y*newRel.Y + newRel.Z*newRel.Z)))
	surfR := protocol.SurfaceRadius(newRel)
	if newDist >= surfR+1.5 {
		return
	}
	dot := newRel.X*relVel.X + newRel.Y*relVel.Y + newRel.Z*relVel.Z
	if dot >= 0 && newDist >= surfR {
		return
	}
	scale := (surfR + 1.5) / newDist
	newRel.X *= scale
	newRel.Y *= scale
	newRel.Z *= scale
	r.Pos = protocol.Vector3{X: ep.X + newRel.X, Y: ep.Y + newRel.Y, Z: ep.Z + newRel.Z}
	r.Vel = ev
}

// collideWithSun — если ракета воткнулась в Солнце, ставим на поверхность.
func collideWithSun(s *Server, r *Rocket) {
	sdx := r.Pos.X - protocol.SunPos.X
	sdy := r.Pos.Y - protocol.SunPos.Y
	sdz := r.Pos.Z - protocol.SunPos.Z
	sunDistSq := sdx*sdx + sdy*sdy + sdz*sdz
	sunClampR := protocol.SunRadius + 1.5
	if sunDistSq >= sunClampR*sunClampR {
		return
	}
	sunDist := float32(math.Sqrt(float64(sunDistSq)))
	if sunDist < 0.01 {
		sunDist = 0.01
	}
	dotSun := r.Vel.X*sdx + r.Vel.Y*sdy + r.Vel.Z*sdz
	if dotSun >= 0 && sunDist >= protocol.SunRadius {
		return
	}
	scale := sunClampR / sunDist
	r.Pos.X = protocol.SunPos.X + sdx*scale
	r.Pos.Y = protocol.SunPos.Y + sdy*scale
	r.Pos.Z = protocol.SunPos.Z + sdz*scale
	r.Vel = protocol.Vector3{}
	r.PrimaryBody = "sun"
	s.log.Info("rocket landed on sun", "id", r.ID)
}

// updateRocketFrame — LocalPos, PrimaryBody, DistSun, SOIRadius.
func updateRocketFrame(r *Rocket, ep protocol.Vector3, isEarth bool, distSun float32) {
	if isEarth {
		r.LocalPos = protocol.Vector3{X: r.Pos.X - ep.X, Y: r.Pos.Y - ep.Y, Z: r.Pos.Z - ep.Z}
		r.PrimaryBody = "earth"
	} else {
		r.LocalPos = protocol.Vector3{
			X: r.Pos.X - protocol.SunPos.X,
			Y: r.Pos.Y - protocol.SunPos.Y,
			Z: r.Pos.Z - protocol.SunPos.Z,
		}
		r.PrimaryBody = "sun"
	}
	r.DistSun = distSun
	r.SOIRadius = protocol.SOIEarth
}

// logRocketTick — раз в секунду пишет состояние ракеты в лог.
func logRocketTick(s *Server, r *Rocket, distEarth, distSun float32) {
	if time.Since(r.lastLogAt) <= time.Second {
		return
	}
	r.lastLogAt = time.Now()
	s.log.Info("rocket tick",
		"id", r.ID, "body", r.PrimaryBody,
		"alt", r.Altitude, "speed", r.Speed,
		"distEarth", distEarth, "distSun", distSun)
}
