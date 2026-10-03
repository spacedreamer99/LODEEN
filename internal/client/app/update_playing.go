package app

import (
	"fmt"

	rl "github.com/gen2brain/raylib-go/raylib"

	"github.com/spacedreamer99/lodeen/internal/client/state"
)

// updatePlaying — оркестратор игрового режима.
// Каждый helper возвращает true если обработал кадр и остальную логику надо пропустить.
func (a *App) updatePlaying(dt float32) {
	a.syncInvSlots()

	if a.updateRocketMode(dt) {
		return
	}

	a.syncFlightBodies(dt)
	a.applyUILock(dt)

	if a.updateDialogOverlays() {
		return
	}

	a.syncHPFromNetwork()

	if a.updateChatOverlay() {
		return
	}

	if a.updateOrbitMapOverlay() {
		return
	}

	a.handleModeSwitchKey()

	if a.updateInventoryOverlay() {
		return
	}

	a.updateSlotWheel()
	a.syncHeldItem()

	if a.handleChatKeys() {
		return
	}

	a.handlePickupKey()
	a.handleMountKey()

	if a.handleMouseClick() {
		return
	}

	if a.checkDeath() {
		return
	}

	if a.checkPause() {
		return
	}

	if a.updateUnfocusFreeze() {
		return
	}

	a.restoreFocus()
	a.updateFlightPhysics(dt)
	a.emitPlayerState()
}

// updateRocketMode возвращает true если пилотирование ракеты забрало кадр.
func (a *App) updateRocketMode(dt float32) bool {
	if a.player.rocketID == "" || a.flight == nil {
		return false
	}
	if a.updatePilotedRocket(dt) {
		return true
	}
	a.player.rocketID = ""
	return false
}

// checkDeath — при нулевом HP переключает в режим смерти и закрывает соединение.
func (a *App) checkDeath() bool {
	if !a.player.hpReceived || a.myHP() > 0 {
		return false
	}
	a.log.Info("player HP zero, switching to death screen")
	a.mode = state.ModeDead
	rl.EnableCursor()
	rl.ShowCursor()
	if a.nc != nil {
		a.nc.Close()
	}
	return true
}

// checkPause — Esc: переводит в режим паузы, запоминая относительную позицию.
func (a *App) checkPause() bool {
	if !rl.IsKeyPressed(rl.KeyEscape) {
		return false
	}
	if a.flight != nil && a.flight.HelioInit {
		a.pause.pausedRelPos = protocolVector3Sub(a.flight.Pos, a.world.earthPos)
		a.pause.pausedRelVel = protocolVector3Sub(a.flight.Vel, a.world.earthVel)
		a.log.Info("PAUSE ENTERED",
			"helio", fmt.Sprintf("%.2f,%.2f,%.2f", a.flight.Pos.X, a.flight.Pos.Y, a.flight.Pos.Z),
			"earth", fmt.Sprintf("%.2f,%.2f,%.2f", a.world.earthPos.X, a.world.earthPos.Y, a.world.earthPos.Z),
			"relPos", fmt.Sprintf("%.2f,%.2f,%.2f", a.pause.pausedRelPos.X, a.pause.pausedRelPos.Y, a.pause.pausedRelPos.Z),
			"relVel", fmt.Sprintf("%.3f,%.3f,%.3f", a.pause.pausedRelVel.X, a.pause.pausedRelVel.Y, a.pause.pausedRelVel.Z))
	}
	a.mode = state.ModePaused
	return true
}

// updateUnfocusFreeze — при потере фокуса окна «замораживает» ракету относительно Земли.
// Возвращает true если кадр поглощён.
func (a *App) updateUnfocusFreeze() bool {
	if rl.IsWindowFocused() {
		return false
	}
	a.setCursorCaptured(false)
	if a.flight == nil || !a.flight.HelioInit {
		return true
	}
	if !a.pause.unfocusFreeze {
		a.pause.unfocusRelPos = protocolVector3Sub(a.flight.Pos, a.world.earthPos)
		a.pause.unfocusRelVel = protocolVector3Sub(a.flight.Vel, a.world.earthVel)
		a.pause.unfocusFreeze = true
		a.log.Info("UNFOCUS freeze",
			"relPos", fmt.Sprintf("%.2f,%.2f,%.2f", a.pause.unfocusRelPos.X, a.pause.unfocusRelPos.Y, a.pause.unfocusRelPos.Z))
	}
	a.flight.Pos = rl.NewVector3(
		a.world.earthPos.X+a.pause.unfocusRelPos.X,
		a.world.earthPos.Y+a.pause.unfocusRelPos.Y,
		a.world.earthPos.Z+a.pause.unfocusRelPos.Z,
	)
	a.flight.Vel = rl.NewVector3(
		a.world.earthVel.X+a.pause.unfocusRelVel.X,
		a.world.earthVel.Y+a.pause.unfocusRelVel.Y,
		a.world.earthVel.Z+a.pause.unfocusRelVel.Z,
	)
	fw := a.flight.Forward()
	a.camera.camera.Position = a.flight.Pos
	a.camera.camera.Target = rl.Vector3Add(a.flight.Pos, rl.Vector3Scale(fw, 100.0))
	a.camera.camera.Up = a.flight.CameraUp()
	return true
}

// restoreFocus — сбрасывает флаг заморозки при возврате фокуса.
func (a *App) restoreFocus() {
	if !a.pause.unfocusFreeze {
		return
	}
	a.pause.unfocusFreeze = false
	a.log.Info("UNFOCUS restored",
		"helio", fmt.Sprintf("%.2f,%.2f,%.2f", a.flight.Pos.X, a.flight.Pos.Y, a.flight.Pos.Z),
		"earth", fmt.Sprintf("%.2f,%.2f,%.2f", a.world.earthPos.X, a.world.earthPos.Y, a.world.earthPos.Z))
}
