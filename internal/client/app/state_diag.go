package app

import (
	"time"

	rl "github.com/gen2brain/raylib-go/raylib"
)

// DiagState — состояние диагностики и rate-limited логов.
// Вынесено из App чтобы не засорять главный struct.
type DiagState struct {
	// FPS-сэмплирование (пакет по 60 кадров).
	debugFrame int
	cachedFPS  int32
	lastFPSAt  time.Time

	// F8 diagnostic mode — N кадров детального лога.
	diagFrames int

	// F3 — overlay с координатами.
	showDebug bool

	// Rate-limited state logger: последние значения клавиш/мыши/позиции.
	lastKeys  string
	lastMouse string
	lastLogAt time.Time
	lastPos   rl.Vector3
}
