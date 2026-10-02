package app

import (
	"time"

	rl "github.com/gen2brain/raylib-go/raylib"

	"github.com/spacedreamer99/lodeen/internal/client/fonts"
	"github.com/spacedreamer99/lodeen/internal/client/ui"
)

func (a *App) updateContract() {
	if time.Since(a.contractOpenedAt) < 250*time.Millisecond {
		return
	}
	if rl.IsKeyPressed(rl.KeyEscape) {
		a.showContract = false
		a.contractMobID = ""
		rl.DisableCursor()
		return
	}
}

func (a *App) drawContract() {
	ignoreClicks := time.Since(a.contractOpenedAt) < 250*time.Millisecond
	sw := int32(rl.GetScreenWidth())
	sh := int32(rl.GetScreenHeight())

	// Затемнение
	rl.DrawRectangle(0, 0, sw, sh, rl.Fade(rl.Black, 0.55))

	// Панель
	pw := int32(520)
	ph := int32(300)
	px := (sw - pw) / 2
	py := (sh - ph) / 2
	panel := rl.NewRectangle(float32(px), float32(py), float32(pw), float32(ph))
	rl.DrawRectangleRec(panel, rl.NewColor(40, 40, 50, 245))
	rl.DrawRectangleLinesEx(panel, 2, rl.NewColor(240, 130, 175, 255))

	title := "HELPER"
	tw := fonts.Measure(title, 36)
	fonts.Draw(title, px+(pw-tw)/2, py+20, 36, rl.NewColor(240, 130, 175, 255))

	// Кнопки
	const bw = 460
	const bh = 60
	bx := px + (pw-bw)/2

	g1 := ui.Button{
		Rect: rl.NewRectangle(float32(bx), float32(py+100), bw, bh),
		Text: "Give 1 fruit -> brings 4 resources",
	}
	g1.Draw()
	if !ignoreClicks && g1.Clicked() {
		_ = a.nc.AcceptContract(a.contractMobID, "gather4")
		a.showContract = false
		a.contractMobID = ""
		rl.DisableCursor()
	}

	g2 := ui.Button{
		Rect: rl.NewRectangle(float32(bx), float32(py+170), bw, bh),
		Text: "Give 10 spears -> guard me",
	}
	g2.Draw()
	if !ignoreClicks && g2.Clicked() {
		_ = a.nc.AcceptContract(a.contractMobID, "guard")
		a.showContract = false
		a.contractMobID = ""
		rl.DisableCursor()
	}

}
