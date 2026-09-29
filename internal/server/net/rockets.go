package net

import (
	"math"
	"time"

	"github.com/spacedreamer99/lodeen/internal/shared/protocol"
)

// Rocket — летающий объект. Ставится как house, садится ЛКМ пустой рукой.
type Rocket struct {
	ID      string
	Pos     protocol.Vector3
	Vel     protocol.Vector3
	Up      protocol.Vector3 // нормаль к поверхности в момент старта
	Yaw     float32
	Fuel    int
	MaxFuel int
	Piloted bool
	OwnerID string
	InOrbit bool
	Thrust  float32 // -1..+1
	lastLogAt time.Time
	fuelAccum float32
	TargetUp protocol.Vector3
	Apoapsis  float32
	Periapsis float32
	Speed     float32
	Altitude  float32
	TargetVelocity float32
	InfiniteFuel bool
	AutoPilot string
}

const (
	rocketMaxFuel     = 100
	rocketThrustAccel = 50.0  // чуть выше g=40 — медленный контролируемый подъём
	rocketFuelBurn    = 5.0
	rocketGravity     = 40.0
	orbitHeight       = 150.0
	atmosphereHeight  = 50.0
	atmoDensityCoef   = 0.02
	atmoScaleHeight   = 12.0
	steerRate         = 3.0
	maxSpeedSoft      = 30.0  // жёстко, нельзя превысить v_esc
	maxGravity        = 500.0
)

