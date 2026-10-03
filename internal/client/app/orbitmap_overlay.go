package app

import (
	rl "github.com/gen2brain/raylib-go/raylib"

	"github.com/spacedreamer99/lodeen/internal/client/fonts"
	"github.com/spacedreamer99/lodeen/internal/shared/protocol"
)

// drawOrbitOverlays — все 2D-проекции тел поверх 3D (обход far plane).
func (a *App) drawOrbitOverlays(mapCam rl.Camera3D, fwd rl.Vector3, scale float32, sw, sh int32, haveRocket bool, r protocol.Rocket) {
	a.drawOrbitSunOverlay(mapCam, fwd, scale, sw, sh)
	a.drawOrbitStar2Overlay(mapCam, fwd, scale, sw, sh)
	a.drawOrbitPlanet2Overlay(mapCam, fwd, scale, sw, sh)
	a.drawOrbitEarthOverlay(mapCam, fwd, scale, sw, sh)
	if haveRocket {
		a.drawOrbitRocketOverlay(mapCam, fwd, scale, sw, sh, r)
	}
}

// inFront — точка перед камерой? GetWorldToScreen зеркалит точки за камерой.
func inFront(cam rl.Camera3D, world rl.Vector3, fwd rl.Vector3) bool {
	to := rl.Vector3Subtract(world, cam.Position)
	return fwd.X*to.X+fwd.Y*to.Y+fwd.Z*to.Z > 0
}

// onScreen — точка в пределах экрана (+запас).
func onScreen(p rl.Vector2, sw, sh int32, margin float32) bool {
	swF, shF := float32(sw), float32(sh)
	return p.X > -margin && p.X < swF+margin && p.Y > -margin && p.Y < shF+margin
}

func (a *App) drawOrbitSunOverlay(cam rl.Camera3D, fwd rl.Vector3, scale float32, sw, sh int32) {
	world := rl.NewVector3(protocol.SunPos.X/scale, protocol.SunPos.Y/scale, protocol.SunPos.Z/scale)
	if !inFront(cam, world, fwd) {
		return
	}
	scr := rl.GetWorldToScreen(world, cam)
	if !onScreen(scr, sw, sh, 200) {
		return
	}
	cx, cy := int32(scr.X), int32(scr.Y)
	rl.DrawCircle(cx, cy, 16, rl.NewColor(255, 220, 100, 255))
	rl.DrawCircleLines(cx, cy, 16, rl.NewColor(255, 180, 60, 255))
	rl.DrawCircleLines(cx, cy, 22, rl.NewColor(255, 220, 100, 140))
	rl.DrawCircleLines(cx, cy, 28, rl.NewColor(255, 220, 100, 80))
	fonts.Draw("SUN", cx+20, cy-10, 16, rl.NewColor(255, 220, 100, 255))
	a.orbit.sunScrX = float32(cx)
	a.orbit.sunScrY = float32(cy)
}

func (a *App) drawOrbitStar2Overlay(cam rl.Camera3D, fwd rl.Vector3, scale float32, sw, sh int32) {
	world := rl.NewVector3(protocol.Star2Pos.X/scale, protocol.Star2Pos.Y/scale, protocol.Star2Pos.Z/scale)
	scr := rl.GetWorldToScreen(world, cam)
	if !onScreen(scr, sw, sh, 200) {
		return
	}
	cx, cy := int32(scr.X), int32(scr.Y)
	rl.DrawCircle(cx, cy, 14, rl.NewColor(255, 140, 60, 255))
	rl.DrawCircleLines(cx, cy, 14, rl.NewColor(200, 80, 30, 255))
	rl.DrawCircleLines(cx, cy, 20, rl.NewColor(255, 140, 60, 140))
	rl.DrawCircleLines(cx, cy, 26, rl.NewColor(255, 140, 60, 80))
	fonts.Draw("STAR2", cx+18, cy-10, 16, rl.NewColor(255, 140, 60, 255))
	a.orbit.star2ScrX = float32(cx)
	a.orbit.star2ScrY = float32(cy)
}

