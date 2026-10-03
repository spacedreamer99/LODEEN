package app

import (
	rl "github.com/gen2brain/raylib-go/raylib"

	"github.com/spacedreamer99/lodeen/internal/client/fonts"
	"github.com/spacedreamer99/lodeen/internal/client/state"
	"github.com/spacedreamer99/lodeen/internal/client/ui"
)

func (a *App) drawMenu() {
	sw := int32(rl.GetScreenWidth())
	sh := int32(rl.GetScreenHeight())

	title := "LODEEN"
	tw := fonts.Measure(title, 64)
	fonts.Draw(title, (sw-tw)/2, sh/6, 64, rl.RayWhite)

	sub := "federated cooperative multiplayer - pve"
	subW := fonts.Measure(sub, 20)
	fonts.Draw(sub, (sw-subW)/2, sh/6+80, 20, rl.Gray)

	// Вертикальный центр для формы
	centerY := sh/2 - 50
	const fw = 400
	fx := (sw - fw) / 2

	// Ник
	fonts.Draw("Nick", fx, centerY, 18, rl.LightGray)
	nickRect := rl.NewRectangle(float32(fx), float32(centerY+22), fw, 36)
	if ui.TextField(nickRect, a.ui.menuNick, a.ui.menuFocus == 0) {
		a.ui.menuFocus = 0
	}

	// Server
	fonts.Draw("Server", fx, centerY+76, 18, rl.LightGray)
	addrRect := rl.NewRectangle(float32(fx), float32(centerY+98), fw, 36)
	if ui.TextField(addrRect, a.ui.menuAddr, a.ui.menuFocus == 1) {
		a.ui.menuFocus = 1
	}

	// Connect
	connect := ui.Button{Rect: rl.NewRectangle(float32(fx), float32(centerY+160), fw, 46), Text: "Connect"}
	connect.Draw()
	if connect.Clicked() {
		a.startConnect()
	}

	// Fullscreen
	fsLabel := "Fullscreen: Off"
	if rl.IsWindowFullscreen() {
		fsLabel = "Fullscreen: On"
	}
	fs := ui.Button{Rect: rl.NewRectangle(float32(fx), float32(centerY+220), fw, 46), Text: fsLabel}
	fs.Draw()
	if fs.Clicked() {
		rl.ToggleFullscreen()
	}

	// Quit
	quit := ui.Button{Rect: rl.NewRectangle(float32(fx), float32(centerY+280), fw, 46), Text: "Quit"}
	quit.Draw()
	if quit.Clicked() {
		a.quit = true
	}

	// Ошибка
	if a.ui.menuErr != "" {
		fonts.Draw(a.ui.menuErr, fx, centerY+350, 18, rl.Red)
	}

	// Подсказка внизу по центру
	hint := "Tab - switch field - Enter - connect"
	hw := fonts.Measure(hint, 16)
	fonts.Draw(hint, (sw-hw)/2, sh-40, 16, rl.DarkGray)
}

func (a *App) drawPause() {
	sw := int32(rl.GetScreenWidth())
	sh := int32(rl.GetScreenHeight())

	const pw, ph = 400, 360
	px := float32(sw)/2 - pw/2
	py := float32(sh)/2 - ph/2

	panel := rl.NewRectangle(px, py, pw, ph)
	rl.DrawRectangleRec(panel, rl.NewColor(18, 18, 28, 240))
	rl.DrawRectangleLinesEx(panel, 2, rl.Gray)

	title := "Paused"
	tw := fonts.Measure(title, 36)
	fonts.Draw(title, int32(px)+(pw-tw)/2, int32(py)+20, 36, rl.White)

	resume := ui.Button{Rect: rl.NewRectangle(px+40, py+90, pw-80, 44), Text: "Resume"}
	resume.Draw()
	if resume.Clicked() {
		a.mode = state.ModePlaying
	}

	fsLabel := "Fullscreen: Off"
	if rl.IsWindowFullscreen() {
		fsLabel = "Fullscreen: On"
	}
	fs := ui.Button{Rect: rl.NewRectangle(px+40, py+150, pw-80, 44), Text: fsLabel}
	fs.Draw()
	if fs.Clicked() {
		rl.ToggleFullscreen()
	}

	disc := ui.Button{Rect: rl.NewRectangle(px+40, py+210, pw-80, 44), Text: "Disconnect"}
	disc.Draw()
	if disc.Clicked() {
		a.disconnect()
	}

	quit := ui.Button{Rect: rl.NewRectangle(px+40, py+270, pw-80, 44), Text: "Quit"}
	quit.Draw()
	if quit.Clicked() {
		a.quit = true
	}
}
