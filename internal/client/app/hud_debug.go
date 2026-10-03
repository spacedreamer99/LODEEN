package app

import (
	"fmt"
	"math"

	rl "github.com/gen2brain/raylib-go/raylib"

	"github.com/spacedreamer99/lodeen/internal/client/fonts"
	"github.com/spacedreamer99/lodeen/internal/client/input"
	"github.com/spacedreamer99/lodeen/internal/shared/protocol"
)

func (a *App) drawDebugOverlay() {
	if !a.diag.showDebug || a.flight == nil {
		return
	}
	sw := int32(rl.GetScreenWidth())
	x := sw - 340
	y := int32(80)
	col := rl.NewColor(200, 220, 255, 220)
	colDim := rl.NewColor(140, 160, 200, 180)

	fonts.Draw("F3 DEBUG", x, y, 16, rl.NewColor(255, 200, 60, 255))
	y += 22

	helioStr := fmt.Sprintf("HELIO %.1f %.1f %.1f",
		a.flight.Pos.X, a.flight.Pos.Y, a.flight.Pos.Z)
	fonts.Draw(helioStr, x, y, 14, col)
	y += 18

	relX := a.flight.Pos.X - a.world.earthPos.X
	relY := a.flight.Pos.Y - a.world.earthPos.Y
	relZ := a.flight.Pos.Z - a.world.earthPos.Z
	relDist := float32(math.Sqrt(float64(relX*relX + relY*relY + relZ*relZ)))
	geoStr := fmt.Sprintf("GEO  %.1f %.1f %.1f  |r|=%.2f",
		relX, relY, relZ, relDist)
	fonts.Draw(geoStr, x, y, 14, col)
	y += 18

	velLen := float32(math.Sqrt(float64(
		a.flight.Vel.X*a.flight.Vel.X +
			a.flight.Vel.Y*a.flight.Vel.Y +
			a.flight.Vel.Z*a.flight.Vel.Z)))
	velStr := fmt.Sprintf("VEL  %.2f (%.2f %.2f %.2f)",
		velLen, a.flight.Vel.X, a.flight.Vel.Y, a.flight.Vel.Z)
	fonts.Draw(velStr, x, y, 14, col)
	y += 18

	surfaceR := protocol.SurfaceRadius(protocol.Vector3{X: relX, Y: relY, Z: relZ})
	minR := surfaceR + protocol.PlayerHeight
	onGround := relDist <= minR+0.5
	stateStr := "AIRBORNE"
	stateCol := rl.NewColor(240, 180, 100, 240)
	if onGround {
		stateStr = "ON GROUND"
		stateCol = rl.NewColor(100, 240, 120, 240)
	}
	stateFull := fmt.Sprintf("%s  minR=%.2f", stateStr, minR)
	fonts.Draw(stateFull, x, y, 14, stateCol)
	y += 18

	modeStr := "SURVIVAL"
	if a.flight.Mode == input.ModeCreative {
		modeStr = "CREATIVE"
		if a.flight.AttachedBody != "" {
			modeStr += "  ATTACHED:" + a.flight.AttachedBody
		}
	}
	fonts.Draw(modeStr, x, y, 14, colDim)
	y += 18

	earthStr := fmt.Sprintf("EARTH %.1f %.1f %.1f",
		a.world.earthPos.X, a.world.earthPos.Y, a.world.earthPos.Z)
	fonts.Draw(earthStr, x, y, 14, colDim)
	y += 18
	earthVelStr := fmt.Sprintf("EVEL %.2f %.2f %.2f",
		a.world.earthVel.X, a.world.earthVel.Y, a.world.earthVel.Z)
	fonts.Draw(earthVelStr, x, y, 14, colDim)
}

func (a *App) drawMiniStatus() {
	if a.flight == nil {
		return
	}

	// Позиция относительно текущего тела.
	bodyName := "EARTH"
	var bodyPos protocol.Vector3
	bodyRadius := protocol.PlanetRadius
	if a.flight.AttachedBody == "sun" {
		bodyName = "SUN"
		bodyPos = protocol.SunPos
		bodyRadius = protocol.SunRadius
	} else {
		bodyPos = a.world.earthPos
	}

	relX := a.flight.Pos.X - bodyPos.X
	relY := a.flight.Pos.Y - bodyPos.Y
	relZ := a.flight.Pos.Z - bodyPos.Z
	relDist := float32(math.Sqrt(float64(relX*relX + relY*relY + relZ*relZ)))

	surfaceR := bodyRadius
	if bodyName == "EARTH" {
		surfaceR = protocol.SurfaceRadius(protocol.Vector3{X: relX, Y: relY, Z: relZ})
	}
	alt := relDist - surfaceR

	// Скорость относительно тела.
	var bodyVel protocol.Vector3
	if bodyName == "EARTH" {
		bodyVel = a.world.earthVel
	}
	relVx := a.flight.Vel.X - bodyVel.X
	relVy := a.flight.Vel.Y - bodyVel.Y
	relVz := a.flight.Vel.Z - bodyVel.Z
	relSpeed := float32(math.Sqrt(float64(relVx*relVx + relVy*relVy + relVz*relVz)))

	// Радиальная скорость (вверх/вниз).
	upX := relX / relDist
	upY := relY / relDist
	upZ := relZ / relDist
	radialVel := relVx*upX + relVy*upY + relVz*upZ

	onGround := alt <= protocol.PlayerHeight+0.5

	// Панель.
	x := int32(20)
	y := int32(120)
	pad := int32(10)
	panelW := int32(220)
	panelH := int32(70)

	bg := rl.NewColor(10, 15, 30, 200)
	rl.DrawRectangle(x, y, panelW, panelH, bg)
	rl.DrawRectangleLines(x, y, panelW, panelH, rl.NewColor(120, 180, 240, 255))

	// Строка 1: тело + статус.
	bodyCol := rl.NewColor(100, 180, 240, 255)
	if bodyName == "SUN" {
		bodyCol = rl.NewColor(240, 180, 60, 255)
	}
	fonts.Draw(bodyName, x+pad, y+pad, 16, bodyCol)

	statusStr := "AIRBORNE"
	statusCol := rl.NewColor(240, 180, 100, 255)
	if onGround {
		statusStr = "ON GROUND"
		statusCol = rl.NewColor(100, 240, 120, 255)
	}
	fonts.Draw(statusStr, x+pad+80, y+pad, 16, statusCol)

	// Строка 2: высота.
	altStr := fmt.Sprintf("ALT %.1f m", alt)
	fonts.Draw(altStr, x+pad, y+pad+22, 14, rl.RayWhite)

	// Строка 3: скорость.
	spdStr := fmt.Sprintf("SPD %.1f m/s", relSpeed)
	fonts.Draw(spdStr, x+pad, y+pad+42, 14, rl.RayWhite)

	// Если в воздухе — показать вертикальную скорость.
	if !onGround {
		vsStr := fmt.Sprintf("V/S %+.1f m/s", radialVel)
		vsCol := rl.NewColor(200, 200, 220, 220)
		if radialVel > 0.5 {
			vsCol = rl.NewColor(100, 240, 120, 240) // вверх
		} else if radialVel < -0.5 {
			vsCol = rl.NewColor(240, 120, 120, 240) // вниз
		}
		fonts.Draw(vsStr, x+pad+100, y+pad+42, 14, vsCol)
	}
}
