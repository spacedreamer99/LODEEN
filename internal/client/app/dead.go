package app

import (
	rl "github.com/gen2brain/raylib-go/raylib"

	"github.com/spacedreamer99/lodeen/internal/client/fonts"
	"github.com/spacedreamer99/lodeen/internal/client/state"
	"github.com/spacedreamer99/lodeen/internal/client/ui"
)

func (a *App) updateDead() {
	if rl.IsKeyPressed(rl.KeyR) {
		a.respawn()
	}
	if rl.IsKeyPressed(rl.KeyEscape) {
		a.returnToMenu()
	}
}

func (a *App) respawn() {
	a.log.Info("respawn")
	a.hp = 100
	a.hpReceived = false
	a.startConnect()
	rl.DisableCursor()
}

func (a *App) returnToMenu() {
	a.log.Info("return to menu")
	a.hp = 100
	a.hpReceived = false
	a.mode = state.ModeMenu
	rl.EnableCursor()
	rl.ShowCursor()
}

func (a *App) drawDead() {
	sw := int32(rl.GetScreenWidth())
	sh := int32(rl.GetScreenHeight())

	// Затемнение на весь экран.
	rl.DrawRectangle(0, 0, sw, sh, rl.Fade(rl.Black, 0.78))

	// Заголовок.
	title := "YOUR CHARACTER DIED"
	tw := fonts.Measure(title, 72)
	fonts.Draw(title, (sw-tw)/2, sh/3, 72, rl.NewColor(220, 40, 40, 255))

	// Кнопки.
	const bw = 300
	const bh = 54
	bx := (sw - bw) / 2

	respRect := rl.NewRectangle(float32(bx), float32(sh/2+60), bw, bh)
	respBtn := ui.Button{Rect: respRect, Text: "Respawn  [R]"}
	respBtn.Draw()
	if respBtn.Clicked() {
		a.respawn()
	}

	menuRect := rl.NewRectangle(float32(bx), float32(sh/2+130), bw, bh)
	menuBtn := ui.Button{Rect: menuRect, Text: "Main Menu  [Esc]"}
	menuBtn.Draw()
	if menuBtn.Clicked() {
		a.returnToMenu()
	}
}
