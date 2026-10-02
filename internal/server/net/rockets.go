package net

import (
	"math"
	"time"

	"github.com/spacedreamer99/lodeen/internal/shared/protocol"
)

// Rocket — летающий объект. Ставится как house, садится ЛКМ пустой рукой.
type Rocket struct {
	ID             string
	Pos            protocol.Vector3
	Vel            protocol.Vector3
	Up             protocol.Vector3 // нормаль к поверхности в момент старта
	Yaw            float32
	Fuel           int
	MaxFuel        int
	Piloted        bool
	OwnerID        string
	InOrbit        bool
	Thrust         float32 // -1..+1
	lastLogAt      time.Time
	fuelAccum      float32
	TargetUp       protocol.Vector3
	Apoapsis       float32
	Periapsis      float32
	Speed          float32
	Altitude       float32
	TargetVelocity float32
	InfiniteFuel   bool
	AutoPilot      string
	PrimaryBody    string
	DistSun        float32
	SOIRadius      float32
	LocalPos       protocol.Vector3 // координаты относительно Земли
}

const (
	rocketMaxFuel     = 100
	rocketThrustAccel = 50.0 // чуть выше g=40 — медленный контролируемый подъём
	rocketFuelBurn    = 5.0
	atmosphereHeight  = 50.0
	atmoDensityCoef   = 0.02
	atmoScaleHeight   = 12.0
	steerRate         = 3.0
	maxSpeedSoft      = 100.0
	maxGravity        = 500.0
)

func (s *Server) handlePlaceRocket(c *Client, p protocol.PlaceRocket) {
	if !c.consumeItem("rocket") {
		return
	}
	// Клиент шлёт geo-координаты (Земля в 0,0,0). Сервер работает в helio.
	localPos := protocol.Vector3{X: p.X, Y: p.Y, Z: p.Z}
	localPos = protocol.ClampToSurface(localPos)

	l := float32(math.Sqrt(float64(
		localPos.X*localPos.X + localPos.Y*localPos.Y + localPos.Z*localPos.Z)))
	if l < 0.01 {
		return
	}
	up := protocol.Vector3{X: localPos.X / l, Y: localPos.Y / l, Z: localPos.Z / l}

	ep := s.world.EarthPos
	ev := s.world.EarthVel

	id := newID()
	s.rocketsMu.Lock()
	s.rockets[id] = &Rocket{
		ID:       id,
		Pos:      protocol.Vector3{X: ep.X + localPos.X, Y: ep.Y + localPos.Y, Z: ep.Z + localPos.Z},
		Vel:      ev,
		Up:       up,
		Fuel:     rocketMaxFuel,
		MaxFuel:  rocketMaxFuel,
		LocalPos: localPos,
	}
	s.rocketsMu.Unlock()

	c.mu.Lock()
	inv := make(map[string]int, len(c.inventory))
	for k, v := range c.inventory {
		inv[k] = v
	}
	c.mu.Unlock()
	c.sendEnvelope(protocol.TypeInventoryUpdate, protocol.InventoryUpdate{Items: inv})
	c.log.Info("rocket placed", "id", id,
		"local", localPos,
		"helio", s.rockets[id].Pos,
		"dist", l)
}

func (s *Server) handleBoardRocket(c *Client, rocketID string) {
	ps := c.State()

	s.rocketsMu.Lock()
	r, ok := s.rockets[rocketID]
	if !ok {
		s.rocketsMu.Unlock()
		c.log.Warn("rocket: not found")
		return
	}
	if r.Piloted && r.OwnerID != c.ID {
		s.rocketsMu.Unlock()
		c.log.Warn("rocket: already piloted", "owner", r.OwnerID)
		return
	}
	// Игрок в geo-фрейме (относительно Земли), ракета в helio.
	// Переводим ракету в geo для проверки дистанции.
	ep := s.world.EarthPos
	rx := r.Pos.X - ep.X
	ry := r.Pos.Y - ep.Y
	rz := r.Pos.Z - ep.Z
	dx := rx - ps.X
	dy := ry - ps.Y
	dz := rz - ps.Z
	if dx*dx+dy*dy+dz*dz > 8.0*8.0 {
		s.rocketsMu.Unlock()
		c.log.Warn("rocket: too far",
			"r_geo", []float32{rx, ry, rz},
			"player", []float32{ps.X, ps.Y, ps.Z},
			"d2", dx*dx+dy*dy+dz*dz)
		return
	}
	r.Piloted = true
	r.OwnerID = c.ID
	s.rocketsMu.Unlock()
	c.log.Info("rocket boarded", "id", rocketID)
}

