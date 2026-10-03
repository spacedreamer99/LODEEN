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
	s.rockets.Lock()
	s.rockets.Map()[id] = &Rocket{
		ID:       id,
		Pos:      protocol.Vector3{X: ep.X + localPos.X, Y: ep.Y + localPos.Y, Z: ep.Z + localPos.Z},
		Vel:      ev,
		Up:       up,
		Fuel:     rocketMaxFuel,
		MaxFuel:  rocketMaxFuel,
		LocalPos: localPos,
	}
	s.rockets.Unlock()

	c.mu.Lock()
	inv := make(map[string]int, len(c.inventory))
	for k, v := range c.inventory {
		inv[k] = v
	}
	c.mu.Unlock()
	c.sendEnvelope(protocol.TypeInventoryUpdate, protocol.InventoryUpdate{Items: inv})
	c.log.Info("rocket placed", "id", id,
		"local", localPos,
		"helio", s.rockets.Map()[id].Pos,
		"dist", l)
}

func (s *Server) handleBoardRocket(c *Client, rocketID string) {
	ps := c.State()

	s.rockets.Lock()
	r, ok := s.rockets.Map()[rocketID]
	if !ok {
		s.rockets.Unlock()
		c.log.Warn("rocket: not found")
		return
	}
	if r.Piloted && r.OwnerID != c.ID {
		s.rockets.Unlock()
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
		s.rockets.Unlock()
		c.log.Warn("rocket: too far",
			"r_geo", []float32{rx, ry, rz},
			"player", []float32{ps.X, ps.Y, ps.Z},
			"d2", dx*dx+dy*dy+dz*dz)
		return
	}
	r.Piloted = true
	r.OwnerID = c.ID
	s.rockets.Unlock()
	c.log.Info("rocket boarded", "id", rocketID)
}

func (s *Server) handleExitRocket(c *Client) {
	s.rockets.Lock()
	defer s.rockets.Unlock()
	for _, r := range s.rockets.Map() {
		if r.OwnerID == c.ID && r.Piloted {
			r.Piloted = false
			r.OwnerID = ""
			c.log.Info("rocket exited", "id", r.ID)
			return
		}
	}
}

func (s *Server) handleRocketInput(c *Client, p protocol.RocketInput) {
	s.rockets.Lock()
	defer s.rockets.Unlock()
	for _, r := range s.rockets.Map() {
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

// tickRockets — оркестратор: обходит все ракеты и вызывает подходящий тик.
func (s *Server) tickRockets(dt float32) {
	s.rockets.Lock()
	defer s.rockets.Unlock()

	ep := s.world.EarthPos
	ev := s.world.EarthVel

	for _, r := range s.rockets.Map() {
		if !r.Piloted {
			tickFreeRocket(r, ep, ev)
			continue
		}
		s.tickPilotedRocket(r, ep, ev, dt)
	}
}

// tickFreeRocket — непривязанная ракета «едет» вместе с телом (Землёй или Солнцем).
func tickFreeRocket(r *Rocket, ep, ev protocol.Vector3) {
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
}
