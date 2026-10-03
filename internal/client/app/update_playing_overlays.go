package app

import (
	rl "github.com/gen2brain/raylib-go/raylib"

	"github.com/spacedreamer99/lodeen/internal/client/input"
)

// updateDialogOverlays — контракт или фабрика: кадр поглощён диалогом.
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

// updateChatOverlay — если чат открыт, поглощаем ввод.
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

// updateOrbitMapOverlay — M переключает карту орбит, при открытой карте поглощаем ввод.
func (a *App) updateOrbitMapOverlay() bool {
	if rl.IsKeyPressed(rl.KeyM) {
		a.orbit.showOrbitMap = !a.orbit.showOrbitMap
		if a.orbit.showOrbitMap {
			a.orbit.orbitInit = false
			rl.EnableCursor()
			rl.ShowCursor()
		} else {
			rl.DisableCursor()
		}
	}
	if a.orbit.showOrbitMap {
		a.updateOrbitMapInput()
		return true
	}
	return false
}

// updateInventoryOverlay — C/I/Esc: окна крафта и инвентаря.
// Возвращает true если UI открыт — остальная игровая логика не работает.
func (a *App) updateInventoryOverlay() bool {
	if rl.IsKeyPressed(rl.KeyC) {
		a.ui.showCraft = true
		a.ui.showInventory = false
		rl.EnableCursor()
		rl.ShowCursor()
	}
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

// handleModeSwitchKey — F1 переключает creative/survival.
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
