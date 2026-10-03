package app

import (
	"fmt"
	"math"
	"time"

	rl "github.com/gen2brain/raylib-go/raylib"

	"github.com/spacedreamer99/lodeen/internal/client/input"
	"github.com/spacedreamer99/lodeen/internal/client/state"
	"github.com/spacedreamer99/lodeen/internal/shared/protocol"
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

// --- Ракета ---

// updateRocketMode возвращает true если пилотирование ракеты забрало кадр.
func (a *App) updateRocketMode(dt float32) bool {
	if a.rocketID == "" || a.flight == nil {
		return false
	}
	if a.updatePilotedRocket(dt) {
		return true
	}
	a.rocketID = ""
	return false
}

// --- Синк тел в flight ---

func (a *App) syncFlightBodies(dt float32) {
	if a.flight == nil {
		return
	}
	a.flight.EarthPos = rl.NewVector3(a.earthPos.X, a.earthPos.Y, a.earthPos.Z)
	a.flight.SunPos = rl.NewVector3(protocol.SunPos.X, protocol.SunPos.Y, protocol.SunPos.Z)
	a.flight.EarthVel = rl.NewVector3(a.earthVel.X, a.earthVel.Y, a.earthVel.Z)
	a.flight.SyncToEarthFrame(dt)
}

// --- UI lock: жёстко прибить игрока к Земле пока открыт UI ---

func (a *App) applyUILock(dt float32) {
	anyUI := a.ui.showInventory || a.ui.showCraft || a.ui.showContract || a.ui.showFactory || a.chat.Open
	if a.flight == nil || !anyUI {
		a.flightLocked = false
		return
	}

	if !a.flightLocked {
		a.flightLockedRelPos = protocol.Vector3{
			X: a.flight.Pos.X - a.earthPos.X,
			Y: a.flight.Pos.Y - a.earthPos.Y,
			Z: a.flight.Pos.Z - a.earthPos.Z,
		}
		a.flightLockedRelVel = protocol.Vector3{
			X: a.flight.Vel.X - a.earthVel.X,
			Y: a.flight.Vel.Y - a.earthVel.Y,
			Z: a.flight.Vel.Z - a.earthVel.Z,
		}
		a.flightLocked = true
		a.flight.RelPos = rl.NewVector3(
			a.flightLockedRelPos.X, a.flightLockedRelPos.Y, a.flightLockedRelPos.Z)
		a.flight.RelVel = rl.NewVector3(
			a.flightLockedRelVel.X, a.flightLockedRelVel.Y, a.flightLockedRelVel.Z)
		a.flight.RelInit = true
	}

	a.flight.Pos = rl.NewVector3(
		a.earthPos.X+a.flightLockedRelPos.X,
		a.earthPos.Y+a.flightLockedRelPos.Y,
		a.earthPos.Z+a.flightLockedRelPos.Z,
	)
	a.flight.Vel = rl.NewVector3(
		a.earthVel.X+a.flightLockedRelVel.X,
		a.earthVel.Y+a.flightLockedRelVel.Y,
		a.earthVel.Z+a.flightLockedRelVel.Z,
	)
	a.flight.RelPos = rl.NewVector3(
		a.flightLockedRelPos.X, a.flightLockedRelPos.Y, a.flightLockedRelPos.Z)
	a.flight.RelVel = rl.NewVector3(
		a.flightLockedRelVel.X, a.flightLockedRelVel.Y, a.flightLockedRelVel.Z)

	// Камера едет с Землёй без lerp — иначе при закрытии UI дёргается.
	fw := a.flight.Forward()
	a.camSmoothPos = a.flight.Pos
	a.camSmoothTarget = rl.Vector3Add(a.flight.Pos, rl.Vector3Scale(fw, 100.0))
	a.camSmoothInit = true
	a.camera.Position = a.camSmoothPos
	a.camera.Target = a.camSmoothTarget
	a.camera.Up = a.flight.CameraUp()
}

// --- Диалоги ---

func (a *App) updateDialogOverlays() bool {
	if a.ui.showContract {
		a.updateContract()
		return true
	}
	if a.ui.showFactory {
		a.updateFactory()
		return true
	}
	return false
}

// --- HP, чат, орбитальная карта ---

func (a *App) syncHPFromNetwork() {
	if a.nc == nil {
		return
	}
	if hp := a.nc.OwnHP(); hp > 0 || a.hpReceived {
		a.hp = hp
		a.hpReceived = true
	}
}

func (a *App) updateChatOverlay() bool {
	if !a.chat.Open {
		return false
	}
	a.setCursorCaptured(false)
	if text, ok := a.chat.Update(); ok && text != "" {
		if err := a.nc.SendChat(text); err != nil {
			a.log.Warn("send chat", "err", err)
		}
	}
	return true
}

func (a *App) updateOrbitMapOverlay() bool {
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
	if a.showOrbitMap {
		a.updateOrbitMapInput()
		return true
	}
	return false
}

// --- Клавиши HUD: F1, C, I, Esc ---

func (a *App) handleModeSwitchKey() {
	if !rl.IsKeyPressed(rl.KeyF1) || a.flight == nil {
		return
	}
	if a.flight.Mode == input.ModeCreative {
		a.flight.SwitchMode(input.ModeSurvival)
	} else {
		a.flight.SwitchMode(input.ModeCreative)
	}
	a.log.Info("mode changed", "mode", a.flight.Mode)
}

// updateInventoryOverlay открывает/закрывает окна крафта и инвентаря.
// Возвращает true если UI открыт — остальная игровая логика не работает.
func (a *App) updateInventoryOverlay() bool {
	// C — открыть окно крафта
	if rl.IsKeyPressed(rl.KeyC) {
		a.ui.showCraft = true
		a.ui.showInventory = false
		rl.EnableCursor()
		rl.ShowCursor()
	}
	// I — toggle инвентаря
	if rl.IsKeyPressed(rl.KeyI) {
		a.ui.showInventory = !a.ui.showInventory
		if a.ui.showInventory {
			a.ui.showCraft = false
			rl.EnableCursor()
			rl.ShowCursor()
		} else {
			rl.DisableCursor()
		}
	}
	if a.ui.showInventory || a.ui.showCraft {
		if rl.IsKeyPressed(rl.KeyEscape) {
			a.ui.showInventory = false
			a.ui.showCraft = false
			rl.DisableCursor()
		}
		return true
	}
	return false
}

// --- Выбор слота и синк held item ---

func (a *App) updateSlotWheel() {
	if a.flight == nil || a.flight.Mode != input.ModeSurvival {
		return
	}
	wheel := rl.GetMouseWheelMove()
	if wheel > 0 {
		a.ui.selectedSlot = (a.ui.selectedSlot + 1) % 16
	} else if wheel < 0 {
		a.ui.selectedSlot = (a.ui.selectedSlot + 15) % 16
	}
}

func (a *App) syncHeldItem() {
	held := a.heldItem()
	if held == a.lastSentHeld {
		return
	}
	a.lastSentHeld = held
	if err := a.nc.SelectItem(held); err != nil {
		a.log.Warn("select item", "err", err)
	}
}

// --- Чат: T, /, F, E ---

func (a *App) handleChatKeys() bool {
	if rl.IsKeyPressed(rl.KeyT) {
		a.chat.Begin()
		return true
	}
	if rl.IsKeyPressed(rl.KeySlash) {
		a.chat.BeginWith("/")
		return true
	}
	return false
}

func (a *App) handlePickupKey() {
	if rl.IsKeyPressed(rl.KeyF) {
		a.tryPickup()
	}
}

// handleMountKey — E: сесть/встать с мамонта или лодки.
func (a *App) handleMountKey() {
	if !rl.IsKeyPressed(rl.KeyE) || a.flight == nil {
		return
	}
	switch {
	case a.ridingID != "":
		if err := a.nc.RideMammoth(""); err != nil {
			a.log.Warn("ride", "err", err)
		}
		a.ridingID = ""
		a.flight.Riding = false
		a.log.Info("dismount sent")

	case a.boatID != "":
		if err := a.nc.EnterBoat(""); err != nil {
			a.log.Warn("boat exit", "err", err)
		}
		a.boatID = ""
		a.flight.InBoat = false
		a.log.Info("boat exit sent")

	default:
		if id := a.saddledMammothNearby(); id != "" {
			if err := a.nc.RideMammoth(id); err != nil {
				a.log.Warn("ride", "err", err)
			}
			a.ridingID = id
			a.flight.Riding = true
			a.log.Info("mount sent", "id", id)
			return
		}
		if id := a.boatNearby(); id != "" {
			if err := a.nc.EnterBoat(id); err != nil {
				a.log.Warn("boat enter", "err", err)
			}
			a.boatID = id
			a.flight.InBoat = true
			a.log.Info("boat enter sent", "id", id)
		}
	}
}

// --- ЛКМ ---

// handleMouseClick возвращает true если клик требует выхода из updatePlaying
// (например открыл контракт с розовым ботом).
func (a *App) handleMouseClick() bool {
	if !rl.IsMouseButtonPressed(rl.MouseLeftButton) || a.flight == nil {
		return false
	}

	// Приоритет: колодец. Клик по нему всегда даёт воду.
	if id := a.wellInSight(); id != "" {
		if err := a.nc.TakeWater(id); err != nil {
			a.log.Warn("take water", "err", err)
		}
		a.log.Info("take water sent (LMB)", "well", id)
		return false
	}

	held := a.heldItem()
	if held == "" {
		return a.handleEmptyHandClick()
	}
	a.handleHeldItemClick(held)
	return false
}

// handleHeldItemClick — клик с предметом в руке.
func (a *App) handleHeldItemClick(held string) {
	switch held {
	case "spear":
		a.throwSpear()
	case "fruit":
		a.useFruit()
	case "water":
		a.useWater()
	case "leash":
		a.useLeash()
	case "house", "solar", "battery", "factory":
		a.placeStructure(held)
	case "rocket":
		x, y, z, _ := a.placeForward(5.0)
		if err := a.nc.PlaceRocket(x, y, z); err != nil {
			a.log.Warn("place rocket", "err", err)
		}
		a.log.Info("place rocket sent (LMB)")
	case "boat":
		fw := a.flight.Forward()
		pos := rl.Vector3Add(a.camera.Position, rl.Vector3Scale(fw, 5.0))
		yaw := float32(math.Atan2(float64(fw.X), float64(fw.Z)))
		if err := a.nc.PlaceBoat(pos.X, pos.Y, pos.Z, yaw); err != nil {
			a.log.Warn("place boat", "err", err)
		}
		a.log.Info("place boat sent (LMB)")
	case "saddle":
		if id := a.mammothInReach(); id != "" {
			if err := a.nc.SaddleMammoth(id); err != nil {
				a.log.Warn("saddle mammoth", "err", err)
			}
			a.log.Info("saddle mammoth sent (LMB)", "id", id)
		} else {
			a.log.Info("saddle: no mammoth in reach")
		}
	}
}

// handleEmptyHandClick — клик пустой рукой: ракета, фабрика, контракт, дом, сёдла.
// Возвращает true если клик открыл UI и надо выйти из updatePlaying.
func (a *App) handleEmptyHandClick() bool {
	if id := a.rocketInSight(); id != "" {
		if err := a.nc.BoardRocket(id); err != nil {
			a.log.Warn("board rocket", "err", err)
		}
		a.rocketID = id
		a.rocketBoardedAt = time.Now()
		a.log.Info("board rocket sent", "id", id)
		return true
	}
	if id := a.factoryInSight(); id != "" {
		a.ui.factoryID = id
		a.ui.showFactory = true
		a.ui.factoryOpenedAt = time.Now()
		rl.EnableCursor()
		rl.ShowCursor()
		a.log.Info("factory dialog opened", "id", id)
		return true
	}
	if id, pos := a.pinkMobInSight(); id != "" {
		a.ui.contractMobID = id
		a.ui.contractPos = pos
		a.ui.showContract = true
		a.ui.contractOpenedAt = time.Now()
		rl.EnableCursor()
		rl.ShowCursor()
		a.log.Info("contract dialog opened", "mob", id)
		return true
	}
	if id := a.houseInSight(); id != "" {
		if err := a.nc.ToggleDoor(id); err != nil {
			a.log.Warn("toggle door", "err", err)
		}
		a.log.Info("toggle door sent (LMB)", "id", id)
		return false
	}
	if id := a.saddledMammothNearby(); id != "" {
		if err := a.nc.SaddleMammoth(id); err != nil {
			a.log.Warn("saddle mammoth", "err", err)
		}
		a.log.Info("unsaddle mammoth sent (LMB)", "id", id)
	}
	return false
}

// --- Действия с предметами ---

func (a *App) throwSpear() {
	fw := a.flight.Forward()
	dir := protocol.Vector3{X: fw.X, Y: fw.Y, Z: fw.Z}
	if err := a.nc.ThrowSpear(dir); err != nil {
		a.log.Warn("throw spear", "err", err)
	}
	ep := a.nc.EarthPos()
	a.projectiles = append(a.projectiles, projectile{
		pos: rl.NewVector3(
			a.camera.Position.X-ep.X,
			a.camera.Position.Y-ep.Y,
			a.camera.Position.Z-ep.Z,
		),
		dir:   fw,
		spawn: time.Now(),
	})
	a.log.Info("spear thrown (LMB)")
}

func (a *App) useFruit() {
	// Приоритет: мамонт рядом → приручить.
	if id := a.mammothInReach(); id != "" {
		if err := a.nc.TameMammoth(id); err != nil {
			a.log.Warn("tame mammoth", "err", err)
		}
		a.log.Info("tame mammoth sent (LMB)", "id", id)
		return
	}
	// Иначе — посадить семечко в 2 юнитах перед собой.
	fw := a.flight.Forward()
	helio := rl.Vector3Add(a.camera.Position, rl.Vector3Scale(fw, 2.0))
	ep := a.nc.EarthPos()
	pos := rl.NewVector3(helio.X-ep.X, helio.Y-ep.Y, helio.Z-ep.Z)
	if err := a.nc.PlantSeed(pos.X, pos.Y, pos.Z); err != nil {
		a.log.Warn("plant seed", "err", err)
	}
	a.log.Info("plant seed sent (LMB)",
		"geo", fmt.Sprintf("%.1f,%.1f,%.1f", pos.X, pos.Y, pos.Z))
}

func (a *App) useWater() {
	if id := a.seedInSight(); id != "" {
		if err := a.nc.WaterPlant(id); err != nil {
			a.log.Warn("water plant", "err", err)
		}
		a.log.Info("water plant sent (LMB)", "seed", id)
	} else {
		a.log.Debug("water: no seed in sight")
	}
}

func (a *App) useLeash() {
	if id := a.mammothInReach(); id != "" {
		if err := a.nc.LeashMammoth(id); err != nil {
			a.log.Warn("leash mammoth", "err", err)
		}
		a.log.Info("leash mammoth sent (LMB)", "id", id)
	} else {
		a.log.Info("leash: no mammoth in reach")
	}
}

func (a *App) placeStructure(kind string) {
	x, y, z, yaw := a.placeForward(5.0)
	var err error
	switch kind {
	case "house":
		err = a.nc.PlaceHouse(x, y, z, yaw)
	case "solar":
		err = a.nc.PlaceSolar(x, y, z, yaw)
	case "battery":
		err = a.nc.PlaceBattery(x, y, z, yaw)
	case "factory":
		err = a.nc.PlaceFactory(x, y, z, yaw)
	}
	if err != nil {
		a.log.Warn("place "+kind, "err", err)
	}
	a.log.Info("place " + kind + " sent (LMB)")
}

// --- Переходы состояний ---

func (a *App) checkDeath() bool {
	if !a.hpReceived || a.myHP() > 0 {
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

func (a *App) checkPause() bool {
	if !rl.IsKeyPressed(rl.KeyEscape) {
		return false
	}
	if a.flight != nil && a.flight.HelioInit {
		a.pausedRelPos = protocol.Vector3{
			X: a.flight.Pos.X - a.earthPos.X,
			Y: a.flight.Pos.Y - a.earthPos.Y,
			Z: a.flight.Pos.Z - a.earthPos.Z,
		}
		a.pausedRelVel = protocol.Vector3{
			X: a.flight.Vel.X - a.earthVel.X,
			Y: a.flight.Vel.Y - a.earthVel.Y,
			Z: a.flight.Vel.Z - a.earthVel.Z,
		}
		a.log.Info("PAUSE ENTERED",
			"helio", fmt.Sprintf("%.2f,%.2f,%.2f", a.flight.Pos.X, a.flight.Pos.Y, a.flight.Pos.Z),
			"earth", fmt.Sprintf("%.2f,%.2f,%.2f", a.earthPos.X, a.earthPos.Y, a.earthPos.Z),
			"relPos", fmt.Sprintf("%.2f,%.2f,%.2f", a.pausedRelPos.X, a.pausedRelPos.Y, a.pausedRelPos.Z),
			"relVel", fmt.Sprintf("%.3f,%.3f,%.3f", a.pausedRelVel.X, a.pausedRelVel.Y, a.pausedRelVel.Z))
	}
	a.mode = state.ModePaused
	return true
}

// --- Потеря фокуса окна ---

func (a *App) updateUnfocusFreeze() bool {
	if rl.IsWindowFocused() {
		return false
	}
	a.setCursorCaptured(false)
	if a.flight == nil || !a.flight.HelioInit {
		return true
	}
	if !a.unfocusFreeze {
		a.unfocusRelPos = protocol.Vector3{
			X: a.flight.Pos.X - a.earthPos.X,
			Y: a.flight.Pos.Y - a.earthPos.Y,
			Z: a.flight.Pos.Z - a.earthPos.Z,
		}
		a.unfocusRelVel = protocol.Vector3{
			X: a.flight.Vel.X - a.earthVel.X,
			Y: a.flight.Vel.Y - a.earthVel.Y,
			Z: a.flight.Vel.Z - a.earthVel.Z,
		}
		a.unfocusFreeze = true
		a.log.Info("UNFOCUS freeze",
			"relPos", fmt.Sprintf("%.2f,%.2f,%.2f", a.unfocusRelPos.X, a.unfocusRelPos.Y, a.unfocusRelPos.Z))
	}
	a.flight.Pos = rl.NewVector3(
		a.earthPos.X+a.unfocusRelPos.X,
		a.earthPos.Y+a.unfocusRelPos.Y,
		a.earthPos.Z+a.unfocusRelPos.Z,
	)
	a.flight.Vel = rl.NewVector3(
		a.earthVel.X+a.unfocusRelVel.X,
		a.earthVel.Y+a.unfocusRelVel.Y,
		a.earthVel.Z+a.unfocusRelVel.Z,
	)
	fw := a.flight.Forward()
	a.camera.Position = a.flight.Pos
	a.camera.Target = rl.Vector3Add(a.flight.Pos, rl.Vector3Scale(fw, 100.0))
	a.camera.Up = a.flight.CameraUp()
	return true
}

func (a *App) restoreFocus() {
	if !a.unfocusFreeze {
		return
	}
	a.unfocusFreeze = false
	a.log.Info("UNFOCUS restored",
		"helio", fmt.Sprintf("%.2f,%.2f,%.2f", a.flight.Pos.X, a.flight.Pos.Y, a.flight.Pos.Z),
		"earth", fmt.Sprintf("%.2f,%.2f,%.2f", a.earthPos.X, a.earthPos.Y, a.earthPos.Z))
}

// --- Физика и камера ---

func (a *App) updateFlightPhysics(dt float32) {
	a.setCursorCaptured(true)
	if a.flight == nil {
		return
	}

	// HelioInit — только когда EarthPos реально пришла.
	epLen := a.earthPos.X*a.earthPos.X + a.earthPos.Y*a.earthPos.Y + a.earthPos.Z*a.earthPos.Z
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

	relX := a.flight.Pos.X - a.earthPos.X
	relY := a.flight.Pos.Y - a.earthPos.Y
	relZ := a.flight.Pos.Z - a.earthPos.Z
	relDist := float32(math.Sqrt(float64(relX*relX + relY*relY + relZ*relZ)))
	surfaceR := protocol.SurfaceRadius(protocol.Vector3{X: relX, Y: relY, Z: relZ})
	minR := surfaceR + protocol.PlayerHeight
	onGround := relDist <= minR+0.5

	a.log.Info("state",
		"dt", fmt.Sprintf("%.4f", dt),
		"helioPos", fmt.Sprintf("%.2f,%.2f,%.2f", a.flight.Pos.X, a.flight.Pos.Y, a.flight.Pos.Z),
		"earthPos", fmt.Sprintf("%.2f,%.2f,%.2f", a.earthPos.X, a.earthPos.Y, a.earthPos.Z),
		"earthVel", fmt.Sprintf("%.3f,%.3f,%.3f", a.earthVel.X, a.earthVel.Y, a.earthVel.Z),
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

func (a *App) updateCameraSmoothing() {
	fw := a.flight.Forward()
	targetNew := rl.Vector3Add(a.flight.Pos, rl.Vector3Scale(fw, 100.0))

	if !a.camSmoothInit {
		a.camSmoothPos = a.flight.Pos
		a.camSmoothTarget = targetNew
		a.camSmoothInit = true
	} else {
		const camAlpha = float32(0.5)
		a.camSmoothPos = rl.Vector3Add(a.camSmoothPos,
			rl.Vector3Scale(rl.Vector3Subtract(a.flight.Pos, a.camSmoothPos), camAlpha))
		a.camSmoothTarget = rl.Vector3Add(a.camSmoothTarget,
			rl.Vector3Scale(rl.Vector3Subtract(targetNew, a.camSmoothTarget), camAlpha))
	}
	a.camera.Position = a.camSmoothPos
	a.camera.Target = a.camSmoothTarget
	a.camera.Up = a.flight.CameraUp()
}

// --- Отправка состояния на сервер ---

func (a *App) emitPlayerState() {
	if a.flight == nil {
		return
	}
	fw := a.flight.Forward()
	yawF := float32(math.Atan2(float64(-fw.X), float64(-fw.Z)))
	pitchF := float32(math.Asin(float64(fw.Y)))
	a.nc.SetState(protocol.PlayerState{
		X:   a.flight.Pos.X - a.earthPos.X,
		Y:   a.flight.Pos.Y - a.earthPos.Y,
		Z:   a.flight.Pos.Z - a.earthPos.Z,
		Yaw: yawF, Pitch: pitchF,
	})
}