func (a *App) drawOrbitPlanet2Overlay(cam rl.Camera3D, fwd rl.Vector3, scale float32, sw, sh int32) {
	world := rl.NewVector3(
		a.world.planet2Pos.X/scale,
		a.world.planet2Pos.Y/scale,
		a.world.planet2Pos.Z/scale,
	)
	scr := rl.GetWorldToScreen(world, cam)
	if !onScreen(scr, sw, sh, 200) {
		return
	}
	cx, cy := int32(scr.X), int32(scr.Y)
	rl.DrawCircle(cx, cy, 8, rl.NewColor(80, 180, 200, 255))
	rl.DrawCircleLines(cx, cy, 8, rl.NewColor(40, 120, 160, 255))
	rl.DrawCircleLines(cx, cy, 12, rl.NewColor(80, 180, 200, 140))
	fonts.Draw("PLANET2", cx+12, cy-8, 14, rl.NewColor(120, 200, 220, 255))
	a.orbit.planet2ScrX = float32(cx)
	a.orbit.planet2ScrY = float32(cy)
}

func (a *App) drawOrbitEarthOverlay(cam rl.Camera3D, fwd rl.Vector3, scale float32, sw, sh int32) {
	world := rl.NewVector3(
		a.world.earthPos.X/scale,
		a.world.earthPos.Y/scale,
		a.world.earthPos.Z/scale,
	)
	if !inFront(cam, world, fwd) {
		return
	}
	scr := rl.GetWorldToScreen(world, cam)
	if !onScreen(scr, sw, sh, 800) {
		return
	}
	cx, cy := int32(scr.X), int32(scr.Y)

	shF := float32(sh)
	distE := rl.Vector3Distance(cam.Position, world)
	if distE < 1 {
		distE = 1
	}
	// FOV=60 → tan(30°)=0.5773.
	projR := (protocol.PlanetRadius / scale) / distE * (shF / 2) / 0.5773
	if projR < 5 {
		projR = 5
	}
	if projR > 220 {
		projR = 220
	}
	soiR := (protocol.SOIEarth / scale) / distE * (shF / 2) / 0.5773
	if soiR > projR+4 && soiR < 3000 {
		rl.DrawCircleLines(cx, cy, soiR, rl.NewColor(120, 180, 240, 90))
	}
	a.orbit.earthScrX = float32(cx)
	a.orbit.earthScrY = float32(cy)
	a.orbit.earthScrR = projR

	rl.DrawCircle(cx, cy, projR, rl.NewColor(60, 130, 200, 255))
	rl.DrawCircleLines(cx, cy, projR, rl.NewColor(140, 200, 255, 255))
	rl.DrawCircleLines(cx, cy, projR+3, rl.NewColor(80, 140, 200, 160))

	lblX := cx + int32(projR) + 6
	lblY := cy - 8
	if lblX > sw-60 {
		lblX = cx - int32(projR) - 60
	}
	fonts.Draw("EARTH", lblX, lblY, 14, rl.NewColor(140, 200, 255, 255))
}

func (a *App) drawOrbitRocketOverlay(cam rl.Camera3D, fwd rl.Vector3, scale float32, sw, sh int32, r protocol.Rocket) {
	world := rl.NewVector3(
		(r.X+a.world.earthPos.X)/scale,
		(r.Y+a.world.earthPos.Y)/scale,
		(r.Z+a.world.earthPos.Z)/scale,
	)
	if !inFront(cam, world, fwd) {
		return
	}
	scr := rl.GetWorldToScreen(world, cam)
	if !onScreen(scr, sw, sh, 200) {
		return
	}
	cx, cy := int32(scr.X), int32(scr.Y)
	rl.DrawCircle(cx, cy, 6, rl.NewColor(80, 220, 100, 255))
	rl.DrawCircleLines(cx, cy, 6, rl.NewColor(20, 80, 20, 255))
	rl.DrawCircleLines(cx, cy, 10, rl.NewColor(80, 220, 100, 200))
	rl.DrawCircleLines(cx, cy, 14, rl.NewColor(80, 220, 100, 100))
	a.orbit.rocketScrX = float32(cx)
	a.orbit.rocketScrY = float32(cy)
}
