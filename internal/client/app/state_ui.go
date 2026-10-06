package app

import (
	"time"

	rl "github.com/gen2brain/raylib-go/raylib"
)

// UIState — состояние всего UI: меню, инвентарь, крафт, контракт, фабрика.
// Cursor захват тоже здесь: он переключается при открытии/закрытии UI.
type UIState struct {
	// Cursor
	cursorCaptured bool

	// Меню (логин + адрес + фокус).
	menuNick   string
	menuAddr   string
	menuColor  string
	menuColorH float32
	menuColorS float32
	menuColorV float32
	menuErr    string
	menuFocus  int

	// Инвентарь + крафт.
	showInventory bool
	showCraft     bool
	selectedSlot  int
	invSlots      [256]string
	dragging      bool
	dragFrom      int

	// Контракт (диалог с розовым мобом).
	showContract     bool
	contractMobID    string
	contractPos      rl.Vector3
	contractOpenedAt time.Time

	// Фабрика (диалог производства).
	showFactory     bool
	factoryID       string
	factoryOpenedAt time.Time
}
