package app

import (
	"github.com/spacedreamer99/lodeen/internal/shared/protocol"
)

// PauseState — состояние паузы (Escape) и потери фокуса окна (Alt+Tab).
// Оба сценария используют одну и ту же технику: заморозить игрока относительно Земли.
type PauseState struct {
	// Сохранённые rel-координаты при входе в паузу.
	pausedRelPos protocol.Vector3
	pausedRelVel protocol.Vector3

	// Заморозка при потере фокуса окна (Alt+Tab).
	unfocusFreeze bool
	unfocusRelPos protocol.Vector3
	unfocusRelVel protocol.Vector3
}
