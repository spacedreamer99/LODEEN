package app

import (
	"github.com/spacedreamer99/lodeen/internal/shared/protocol"
)

// RocketState — локальное состояние ракеты (HUD, автопилот, нос).
type RocketState struct {
	hudRocket protocol.Rocket
	autoPilot string
	noseX     float32
	noseY     float32
	noseZ     float32
	noseInit  bool
}
