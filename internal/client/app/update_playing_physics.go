package app

import (
	"fmt"
	"math"
	"time"

	rl "github.com/gen2brain/raylib-go/raylib"

	"github.com/spacedreamer99/lodeen/internal/shared/protocol"
)

// updateFlightPhysics — основной тик физики полёта + камера + диак-лог.
func (a *App) updateFlightPhysics(dt float32) {
	a.setCursorCaptured(true)
	if a.flight == nil {
		return
	}

	// HelioInit — только когда EarthPos реально пришла.
	epLen := a.world.earthPos.X*a.world.earthPos.X + a.world.earthPos.Y*a.world.earthPos.Y + a.world.earthPos.Z*a.world.earthPos.Z
	if !a.flight.HelioInit && epLen > 100*100 {
		a.flight.Pos = rl.Vector3Add(a.flight.Pos, a.flight.EarthPos)
		a.flight.HelioInit = true
		a.log.Info("flight helio init",
			"spawnGeo", fmt.Sprintf("%.2f,%.2f,%.2f", a.flight.Pos.X-a.flight.EarthPos.X, a.flight.Pos.Y-a.flight.EarthPos.Y, a.flight.Pos.Z-a.flight.EarthPos.Z),
			"earth", fmt.Sprintf("%.2f,%.2f,%.2f", a.flight.EarthPos.X, a.flight.EarthPos.Y, a.flight.EarthPos.Z),
			"helio", fmt.Sprintf("%.2f,%.2f,%.2f", a.flight.Pos.X, a.flight.Pos.Y, a.flight.Pos.Z))
	}

	md := rl.GetMouseDelta()
	a.flight.Update(dt, md)

	a.logFlightDiag(dt, md)
	a.updateCameraSmoothing()
}

// logFlightDiag — раз в ~секунду пишет в лог состояние игрока и нажатые клавиши.
func (a *App) logFlightDiag(dt float32, md rl.Vector2) {
	a.diag.debugFrame++
	keys := readKeysString()
	mouse := readMouseString()

	now := time.Now()
	changed := keys != a.diag.lastKeys || mouse != a.diag.lastMouse || md.X != 0 || md.Y != 0
	idle := now.Sub(a.diag.lastLogAt) > time.Second
	if !(changed || idle) || now.Sub(a.diag.lastLogAt) < 30*time.Millisecond {
		return
	}

	edgeKeys := ""
	if keys != a.diag.lastKeys {
		edgeKeys = keys
	}

	relX := a.flight.Pos.X - a.world.earthPos.X
	relY := a.flight.Pos.Y - a.world.earthPos.Y
	relZ := a.flight.Pos.Z - a.world.earthPos.Z
	relDist := float32(math.Sqrt(float64(relX*relX + relY*relY + relZ*relZ)))
	surfaceR := protocol.SurfaceRadius(protocol.Vector3{X: relX, Y: relY, Z: relZ})
	minR := surfaceR + protocol.PlayerHeight
	onGround := relDist <= minR+0.5

	a.log.Info("state",
		"dt", fmt.Sprintf("%.4f", dt),
		"helioPos", fmt.Sprintf("%.2f,%.2f,%.2f", a.flight.Pos.X, a.flight.Pos.Y, a.flight.Pos.Z),
		"earthPos", fmt.Sprintf("%.2f,%.2f,%.2f", a.world.earthPos.X, a.world.earthPos.Y, a.world.earthPos.Z),
		"earthVel", fmt.Sprintf("%.3f,%.3f,%.3f", a.world.earthVel.X, a.world.earthVel.Y, a.world.earthVel.Z),
		"relDist", fmt.Sprintf("%.3f", relDist),
		"minR", fmt.Sprintf("%.3f", minR),
		"onGround", onGround,
		"vel", fmt.Sprintf("%.2f,%.2f,%.2f", a.flight.Vel.X, a.flight.Vel.Y, a.flight.Vel.Z),
		"velLen", fmt.Sprintf("%.2f", rl.Vector3Length(a.flight.Vel)),
		"edgeK", edgeKeys,
		"keys", keys,
	)

	a.diag.lastLogAt = now
	a.diag.lastPos = a.flight.Pos
	a.diag.lastKeys = keys
	a.diag.lastMouse = mouse
}

// updateCameraSmoothing — lerp камеры за ракетой.
func (a *App) updateCameraSmoothing() {
	fw := a.flight.Forward()
	targetNew := rl.Vector3Add(a.flight.Pos, rl.Vector3Scale(fw, 100.0))

	if !a.camera.camSmoothInit {
		a.camera.camSmoothPos = a.flight.Pos
		a.camera.camSmoothTarget = targetNew
		a.camera.camSmoothInit = true
	} else {
		const camAlpha = float32(0.5)
		a.camera.camSmoothPos = rl.Vector3Add(a.camera.camSmoothPos,
			rl.Vector3Scale(rl.Vector3Subtract(a.flight.Pos, a.camera.camSmoothPos), camAlpha))
		a.camera.camSmoothTarget = rl.Vector3Add(a.camera.camSmoothTarget,
			rl.Vector3Scale(rl.Vector3Subtract(targetNew, a.camera.camSmoothTarget), camAlpha))
	}
	a.camera.camera.Position = a.camera.camSmoothPos
	a.camera.camera.Target = a.camera.camSmoothTarget
	a.camera.camera.Up = a.flight.CameraUp()
}

// protocolVector3Sub — вычитание двух rl.Vector3 в protocol.Vector3.
// Используется в applyUILock / checkPause / updateUnfocusFreeze.
func protocolVector3Sub(a rl.Vector3, b protocol.Vector3) protocol.Vector3 {
	return protocol.Vector3{X: a.X - b.X, Y: a.Y - b.Y, Z: a.Z - b.Z}
}
