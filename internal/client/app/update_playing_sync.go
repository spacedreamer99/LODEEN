package app

import (
	"math"

	rl "github.com/gen2brain/raylib-go/raylib"

	"github.com/spacedreamer99/lodeen/internal/shared/protocol"
)

// syncFlightBodies — обновляет позиции небесных тел в FlightController.
func (a *App) syncFlightBodies(dt float32) {
	if a.flight == nil {
		return
	}
	a.flight.EarthPos = rl.NewVector3(a.world.earthPos.X, a.world.earthPos.Y, a.world.earthPos.Z)
	a.flight.SunPos = rl.NewVector3(protocol.SunPos.X, protocol.SunPos.Y, protocol.SunPos.Z)
	a.flight.EarthVel = rl.NewVector3(a.world.earthVel.X, a.world.earthVel.Y, a.world.earthVel.Z)
	a.flight.SyncToEarthFrame(dt)
}

// applyUILock — жёстко прибить игрока к Земле пока открыт UI.
func (a *App) applyUILock(dt float32) {
	anyUI := a.ui.showInventory || a.ui.showCraft || a.ui.showContract || a.ui.showFactory || a.chat.Open
	if a.flight == nil || !anyUI {
		a.player.flightLocked = false
		return
	}

	if !a.player.flightLocked {
		a.player.flightLockedRelPos = protocolVector3Sub(a.flight.Pos, a.world.earthPos)
		a.player.flightLockedRelVel = protocolVector3Sub(a.flight.Vel, a.world.earthVel)
		a.player.flightLocked = true
		a.flight.RelPos = rl.NewVector3(
			a.player.flightLockedRelPos.X, a.player.flightLockedRelPos.Y, a.player.flightLockedRelPos.Z)
		a.flight.RelVel = rl.NewVector3(
			a.player.flightLockedRelVel.X, a.player.flightLockedRelVel.Y, a.player.flightLockedRelVel.Z)
		a.flight.RelInit = true
	}

	a.flight.Pos = rl.NewVector3(
		a.world.earthPos.X+a.player.flightLockedRelPos.X,
		a.world.earthPos.Y+a.player.flightLockedRelPos.Y,
		a.world.earthPos.Z+a.player.flightLockedRelPos.Z,
	)
	a.flight.Vel = rl.NewVector3(
		a.world.earthVel.X+a.player.flightLockedRelVel.X,
		a.world.earthVel.Y+a.player.flightLockedRelVel.Y,
		a.world.earthVel.Z+a.player.flightLockedRelVel.Z,
	)
	a.flight.RelPos = rl.NewVector3(
		a.player.flightLockedRelPos.X, a.player.flightLockedRelPos.Y, a.player.flightLockedRelPos.Z)
	a.flight.RelVel = rl.NewVector3(
		a.player.flightLockedRelVel.X, a.player.flightLockedRelVel.Y, a.player.flightLockedRelVel.Z)

	// Камера едет с Землёй без lerp — иначе при закрытии UI дёргается.
	fw := a.flight.Forward()
	a.camera.camSmoothPos = a.flight.Pos
	a.camera.camSmoothTarget = rl.Vector3Add(a.flight.Pos, rl.Vector3Scale(fw, 100.0))
	a.camera.camSmoothInit = true
	a.camera.camera.Position = a.camera.camSmoothPos
	a.camera.camera.Target = a.camera.camSmoothTarget
	a.camera.camera.Up = a.flight.CameraUp()
}

// syncHPFromNetwork — подтягивает HP с сервера.
func (a *App) syncHPFromNetwork() {
	if a.nc == nil {
		return
	}
	if hp := a.nc.OwnHP(); hp > 0 || a.player.hpReceived {
		a.player.hp = hp
		a.player.hpReceived = true
	}
}

// emitPlayerState — отправляет гео-позицию/поворот игрока на сервер.
func (a *App) emitPlayerState() {
	if a.flight == nil {
		return
	}
	fw := a.flight.Forward()
	yawF := float32(math.Atan2(float64(-fw.X), float64(-fw.Z)))
	pitchF := float32(math.Asin(float64(fw.Y)))
	a.nc.SetState(protocol.PlayerState{
		X:   a.flight.Pos.X - a.world.earthPos.X,
		Y:   a.flight.Pos.Y - a.world.earthPos.Y,
		Z:   a.flight.Pos.Z - a.world.earthPos.Z,
		Yaw: yawF, Pitch: pitchF,
	})
}
