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
		if rs[i].ID == a.player.rocketID {
			found = &rs[i]
			break
		}
	}

	// Grace period: первые 3 секунды после board снапшот может ещё не прийти.
	// Не сбрасываем rocketID, просто ждём.
	graceful := time.Since(a.player.rocketBoardedAt) < 3*time.Second

	if found == nil {
		if graceful {
			// ещё ждём подтверждения от сервера
			return true
		}
		a.log.Info("rocket lost, leaving", "id", a.player.rocketID)
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
	a.rocket.hudRocket = *found

	// Инициализация носа ракеты при первом кадре.
	// Считаем нормаль от центра планеты — не доверяем found.DX/DY/DZ.
	if !a.rocket.noseInit {
		rLen := float32(math.Sqrt(float64(found.X*found.X + found.Y*found.Y + found.Z*found.Z)))
		if rLen > 0.01 {
			a.rocket.noseX = found.X / rLen
			a.rocket.noseY = found.Y / rLen
			a.rocket.noseZ = found.Z / rLen
		} else {
			a.rocket.noseX = 0
			a.rocket.noseY = 1
			a.rocket.noseZ = 0
		}
		a.rocket.noseInit = true
		a.log.Info("rocket nose initialized",
			"x", a.rocket.noseX,
			"y", a.rocket.noseY,
			"z", a.rocket.noseZ)
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
		upLocal := a.camera.camera.Up
		fwLocal := a.flight.Forward()
		rightLocal := rl.Vector3Normalize(rl.Vector3CrossProduct(fwLocal, upLocal))
		upTrue := rl.Vector3Normalize(rl.Vector3CrossProduct(rightLocal, fwLocal))

		const noseRate = 1.8
		angleStep := noseRate * dt

		nose := rl.NewVector3(a.rocket.noseX, a.rocket.noseY, a.rocket.noseZ)
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
			a.rocket.noseX = nose.X
			a.rocket.noseY = nose.Y
			a.rocket.noseZ = nose.Z
			a.log.Info("rocket nose rotated",
				"x", nose.X, "y", nose.Y, "z", nose.Z)
		}
	}

	// Отправляем желаемое направление носа ракеты.
	_ = a.nc.RocketInput(thrust, a.rocket.noseX, a.rocket.noseY, a.rocket.noseZ, a.rocket.autoPilot)

	// Автопилот: G prograde, H retrograde, J radial-out, K radial-in,
	// N normal, B antinormal, X off.
	if rl.IsKeyPressed(rl.KeyG) {
		a.rocket.autoPilot = "prograde"
	}
	if rl.IsKeyPressed(rl.KeyH) {
		a.rocket.autoPilot = "retrograde"
	}
	if rl.IsKeyPressed(rl.KeyJ) {
		a.rocket.autoPilot = "radial_out"
	}
	if rl.IsKeyPressed(rl.KeyK) {
		a.rocket.autoPilot = "radial_in"
	}
	if rl.IsKeyPressed(rl.KeyN) {
		a.rocket.autoPilot = "normal"
	}
	if rl.IsKeyPressed(rl.KeyB) {
		a.rocket.autoPilot = "antinormal"
	}
	if rl.IsKeyPressed(rl.KeyX) {
		a.rocket.autoPilot = ""
	}

	// E — выйти.
	if rl.IsKeyPressed(rl.KeyE) {
		_ = a.nc.ExitRocket()
		a.player.rocketID = ""
		a.orbit.showOrbitMap = false
		a.rocket.noseInit = false
		a.log.Info("exit rocket sent")
		return false
	}

	// Если карта открыта — ракета НЕ крутится мышью.
	if a.orbit.showOrbitMap {
		a.updateOrbitMapInput()
	} else {
		mdRocket := rl.GetMouseDelta()
		a.flight.UpdateLookOnly(dt, mdRocket)
	}

	// Камера следует за ракетой.
	fw := a.flight.Forward()
	a.camera.camera.Position = a.flight.Pos
	a.camera.camera.Target = rl.Vector3Add(a.flight.Pos, rl.Vector3Scale(fw, 100.0))
	a.camera.camera.Up = a.flight.CameraUp()

	a.log.Info("rocket pilot tick",
		"id", a.player.rocketID,
		"x", found.X, "y", found.Y, "z", found.Z,
		"fuel", found.Fuel,
		"thrust", thrust)
	return true
}
