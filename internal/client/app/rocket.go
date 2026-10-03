package app

import (
	"math"
	"time"

	rl "github.com/gen2brain/raylib-go/raylib"

	"github.com/spacedreamer99/lodeen/internal/client/input"
	"github.com/spacedreamer99/lodeen/internal/shared/protocol"
)

// updatePilotedRocket — оркестратор тика ракеты. Возвращает false если вышли из ракеты.
func (a *App) updatePilotedRocket(dt float32) bool {
	found := findPilotedRocket(a.nc.Rockets(), a.player.rocketID)
	if found == nil {
		if time.Since(a.player.rocketBoardedAt) < 3*time.Second {
			return true // grace period — ждём снапшот
		}
		a.log.Info("rocket lost, leaving", "id", a.player.rocketID)
		return false
	}

	syncPlayerToRocket(a.flight, found)
	a.rocket.hudRocket = *found

	initRocketNoseIfNeeded(a)

	thrust := readThrustInput()
	handleNoseRotation(a, dt)
	_ = a.nc.RocketInput(thrust, a.rocket.noseX, a.rocket.noseY, a.rocket.noseZ, a.rocket.autoPilot)
	handleAutopilotKeys(a)

	if handleRocketExitKey(a) {
		return false
	}

	applyLookOrMap(a, dt)
	updateRocketCamera(a)

	a.log.Info("rocket pilot tick",
		"id", a.player.rocketID,
		"x", found.X, "y", found.Y, "z", found.Z,
		"fuel", found.Fuel,
		"thrust", thrust)
	return true
}

// findPilotedRocket ищет ракету по ID в снапшоте.
func findPilotedRocket(rs []protocol.Rocket, id string) *protocol.Rocket {
	for i := range rs {
		if rs[i].ID == id {
			return &rs[i]
		}
	}
	return nil
}

// syncPlayerToRocket — позиция игрока = позиция ракеты + смещение по носу.
func syncPlayerToRocket(f *input.FlightController, r *protocol.Rocket) {
	f.Pos = rl.NewVector3(
		r.X+r.DX*1.5,
		r.Y+r.DY*1.5,
		r.Z+r.DZ*1.5,
	)
	f.Vel = rl.NewVector3(0, 0, 0)
	f.HelioInit = true
}

// initRocketNoseIfNeeded — начальный нос от радиальной нормали (не доверяем DX/DY/DZ).
func initRocketNoseIfNeeded(a *App) {
	if a.rocket.noseInit {
		return
	}
	r := a.rocket.hudRocket
	rLen := float32(math.Sqrt(float64(r.X*r.X + r.Y*r.Y + r.Z*r.Z)))
	if rLen > 0.01 {
		a.rocket.noseX = r.X / rLen
		a.rocket.noseY = r.Y / rLen
		a.rocket.noseZ = r.Z / rLen
	} else {
		a.rocket.noseX, a.rocket.noseY, a.rocket.noseZ = 0, 1, 0
	}
	a.rocket.noseInit = true
	a.log.Info("rocket nose initialized",
		"x", a.rocket.noseX, "y", a.rocket.noseY, "z", a.rocket.noseZ)
}

// readThrustInput — Space = +1, Ctrl = -1.
func readThrustInput() float32 {
	switch {
	case rl.IsKeyDown(rl.KeySpace):
		return 1.0
	case rl.IsKeyDown(rl.KeyLeftControl):
		return -1.0
	default:
		return 0
	}
}

// handleNoseRotation — WASD вращает нос ракеты в локальной системе камеры.
func handleNoseRotation(a *App, dt float32) {
	upLocal := a.camera.camera.Up
	fwLocal := a.flight.Forward()
	rightLocal := rl.Vector3Normalize(rl.Vector3CrossProduct(fwLocal, upLocal))
	upTrue := rl.Vector3Normalize(rl.Vector3CrossProduct(rightLocal, fwLocal))

	const noseRate = 1.8
	step := noseRate * dt

	nose := rl.NewVector3(a.rocket.noseX, a.rocket.noseY, a.rocket.noseZ)
	rotated := false
	if rl.IsKeyDown(rl.KeyW) {
		nose = rotateAroundAxis(nose, rightLocal, -step)
		rotated = true
	}
	if rl.IsKeyDown(rl.KeyS) {
		nose = rotateAroundAxis(nose, rightLocal, step)
		rotated = true
	}
	if rl.IsKeyDown(rl.KeyA) {
		nose = rotateAroundAxis(nose, upTrue, step)
		rotated = true
	}
	if rl.IsKeyDown(rl.KeyD) {
		nose = rotateAroundAxis(nose, upTrue, -step)
		rotated = true
	}
	if rotated {
		nose = rl.Vector3Normalize(nose)
		a.rocket.noseX = nose.X
		a.rocket.noseY = nose.Y
		a.rocket.noseZ = nose.Z
		a.log.Info("rocket nose rotated", "x", nose.X, "y", nose.Y, "z", nose.Z)
	}
}

// handleAutopilotKeys — G/H/J/K/N/B/X переключают режим автопилота.
func handleAutopilotKeys(a *App) {
	switch {
	case rl.IsKeyPressed(rl.KeyG):
		a.rocket.autoPilot = "prograde"
	case rl.IsKeyPressed(rl.KeyH):
		a.rocket.autoPilot = "retrograde"
	case rl.IsKeyPressed(rl.KeyJ):
		a.rocket.autoPilot = "radial_out"
	case rl.IsKeyPressed(rl.KeyK):
		a.rocket.autoPilot = "radial_in"
	case rl.IsKeyPressed(rl.KeyN):
		a.rocket.autoPilot = "normal"
	case rl.IsKeyPressed(rl.KeyB):
		a.rocket.autoPilot = "antinormal"
	case rl.IsKeyPressed(rl.KeyX):
		a.rocket.autoPilot = ""
	}
}

// handleRocketExitKey — E: выход из ракеты. Возвращает true если вышли.
func handleRocketExitKey(a *App) bool {
	if !rl.IsKeyPressed(rl.KeyE) {
		return false
	}
	_ = a.nc.ExitRocket()
	a.player.rocketID = ""
	a.orbit.showOrbitMap = false
	a.rocket.noseInit = false
	a.log.Info("exit rocket sent")
	return true
}

// applyLookOrMap — мышь крутит обзор, если карта закрыта. Иначе — управление картой.
func applyLookOrMap(a *App, dt float32) {
	if a.orbit.showOrbitMap {
		a.updateOrbitMapInput()
		return
	}
	mdRocket := rl.GetMouseDelta()
	a.flight.UpdateLookOnly(dt, mdRocket)
}

// updateRocketCamera — камера следует за ракетой.
func updateRocketCamera(a *App) {
	fw := a.flight.Forward()
	a.camera.camera.Position = a.flight.Pos
	a.camera.camera.Target = rl.Vector3Add(a.flight.Pos, rl.Vector3Scale(fw, 100.0))
	a.camera.camera.Up = a.flight.CameraUp()
}
