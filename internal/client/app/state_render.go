package app

import (
	rl "github.com/gen2brain/raylib-go/raylib"
)

// RenderTarget — offscreen UI-буфер (рисуем UI в него, потом 1:1 в окно).
type RenderTarget struct {
	tex rl.RenderTexture2D
	w   int32
	h   int32
}
