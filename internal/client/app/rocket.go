package app

import (
	"math"
	"time"

	rl "github.com/gen2brain/raylib-go/raylib"

	"github.com/spacedreamer99/lodeen/internal/shared/protocol"
)

func (a *App) updatePilotedRocket(dt float32) bool {
	rs := a.nc.Rockets()

	var found *protocol.Rocket
	for i := range rs {
		if rs[i].ID == a.rocketID {
			found = &rs[i]
			break
		}
	}

	// Grace period: первые 3 секунды после board снапшот может ещё не прийти.
	// Не сбрасываем rocketID, просто ждём.
	graceful := time.Since(a.rocketBoardedAt) < 3*time.Second

	if found == nil {
		if graceful {
			// ещё ждём подтверждения от сервера
			return true
		}
		a.log.Info("rocket lost, leaving", "id", a.rocketID)
		return false
	}

	// Синхронизация позиции игрока с ракетой. found уже в helio.
	a.flight.Pos = rl.NewVector3(
		found.X+found.DX*1.5,
		found.Y+found.DY*1.5,
		found.Z+found.DZ*1.5,
	)
	a.flight.Vel = rl.NewVector3(0, 0, 0)
	a.flight.HelioInit = true

	// Сохранить данные для HUD.
	a.hudRocket = *found

	// Инициализация носа ракеты при первом кадре.
	// Считаем нормаль от центра планеты — не доверяем found.DX/DY/DZ.
	if !a.rocketNoseInit {
		rLen := float32(math.Sqrt(float64(found.X*found.X + found.Y*found.Y + found.Z*found.Z)))
		if rLen > 0.01 {
			a.rocketNoseX = found.X / rLen
			a.rocketNoseY = found.Y / rLen
			a.rocketNoseZ = found.Z / rLen
		} else {
			a.rocketNoseX = 0
			a.rocketNoseY = 1
			a.rocketNoseZ = 0
		}
		a.rocketNoseInit = true
		a.log.Info("rocket nose initialized",
			"x", a.rocketNoseX,
			"y", a.rocketNoseY,
			"z", a.rocketNoseZ)
	}

	// Тяга по Space.
	var thrust float32
	if rl.IsKeyDown(rl.KeySpace) {
		thrust = 1.0
	}
	if rl.IsKeyDown(rl.KeyLeftControl) {
		thrust = -1.0
	}

	// WASD — вращение носа ракеты в локальной системе камеры.
	{
		upLocal := a.camera.Up
		fwLocal := a.flight.Forward()
		rightLocal := rl.Vector3Normalize(rl.Vector3CrossProduct(fwLocal, upLocal))
		upTrue := rl.Vector3Normalize(rl.Vector3CrossProduct(rightLocal, fwLocal))

		const noseRate = 1.8
		angleStep := noseRate * dt

		nose := rl.NewVector3(a.rocketNoseX, a.rocketNoseY, a.rocketNoseZ)
		rotated := false
		if rl.IsKeyDown(rl.KeyW) {
			nose = rotateAroundAxis(nose, rightLocal, -angleStep)
			rotated = true
		}
		if rl.IsKeyDown(rl.KeyS) {
			nose = rotateAroundAxis(nose, rightLocal, angleStep)
			rotated = true
		}
		if rl.IsKeyDown(rl.KeyA) {
			nose = rotateAroundAxis(nose, upTrue, angleStep)
			rotated = true
		}
		if rl.IsKeyDown(rl.KeyD) {
			nose = rotateAroundAxis(nose, upTrue, -angleStep)
			rotated = true
		}
		if rotated {
			nose = rl.Vector3Normalize(nose)
			a.rocketNoseX = nose.X
			a.rocketNoseY = nose.Y
			a.rocketNoseZ = nose.Z
			a.log.Info("rocket nose rotated",
				"x", nose.X, "y", nose.Y, "z", nose.Z)
		}
	}

	// Отправляем желаемое направление носа ракеты.
	_ = a.nc.RocketInput(thrust, a.rocketNoseX, a.rocketNoseY, a.rocketNoseZ, a.autoPilot)

	// Автопилот: G prograde, H retrograde, J radial-out, K radial-in,
	// N normal, B antinormal, X off.
	if rl.IsKeyPressed(rl.KeyG) {
		a.autoPilot = "prograde"
	}
	if rl.IsKeyPressed(rl.KeyH) {
		a.autoPilot = "retrograde"
	}
	if rl.IsKeyPressed(rl.KeyJ) {
		a.autoPilot = "radial_out"
	}
	if rl.IsKeyPressed(rl.KeyK) {
		a.autoPilot = "radial_in"
	}
	if rl.IsKeyPressed(rl.KeyN) {
		a.autoPilot = "normal"
	}
	if rl.IsKeyPressed(rl.KeyB) {
		a.autoPilot = "antinormal"
	}
	if rl.IsKeyPressed(rl.KeyX) {
		a.autoPilot = ""
	}

	// M — карта орбиты.
	if rl.IsKeyPressed(rl.KeyM) {
		a.showOrbitMap = !a.showOrbitMap
		if a.showOrbitMap {
			a.orbitInit = false
			rl.EnableCursor()
			rl.ShowCursor()
		} else {
			rl.DisableCursor()
		}
	}

	// E — выйти.
	if rl.IsKeyPressed(rl.KeyE) {
		_ = a.nc.ExitRocket()
		a.rocketID = ""
		a.showOrbitMap = false
		a.rocketNoseInit = false
		a.log.Info("exit rocket sent")
		return false
	}

	// Если карта открыта — ракета НЕ крутится мышью.
	if a.showOrbitMap {
		a.updateOrbitMapInput()
	} else {
		mdRocket := rl.GetMouseDelta()
		a.flight.UpdateLookOnly(dt, mdRocket)
	}

	// Камера следует за ракетой.
	fw := a.flight.Forward()
	a.camera.Position = a.flight.Pos
	a.camera.Target = rl.Vector3Add(a.flight.Pos, rl.Vector3Scale(fw, 100.0))
	a.camera.Up = a.flight.CameraUp()

	a.log.Info("rocket pilot tick",
		"id", a.rocketID,
		"x", found.X, "y", found.Y, "z", found.Z,
		"fuel", found.Fuel,
		"thrust", thrust)
	return true
}