func (s *Server) handleExitRocket(c *Client) {
	s.rocketsMu.Lock()
	defer s.rocketsMu.Unlock()
	for _, r := range s.rockets {
		if r.OwnerID == c.ID && r.Piloted {
			r.Piloted = false
			r.OwnerID = ""
			c.log.Info("rocket exited", "id", r.ID)
			return
		}
	}
}

func (s *Server) handleRocketInput(c *Client, p protocol.RocketInput) {
	s.rocketsMu.Lock()
	defer s.rocketsMu.Unlock()
	for _, r := range s.rockets {
		if r.OwnerID == c.ID && r.Piloted {
			r.Thrust = p.Thrust
			r.AutoPilot = p.AutoPilot
			// Если автопилот выключен — берём TargetUp с клиента (мышь).
			if p.AutoPilot == "" {
				r.TargetUp = protocol.Vector3{X: p.TargetUpX, Y: p.TargetUpY, Z: p.TargetUpZ}
			}
			c.mu.RLock()
			r.InfiniteFuel = c.infiniteFuel
			c.mu.RUnlock()
			if r.InfiniteFuel {
				r.Fuel = r.MaxFuel
			}
			return
		}
	}
}

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

func (s *Server) tickRockets(dt float32) {
	s.rocketsMu.Lock()
	defer s.rocketsMu.Unlock()

	ep := s.world.EarthPos
	ev := s.world.EarthVel

	for _, r := range s.rockets {
		// ── Непривязанная ракета: «едет» с Землёй ──
		if !r.Piloted {
			var bodyPos, bodyVel protocol.Vector3
			if r.PrimaryBody == "sun" {
				bodyPos = protocol.SunPos
				bodyVel = protocol.Vector3{}
				lp := r.LocalPos
				llen := float32(math.Sqrt(float64(lp.X*lp.X + lp.Y*lp.Y + lp.Z*lp.Z)))
				if llen < 0.01 {
					llen = 0.01
					lp = protocol.Vector3{X: protocol.SunRadius, Y: 0, Z: 0}
				}
				target := protocol.SunRadius + 1.5
				if llen < target {
					scale := target / llen
					lp.X *= scale
					lp.Y *= scale
					lp.Z *= scale
				}
				r.LocalPos = lp
				r.Up = protocol.Vector3{X: lp.X / llen, Y: lp.Y / llen, Z: lp.Z / llen}
			} else {
				bodyPos = ep
				bodyVel = ev
				r.LocalPos = protocol.ClampToSurface(r.LocalPos)
				llen := float32(math.Sqrt(float64(
					r.LocalPos.X*r.LocalPos.X + r.LocalPos.Y*r.LocalPos.Y + r.LocalPos.Z*r.LocalPos.Z)))
				if llen > 0.01 {
					r.Up = protocol.Vector3{X: r.LocalPos.X / llen, Y: r.LocalPos.Y / llen, Z: r.LocalPos.Z / llen}
				}
			}
			r.Pos = protocol.Vector3{X: bodyPos.X + r.LocalPos.X, Y: bodyPos.Y + r.LocalPos.Y, Z: bodyPos.Z + r.LocalPos.Z}
			r.Vel = bodyVel
			r.Thrust = 0
			continue
		}

		// ── Векторы относительно Земли ──
		relPos := protocol.Vector3{X: r.Pos.X - ep.X, Y: r.Pos.Y - ep.Y, Z: r.Pos.Z - ep.Z}
		relVel := protocol.Vector3{X: r.Vel.X - ev.X, Y: r.Vel.Y - ev.Y, Z: r.Vel.Z - ev.Z}
		distEarth := float32(math.Sqrt(float64(
			relPos.X*relPos.X + relPos.Y*relPos.Y + relPos.Z*relPos.Z)))

		sx := r.Pos.X - protocol.SunPos.X
		sy := r.Pos.Y - protocol.SunPos.Y
		sz := r.Pos.Z - protocol.SunPos.Z
		distSun := float32(math.Sqrt(float64(sx*sx + sy*sy + sz*sz)))

		isEarth := distEarth < protocol.SOIEarth

		// Автопилот — TargetUp.
		if r.AutoPilot != "" {
			var target protocol.Vector3
			vLen := float32(math.Sqrt(float64(
				relVel.X*relVel.X + relVel.Y*relVel.Y + relVel.Z*relVel.Z)))
			rLen := distEarth

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

		// Плавный поворот Up к TargetUp.
		if r.TargetUp.X != 0 || r.TargetUp.Y != 0 || r.TargetUp.Z != 0 {
			tLen := float32(math.Sqrt(float64(
				r.TargetUp.X*r.TargetUp.X + r.TargetUp.Y*r.TargetUp.Y + r.TargetUp.Z*r.TargetUp.Z)))
			if tLen > 0.01 {
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
		}

		// ── Гравитация: Солнце всегда, Земля если в SOI ──
		var gx, gy, gz float32

		// Солнце.
		if distSun > 1 {
			g := protocol.MuSun / (distSun * distSun)
			gx += -sx / distSun * g
			gy += -sy / distSun * g
			gz += -sz / distSun * g
		}

		// Земля.
		if isEarth && distEarth > 1 {
			g := protocol.MuEarth / (distEarth * distEarth)
			if g > maxGravity {
				g = maxGravity
			}
			gx += -relPos.X / distEarth * g
			gy += -relPos.Y / distEarth * g
			gz += -relPos.Z / distEarth * g
		}

		// ── Тяга ──
		var tx, ty, tz float32
		if r.Thrust != 0 && r.Fuel > 0 {
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
		}

		// ── Интеграция скорости ──
		r.Vel.X += (gx + tx) * dt
		r.Vel.Y += (gy + ty) * dt
		r.Vel.Z += (gz + tz) * dt

		// ── Атмосферное сопротивление (только Земля) ──
		if isEarth {
			alt := distEarth - float32(protocol.PlanetRadius)
			if alt > 0 && alt < atmosphereHeight {
				// Обновляем relVel после гравитации.
				relVel.X = r.Vel.X - ev.X
				relVel.Y = r.Vel.Y - ev.Y
				relVel.Z = r.Vel.Z - ev.Z
				speedSq := relVel.X*relVel.X + relVel.Y*relVel.Y + relVel.Z*relVel.Z
				if speedSq > 0.01 {
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
			}
		}

		// ── Лимит скорости (относительно Земли) ──
		relVel.X = r.Vel.X - ev.X
		relVel.Y = r.Vel.Y - ev.Y
		relVel.Z = r.Vel.Z - ev.Z
		speedSq := relVel.X*relVel.X + relVel.Y*relVel.Y + relVel.Z*relVel.Z
		if isEarth && speedSq > maxSpeedSoft*maxSpeedSoft {
			speed := float32(math.Sqrt(float64(speedSq)))
			k := maxSpeedSoft / speed
			relVel.X *= k
			relVel.Y *= k
			relVel.Z *= k
			r.Vel.X = ev.X + relVel.X
			r.Vel.Y = ev.Y + relVel.Y
			r.Vel.Z = ev.Z + relVel.Z
		}

		// ── Интеграция позиции ──
		r.Pos.X += r.Vel.X * dt
		r.Pos.Y += r.Vel.Y * dt
		r.Pos.Z += r.Vel.Z * dt

		// ── Коллизия с Землёй ──
		if isEarth {
			newRel := protocol.Vector3{X: r.Pos.X - ep.X, Y: r.Pos.Y - ep.Y, Z: r.Pos.Z - ep.Z}
			newDist := float32(math.Sqrt(float64(
				newRel.X*newRel.X + newRel.Y*newRel.Y + newRel.Z*newRel.Z)))
			surfR := protocol.SurfaceRadius(newRel)
			if newDist < surfR+1.5 {
				dot := newRel.X*relVel.X + newRel.Y*relVel.Y + newRel.Z*relVel.Z
				if dot < 0 || newDist < surfR {
					scale := (surfR + 1.5) / newDist
					newRel.X *= scale
					newRel.Y *= scale
					newRel.Z *= scale
					r.Pos = protocol.Vector3{X: ep.X + newRel.X, Y: ep.Y + newRel.Y, Z: ep.Z + newRel.Z}
					r.Vel = ev
				}
			}
		}

		// Обновляем LocalPos и Primary.
		if isEarth {
			r.LocalPos = protocol.Vector3{X: r.Pos.X - ep.X, Y: r.Pos.Y - ep.Y, Z: r.Pos.Z - ep.Z}
		} else {
			r.LocalPos = protocol.Vector3{
				X: r.Pos.X - protocol.SunPos.X,
				Y: r.Pos.Y - protocol.SunPos.Y,
				Z: r.Pos.Z - protocol.SunPos.Z,
			}
		}
		r.DistSun = distSun
		r.SOIRadius = protocol.SOIEarth
		if isEarth {
			r.PrimaryBody = "earth"
		} else {
			r.PrimaryBody = "sun"
		}

		// ── Коллизия с Солнцем ──
		sdx := r.Pos.X - protocol.SunPos.X
		sdy := r.Pos.Y - protocol.SunPos.Y
		sdz := r.Pos.Z - protocol.SunPos.Z
		sunDistSq := sdx*sdx + sdy*sdy + sdz*sdz
		sunClampR := protocol.SunRadius + 1.5
		if sunDistSq < sunClampR*sunClampR {
			sunDist2 := float32(math.Sqrt(float64(sunDistSq)))
			if sunDist2 < 0.01 {
				sunDist2 = 0.01
			}
			dotSun := r.Vel.X*sdx + r.Vel.Y*sdy + r.Vel.Z*sdz
			if dotSun < 0 || sunDist2 < protocol.SunRadius {
				scale := sunClampR / sunDist2
				r.Pos.X = protocol.SunPos.X + sdx*scale
				r.Pos.Y = protocol.SunPos.Y + sdy*scale
				r.Pos.Z = protocol.SunPos.Z + sdz*scale
				r.Vel = protocol.Vector3{}
				r.PrimaryBody = "sun"
				s.log.Info("rocket landed on sun", "id", r.ID)
			}
		}

		// ── Орбитальные параметры ──
		if r.Piloted {
			var primary protocol.Vector3
			var mu, bodyR float32
			var oPos, oVel protocol.Vector3
			if isEarth {
				primary = ep
				mu = protocol.MuEarth
				bodyR = protocol.PlanetRadius
				oPos = r.Pos
				oVel = r.Vel
			} else {
				primary = protocol.SunPos
				mu = protocol.MuSun
				bodyR = protocol.SunRadius
				oPos = r.Pos
				oVel = r.Vel
			}
			apo, peri, spd, alt, tv := orbitalParams(oPos, oVel, primary, mu, bodyR)
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

		if r.Piloted && time.Since(r.lastLogAt) > time.Second {
			r.lastLogAt = time.Now()
			s.log.Info("rocket tick",
				"id", r.ID, "body", r.PrimaryBody,
				"alt", r.Altitude, "speed", r.Speed,
				"distEarth", distEarth, "distSun", distSun)
		}
	}
}

var _ = time.Now
