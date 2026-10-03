package app

import (
	"math"

	rl "github.com/gen2brain/raylib-go/raylib"

	"github.com/spacedreamer99/lodeen/internal/shared/protocol"
)

// drawOrbitMap — оркестратор орбитальной карты.
// Сама отрисовка — в orbitmap_3d.go, 2D overlay — в orbitmap_overlay.go,
// HUD — в orbitmap_hud.go.
func (a *App) drawOrbitMap() {
	r := a.rocket.hudRocket
	haveRocket := r.ID != ""

	a.initOrbitCameraIfNeeded(r, haveRocket)

	sunH, earthH, star2H, planet2H, rocketH := a.computeHelioPositions(r, haveRocket)
	center := computeOrbitCenter(a.orbit.orbitFocus, sunH, earthH, star2H, planet2H, rocketH)

	scale, _, mapCam := buildOrbitCamera(center, a.orbit)

	sw := int32(rl.GetScreenWidth())
	sh := int32(rl.GetScreenHeight())
	rl.DrawRectangle(0, 0, sw, sh, rl.NewColor(3, 5, 15, 255))

	rl.BeginMode3D(mapCam)
	rl.PushMatrix()
	rl.Scalef(1.0/scale, 1.0/scale, 1.0/scale)

	a.drawOrbit3DScene(r, haveRocket, sunH, earthH, star2H, planet2H, rocketH)

	rl.PopMatrix()
	rl.EndMode3D()

	fwd := cameraForward(mapCam)
	a.drawOrbitOverlays(mapCam, fwd, scale, sw, sh, haveRocket, r)
	a.drawOrbitHUD(r, sw, sh)
}

// initOrbitCameraIfNeeded — инициализация камеры при первом открытии карты.
func (a *App) initOrbitCameraIfNeeded(r protocol.Rocket, haveRocket bool) {
	if a.orbit.orbitInit {
		return
	}
	a.orbit.orbitInit = true
	a.orbit.orbitAzimuth = 0.6
	a.orbit.orbitElevation = 0.5

	dist := float32(500.0)
	switch {
	case haveRocket && r.PrimaryBody == "sun":
		dist = protocol.SunDistance * 1.5
	case haveRocket:
		dist = r.Altitude * 2.5
	default:
		dist = protocol.SOIEarth * 2.0
	}
	if dist < 200 {
		dist = 200
	}
	if dist > 150000 {
		dist = 150000
	}
	a.orbit.orbitDistance = dist
}

// computeHelioPositions — текущие helio-координаты всех тел.
func (a *App) computeHelioPositions(r protocol.Rocket, haveRocket bool) (sunH, earthH, star2H, planet2H, rocketH rl.Vector3) {
	sunH = rl.NewVector3(protocol.SunPos.X, protocol.SunPos.Y, protocol.SunPos.Z)
	earthH = rl.NewVector3(a.world.earthPos.X, a.world.earthPos.Y, a.world.earthPos.Z)
	star2H = rl.NewVector3(protocol.Star2Pos.X, protocol.Star2Pos.Y, protocol.Star2Pos.Z)
	planet2H = rl.NewVector3(a.world.planet2Pos.X, a.world.planet2Pos.Y, a.world.planet2Pos.Z)
	rocketH = earthH
	if haveRocket {
		rocketH = rl.NewVector3(
			r.X+a.world.earthPos.X,
			r.Y+a.world.earthPos.Y,
			r.Z+a.world.earthPos.Z,
		)
	}
	return
}

// computeOrbitCenter — центр карты по orbitFocus или авто (Солнце).
func computeOrbitCenter(focus string, sunH, earthH, star2H, planet2H, rocketH rl.Vector3) rl.Vector3 {
	switch focus {
	case "earth":
		return earthH
	case "sun":
		return sunH
	case "star2":
		return star2H
	case "planet2":
		return planet2H
	case "rocket":
		return rocketH
	default:
		return sunH
	}
}

// buildOrbitCamera — масштаб + позиция камеры + сама Camera3D.
// Обходим far plane raylib (~1000) через scaled-space.
func buildOrbitCamera(center rl.Vector3, o OrbitState) (scale float32, centerS rl.Vector3, cam rl.Camera3D) {
	scale = 1.0
	if o.orbitDistance > 900 {
		scale = o.orbitDistance / 900
	}
	distS := o.orbitDistance / scale
	centerS = rl.NewVector3(center.X/scale, center.Y/scale, center.Z/scale)

	cosEl := float32(math.Cos(float64(o.orbitElevation)))
	camPos := rl.NewVector3(
		centerS.X+distS*cosEl*float32(math.Cos(float64(o.orbitAzimuth))),
		centerS.Y+distS*float32(math.Sin(float64(o.orbitElevation))),
		centerS.Z+distS*cosEl*float32(math.Sin(float64(o.orbitAzimuth))),
	)
	cam = rl.Camera3D{
		Position:   camPos,
		Target:     centerS,
		Up:         rl.NewVector3(0, 1, 0),
		Fovy:       60,
		Projection: rl.CameraPerspective,
	}
	return
}

// cameraForward — вектор направления камеры (для проверки «перед камерой»).
func cameraForward(cam rl.Camera3D) rl.Vector3 {
	return rl.NewVector3(
		cam.Target.X-cam.Position.X,
		cam.Target.Y-cam.Position.Y,
		cam.Target.Z-cam.Position.Z,
	)
}
