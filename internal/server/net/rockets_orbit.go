package net

import (
	"math"

	"github.com/spacedreamer99/lodeen/internal/shared/protocol"
)

// orbitalParams — apoapsis и periapsis через удельную орбитальную энергию.
// Возвращает высоты над поверхностью (может быть отрицательной для periapsis
// если траектория пересекает планету). Для гиперболических орбит возвращает (0, 0).
func orbitalParams(pos, vel, primaryPos protocol.Vector3, mu, bodyRadius float32) (apo, peri, speed, alt, targetV float32) {
	rx := pos.X - primaryPos.X
	ry := pos.Y - primaryPos.Y
	rz := pos.Z - primaryPos.Z
	r2 := rx*rx + ry*ry + rz*rz
	r := float32(math.Sqrt(float64(r2)))
	v2 := vel.X*vel.X + vel.Y*vel.Y + vel.Z*vel.Z
	speed = float32(math.Sqrt(float64(v2)))
	alt = r - bodyRadius

	// Целевая скорость круговой орбиты на текущей высоте.
	if r > 1 {
		targetV = float32(math.Sqrt(float64(mu / r)))
	}

	if r < 1 {
		return 0, 0, speed, alt, targetV
	}

	E := v2/2 - mu/r
	if E >= 0 {
		// параболическая или гиперболическая — орбиты нет
		return 0, 0, speed, alt, targetV
	}

	a := -mu / (2 * E)
	rv := rx*vel.X + ry*vel.Y + rz*vel.Z
	k := v2 - mu/r
	ex := (k*rx - rv*vel.X) / mu
	ey := (k*ry - rv*vel.Y) / mu
	ez := (k*rz - rv*vel.Z) / mu
	e := float32(math.Sqrt(float64(ex*ex + ey*ey + ez*ez)))

	apo = a*(1+e) - bodyRadius
	peri = a*(1-e) - bodyRadius
	return
}

// updateOrbitalParams — считает apo/peri/speed/alt/targetV и флаг стабильной орбиты.
func updateOrbitalParams(s *Server, r *Rocket, ep protocol.Vector3, isEarth bool) {
	var primary protocol.Vector3
	var mu, bodyR float32
	if isEarth {
		primary = ep
		mu = protocol.MuEarth
		bodyR = protocol.PlanetRadius
	} else {
		primary = protocol.SunPos
		mu = protocol.MuSun
		bodyR = protocol.SunRadius
	}
	apo, peri, spd, alt, tv := orbitalParams(r.Pos, r.Vel, primary, mu, bodyR)
	r.Apoapsis = apo
	r.Periapsis = peri
	r.Speed = spd
	r.Altitude = alt
	r.TargetVelocity = tv

	var stable bool
	if isEarth {
		stable = peri > atmosphereHeight && alt > atmosphereHeight
	} else {
		stable = peri > 0
	}
	if stable && !r.InOrbit {
		r.InOrbit = true
		s.log.Info("rocket reached stable orbit",
			"id", r.ID, "body", r.PrimaryBody,
			"apo", apo, "peri", peri, "speed", spd, "alt", alt)
	} else if !stable {
		r.InOrbit = false
	}
}