func (s *Server) handlePlaceRocket(c *Client, p protocol.PlaceRocket) {
	if !c.consumeItem("rocket") {
		return
	}
	pos := protocol.Vector3{X: p.X, Y: p.Y, Z: p.Z}
	pos = protocol.ClampToSurface(pos)

	// Up — нормаль к планете в этой точке.
	l := float32(math.Sqrt(float64(pos.X*pos.X + pos.Y*pos.Y + pos.Z*pos.Z)))
	if l < 0.01 {
		return
	}
	up := protocol.Vector3{X: pos.X / l, Y: pos.Y / l, Z: pos.Z / l}

	id := newID()
	s.rocketsMu.Lock()
	s.rockets[id] = &Rocket{
		ID:      id,
		Pos:     pos,
		Up:      up,
		Fuel:    rocketMaxFuel,
		MaxFuel: rocketMaxFuel,
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
		"pos_x", pos.X, "pos_y", pos.Y, "pos_z", pos.Z,
		"up_x", up.X, "up_y", up.Y, "up_z", up.Z,
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
	dx := r.Pos.X - ps.X
	dy := r.Pos.Y - ps.Y
	dz := r.Pos.Z - ps.Z
	if dx*dx+dy*dy+dz*dz > 8.0*8.0 {
		s.rocketsMu.Unlock()
		c.log.Warn("rocket: too far")
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
func orbitalParams(pos, vel protocol.Vector3) (apo, peri, speed, alt, targetV float32) {
	r2 := pos.X*pos.X + pos.Y*pos.Y + pos.Z*pos.Z
	r := float32(math.Sqrt(float64(r2)))
	v2 := vel.X*vel.X + vel.Y*vel.Y + vel.Z*vel.Z
	speed = float32(math.Sqrt(float64(v2)))
	alt = r - protocol.PlanetRadius

	mu := float32(rocketGravity) * float32(protocol.PlanetRadius) * float32(protocol.PlanetRadius)

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
	rv := pos.X*vel.X + pos.Y*vel.Y + pos.Z*vel.Z
	k := v2 - mu/r
	ex := (k*pos.X - rv*vel.X) / mu
	ey := (k*pos.Y - rv*vel.Y) / mu
	ez := (k*pos.Z - rv*vel.Z) / mu
	e := float32(math.Sqrt(float64(ex*ex + ey*ey + ez*ez)))

	apo = a*(1+e) - protocol.PlanetRadius
	peri = a*(1-e) - protocol.PlanetRadius
	return
}

func (s *Server) tickRockets(dt float32) {
	s.rocketsMu.Lock()
	defer s.rocketsMu.Unlock()

	for _, r := range s.rockets {
		// Непривязанная ракета — стоит на поверхности, физика выключена.
		if !r.Piloted {
			r.Pos = protocol.ClampToSurface(r.Pos)
			l := float32(math.Sqrt(float64(r.Pos.X*r.Pos.X + r.Pos.Y*r.Pos.Y + r.Pos.Z*r.Pos.Z)))
			if l > 0.01 {
				r.Up = protocol.Vector3{X: r.Pos.X / l, Y: r.Pos.Y / l, Z: r.Pos.Z / l}
			}
			r.Vel = protocol.Vector3{}
			r.Thrust = 0
			continue
		}

		// Автопилот — вычисляет TargetUp по вектору скорости/позиции.
		if r.AutoPilot != "" {
			var target protocol.Vector3
			vLen := float32(math.Sqrt(float64(r.Vel.X*r.Vel.X + r.Vel.Y*r.Vel.Y + r.Vel.Z*r.Vel.Z)))
			rLen := float32(math.Sqrt(float64(r.Pos.X*r.Pos.X + r.Pos.Y*r.Pos.Y + r.Pos.Z*r.Pos.Z)))

			switch r.AutoPilot {
			case "prograde":
				if vLen > 0.1 {
					target = protocol.Vector3{X: r.Vel.X / vLen, Y: r.Vel.Y / vLen, Z: r.Vel.Z / vLen}
				}
			case "retrograde":
				if vLen > 0.1 {
					target = protocol.Vector3{X: -r.Vel.X / vLen, Y: -r.Vel.Y / vLen, Z: -r.Vel.Z / vLen}
				}
			case "radial_out":
				if rLen > 0.1 {
					target = protocol.Vector3{X: r.Pos.X / rLen, Y: r.Pos.Y / rLen, Z: r.Pos.Z / rLen}
				}
			case "radial_in":
				if rLen > 0.1 {
					target = protocol.Vector3{X: -r.Pos.X / rLen, Y: -r.Pos.Y / rLen, Z: -r.Pos.Z / rLen}
				}
			case "normal", "antinormal":
				// cross(Pos, Vel) — нормаль к орбитальной плоскости.
				nx := r.Pos.Y*r.Vel.Z - r.Pos.Z*r.Vel.Y
				ny := r.Pos.Z*r.Vel.X - r.Pos.X*r.Vel.Z
				nz := r.Pos.X*r.Vel.Y - r.Pos.Y*r.Vel.X
				nl := float32(math.Sqrt(float64(nx*nx + ny*ny + nz*nz)))
				if nl > 0.01 {
					k := float32(1.0)
					if r.AutoPilot == "antinormal" {
						k = -1.0
					}
					target = protocol.Vector3{X: nx / nl * k, Y: ny / nl * k, Z: nz / nl * k}
				}
			}

			// Если target получился — задаём как TargetUp.
			if target.X != 0 || target.Y != 0 || target.Z != 0 {
				r.TargetUp = target
			}
		}

		// Плавный поворот Up к TargetUp с ограниченной угловой скоростью.
		if r.TargetUp.X != 0 || r.TargetUp.Y != 0 || r.TargetUp.Z != 0 {
			tLen := float32(math.Sqrt(float64(r.TargetUp.X*r.TargetUp.X + r.TargetUp.Y*r.TargetUp.Y + r.TargetUp.Z*r.TargetUp.Z)))
			if tLen > 0.01 {
				tx := r.TargetUp.X / tLen
				ty := r.TargetUp.Y / tLen
				tz := r.TargetUp.Z / tLen
				// Угол между текущим Up и целевым.
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
					// Интерполируем с ограничением шага.
					k := maxAngle / angle
					r.Up.X += (tx - r.Up.X) * k
					r.Up.Y += (ty - r.Up.Y) * k
					r.Up.Z += (tz - r.Up.Z) * k
					// Нормализуем.
					upLen := float32(math.Sqrt(float64(r.Up.X*r.Up.X + r.Up.Y*r.Up.Y + r.Up.Z*r.Up.Z)))
					if upLen > 0.01 {
						r.Up.X /= upLen
						r.Up.Y /= upLen
						r.Up.Z /= upLen
					}
				} else {
					// Уже близко — доводим до цели.
					r.Up.X = tx
					r.Up.Y = ty
					r.Up.Z = tz
				}
			}
		}


		// Расстояние от центра планеты.
		dist := float32(math.Sqrt(float64(r.Pos.X*r.Pos.X + r.Pos.Y*r.Pos.Y + r.Pos.Z*r.Pos.Z)))
		if dist < 5 {
			dist = 5
		}

		// Гравитация (GM/r²).
		g := float32(rocketGravity) * float32(protocol.PlanetRadius) * float32(protocol.PlanetRadius) / (dist * dist)
		if g > maxGravity {
			g = maxGravity
		}
		invDist := 1.0 / dist
		gx := -r.Pos.X * invDist * g
		gy := -r.Pos.Y * invDist * g
		gz := -r.Pos.Z * invDist * g

		// Тяга (если пилотируется и есть топливо).
		var tx, ty, tz float32
		if r.Piloted && r.Thrust != 0 && r.Fuel > 0 {
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

		// Интеграция скорости.
		r.Vel.X += (gx + tx) * dt
		r.Vel.Y += (gy + ty) * dt
		r.Vel.Z += (gz + tz) * dt

		// Атмосферное сопротивление (экспоненциальная плотность).
		alt := dist - float32(protocol.PlanetRadius)
		if alt > 0 && alt < atmosphereHeight {
			density := float32(math.Exp(-float64(alt) / atmoScaleHeight))
			speedSq := r.Vel.X*r.Vel.X + r.Vel.Y*r.Vel.Y + r.Vel.Z*r.Vel.Z
			if speedSq > 0.01 {
				speed := float32(math.Sqrt(float64(speedSq)))
				dragMag := atmoDensityCoef * density * speedSq
				factor := 1.0 - (dragMag/speed)*dt
				if factor < 0 {
					factor = 0
				}
				r.Vel.X *= factor
				r.Vel.Y *= factor
				r.Vel.Z *= factor
			}
		}

		// Мягкий лимит скорости.
		speedSq := r.Vel.X*r.Vel.X + r.Vel.Y*r.Vel.Y + r.Vel.Z*r.Vel.Z
		if speedSq > maxSpeedSoft*maxSpeedSoft {
			speed := float32(math.Sqrt(float64(speedSq)))
			k := maxSpeedSoft / speed
			r.Vel.X *= k
			r.Vel.Y *= k
			r.Vel.Z *= k
		}

		// Интеграция позиции.
		r.Pos.X += r.Vel.X * dt
		r.Pos.Y += r.Vel.Y * dt
		r.Pos.Z += r.Vel.Z * dt

		// Расстояние от центра планеты после интеграции.
		newDist := float32(math.Sqrt(float64(r.Pos.X*r.Pos.X + r.Pos.Y*r.Pos.Y + r.Pos.Z*r.Pos.Z)))

		// Коллизия с поверхностью — только если летим внутрь планеты
		// или уже под поверхностью.
		surfR := protocol.SurfaceRadius(r.Pos)
		if newDist < surfR+1.5 {
			// dot < 0 — летим к центру, dot > 0 — от центра.
			dot := r.Vel.X*r.Pos.X + r.Vel.Y*r.Pos.Y + r.Vel.Z*r.Pos.Z
			if dot < 0 || newDist < surfR {
				scale := (surfR + 1.5) / newDist
				r.Pos.X *= scale
				r.Pos.Y *= scale
				r.Pos.Z *= scale
				r.Vel = protocol.Vector3{}
				newDist = surfR + 1.5
			}
		}

		// Орбитальные параметры (только для пилотируемых — экономим CPU).
		if r.Piloted {
			apo, peri, spd, alt, tv := orbitalParams(r.Pos, r.Vel)
			r.Apoapsis = apo
			r.Periapsis = peri
			r.Speed = spd
			r.Altitude = alt
			r.TargetVelocity = tv

			// Стабильная орбита: перигей выше атмосферы.
			stable := peri > atmosphereHeight && alt > atmosphereHeight
			if stable && !r.InOrbit {
				r.InOrbit = true
				s.log.Info("rocket reached stable orbit",
					"id", r.ID,
					"apo", apo, "peri", peri, "speed", spd, "alt", alt)
			} else if !stable {
				r.InOrbit = false
			}
		}

		// Debug — раз в секунду.
		if r.Piloted && time.Since(r.lastLogAt) > time.Second {
			r.lastLogAt = time.Now()
			s.log.Info("rocket tick",
				"id", r.ID,
				"alt", r.Altitude,
				"speed", r.Speed,
				"apo", r.Apoapsis,
				"peri", r.Periapsis,
				"in_orbit", r.InOrbit,
				"fuel", r.Fuel, "thrust", r.Thrust)
		}

		// Орбита — если высота превысила порог.
		if !r.InOrbit && newDist > float32(orbitHeight) {
			r.InOrbit = true
			s.log.Info("rocket reached orbit", "id", r.ID, "dist", newDist)
		}
	}
}

var _ = time.Now
