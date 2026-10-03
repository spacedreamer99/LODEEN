package app

import (
	rl "github.com/gen2brain/raylib-go/raylib"

	"github.com/spacedreamer99/lodeen/internal/client/input"
)

// updateSlotWheel — колесо мыши переключает слот в survival.
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

// syncHeldItem — отправляет SelectItem при смене предмета в руке.
func (a *App) syncHeldItem() {
	held := a.heldItem()
	if held == a.player.lastSentHeld {
		return
	}
	a.player.lastSentHeld = held
	if err := a.nc.SelectItem(held); err != nil {
		a.log.Warn("select item", "err", err)
	}
}

// handleChatKeys — T/слэш открывают чат.
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

// handlePickupKey — F: подобрать ресурс.
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
	case a.player.ridingID != "":
		if err := a.nc.RideMammoth(""); err != nil {
			a.log.Warn("ride", "err", err)
		}
		a.player.ridingID = ""
		a.flight.Riding = false
		a.log.Info("dismount sent")

	case a.player.boatID != "":
		if err := a.nc.EnterBoat(""); err != nil {
			a.log.Warn("boat exit", "err", err)
		}
		a.player.boatID = ""
		a.flight.InBoat = false
		a.log.Info("boat exit sent")

	default:
		if id := a.saddledMammothNearby(); id != "" {
			if err := a.nc.RideMammoth(id); err != nil {
				a.log.Warn("ride", "err", err)
			}
			a.player.ridingID = id
			a.flight.Riding = true
			a.log.Info("mount sent", "id", id)
			return
		}
		if id := a.boatNearby(); id != "" {
			if err := a.nc.EnterBoat(id); err != nil {
				a.log.Warn("boat enter", "err", err)
			}
			a.player.boatID = id
			a.flight.InBoat = true
			a.log.Info("boat enter sent", "id", id)
		}
	}
}

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
