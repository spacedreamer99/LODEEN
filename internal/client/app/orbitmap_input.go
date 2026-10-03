package app

import (
	"math"

	rl "github.com/gen2brain/raylib-go/raylib"
)

func (a *App) updateOrbitMapInput() {
	md := rl.GetMouseDelta()

	if rl.IsMouseButtonPressed(rl.MouseLeftButton) {
		a.orbitDragged = false
		a.orbitMouseStartX = float32(rl.GetMouseX())
		a.orbitMouseStartY = float32(rl.GetMouseY())
	}
	if rl.IsMouseButtonDown(rl.MouseLeftButton) {
		if md.X*md.X+md.Y*md.Y > 4 {
			a.orbitDragged = true
		}
		if a.orbitDragged {
			a.orbitAzimuth += md.X * 0.008
			a.orbitElevation += md.Y * 0.008
			if a.orbitElevation < -1.4 {
				a.orbitElevation = -1.4
			}
			if a.orbitElevation > 1.4 {
				a.orbitElevation = 1.4
			}
		}
	}
	if rl.IsMouseButtonReleased(rl.MouseLeftButton) && !a.orbitDragged {
		mx := rl.GetMouseX()
		my := rl.GetMouseY()
		a.handleOrbitMapClick(mx, my)
	}

	wheel := rl.GetMouseWheelMove()
	if wheel != 0 {
		mult := wheel * 0.25
		if rl.IsKeyDown(rl.KeyLeftShift) || rl.IsKeyDown(rl.KeyRightShift) {
			mult *= 4
		}
		a.orbitDistance *= 1.0 - mult
		if a.orbitDistance < 40 {
			a.orbitDistance = 40
		}
		if a.orbitDistance > 150000 {
			a.orbitDistance = 150000
		}
	}
	if rl.IsKeyPressed(rl.KeyF) {
		switch a.orbitFocus {
		case "":
			a.orbitFocus = "earth"
		case "earth":
			a.orbitFocus = "sun"
		case "sun":
			a.orbitFocus = "star2"
		case "star2":
			a.orbitFocus = "planet2"
		case "planet2":
			a.orbitFocus = "rocket"
		default:
			a.orbitFocus = ""
		}
	}
}

func (a *App) handleOrbitMapClick(mx, my int32) {
	mfx := float32(mx)
	mfy := float32(my)
	// Ракета — приоритет.
	dx := mfx - a.rocketScrX
	dy := mfy - a.rocketScrY
	if dx*dx+dy*dy < 22*22 {
		a.orbitFocus = "rocket"
		return
	}
	// Земля.
	dx = mfx - a.earthScrX
	dy = mfy - a.earthScrY
	rr := a.earthScrR + 12
	if rr < 24 {
		rr = 24
	}
	if dx*dx+dy*dy < rr*rr {
		a.orbitFocus = "earth"
		return
	}
	// Солнце.
	dx = mfx - a.sunScrX
	dy = mfy - a.sunScrY
	if dx*dx+dy*dy < 40*40 {
		a.orbitFocus = "sun"
		return
	}
	// Star2.
	dx = mfx - a.star2ScrX
	dy = mfy - a.star2ScrY
	if dx*dx+dy*dy < 40*40 {
		a.orbitFocus = "star2"
		return
	}
	// Planet2.
	dx = mfx - a.planet2ScrX
	dy = mfy - a.planet2ScrY
	if dx*dx+dy*dy < 30*30 {
		a.orbitFocus = "planet2"
		return
	}
	// Пустое место — авто.
	a.orbitFocus = ""
}

func rotateAroundAxis(v, axis rl.Vector3, angle float32) rl.Vector3 {
	cosA := float32(math.Cos(float64(angle)))
	sinA := float32(math.Sin(float64(angle)))
	dot := v.X*axis.X + v.Y*axis.Y + v.Z*axis.Z
	cx := axis.Y*v.Z - axis.Z*v.Y
	cy := axis.Z*v.X - axis.X*v.Z
	cz := axis.X*v.Y - axis.Y*v.X
	return rl.NewVector3(
		v.X*cosA+cx*sinA+axis.X*dot*(1-cosA),
		v.Y*cosA+cy*sinA+axis.Y*dot*(1-cosA),
		v.Z*cosA+cz*sinA+axis.Z*dot*(1-cosA),
	)
}
