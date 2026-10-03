package app

import (
	rl "github.com/gen2brain/raylib-go/raylib"
)

// CameraState — камера и её сглаживание + интерполяция по тикам сервера.
type CameraState struct {
	// Основная 3D-камера raylib.
	camera rl.Camera3D

	// Сглаживание камеры (lerp к flight.Pos).
	camSmoothPos    rl.Vector3
	camSmoothTarget rl.Vector3
	camSmoothInit   bool

	// Интерполяция по тикам (renderTick в единицах tick).
	renderTick     float64
	renderTickInit bool
	tickRate       float64
}
