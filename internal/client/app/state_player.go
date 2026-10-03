package app

import (
	"time"

	"github.com/spacedreamer99/lodeen/internal/shared/protocol"
)

// PlayerState — состояние локального игрока: HP, маунт, лодка, локальные снаряды.
// Плюс UI-lock флаг — игрок прибит к Земле пока открыт UI.
type PlayerState struct {
	hp         int
	hpReceived bool

	// Транспорт: мамонт / лодка / ракета.
	ridingID        string
	boatID          string
	rocketID        string
	rocketBoardedAt time.Time

	// Что в руке у сервера (не пересылать повторно).
	lastSentHeld string

	// Локальные снаряды (копьё) — визуализация, hit-detection на клиенте.
	projectiles []projectile

	// UI-lock: при открытом UI игрок жёстко привязан к Земле по rel-координатам.
	flightLocked       bool
	flightLockedRelPos protocol.Vector3
	flightLockedRelVel protocol.Vector3
}
