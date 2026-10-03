package app

import (
	"time"

	"github.com/spacedreamer99/lodeen/internal/shared/protocol"
)

// WorldState — состояние небесных тел: Земля, planet2, буфер снапшотов.
// Всё в helio (мировые координаты).
type WorldState struct {
	// Земля — позиция и скорость.
	earthPos protocol.Vector3
	earthVel protocol.Vector3

	// Вторая планета.
	planet2Pos protocol.Vector3
	planet2Vel protocol.Vector3

	// Сглаженная позиция для рендера (устаревшее, скоро уберём).
	earthPosSmooth protocol.Vector3
	earthPosInit   bool

	// Буфер снапшотов Земли по тикам сервера (для интерполяции).
	earthHistory []earthSnap
	lastServerEP protocol.Vector3
}

// earthSnap — один снапшот Земли в момент тика сервера.
type earthSnap struct {
	tick uint64
	pos  protocol.Vector3
	vel  protocol.Vector3
	at   time.Time
}
