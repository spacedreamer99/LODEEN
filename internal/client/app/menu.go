package app

import (
	rl "github.com/gen2brain/raylib-go/raylib"

	"github.com/spacedreamer99/lodeen/internal/client/fonts"
	"github.com/spacedreamer99/lodeen/internal/client/render"
	"github.com/spacedreamer99/lodeen/internal/client/state"
	"github.com/spacedreamer99/lodeen/internal/client/ui"
)

func (a *App) drawMenu() {
	sw := int32(rl.GetScreenWidth())
	sh := int32(rl.GetScreenHeight())

	title := "LODEEN"
	tw := fonts.Measure(title, 56)
	fonts.Draw(title, (sw-tw)/2, 60, 56, rl.RayWhite)

	sub := "federated cooperative multiplayer - pve"
	subW := fonts.Measure(sub, 18)
	fonts.Draw(sub, (sw-subW)/2, 130, 18, rl.Gray)

	const fw = 400
	fx := (sw - fw) / 2
	y := int32(180)

	// Ник
	fonts.Draw("Nick", fx, y, 18, rl.LightGray)
	nickRect := rl.NewRectangle(float32(fx), float32(y+22), fw, 34)
	if ui.TextField(nickRect, a.ui.menuNick, a.ui.menuFocus == 0) {
		a.ui.menuFocus = 0
	}
	y += 76

	// Server
	fonts.Draw("Server", fx, y, 18, rl.LightGray)
	addrRect := rl.NewRectangle(float32(fx), float32(y+22), fw, 34)
	if ui.TextField(addrRect, a.ui.menuAddr, a.ui.menuFocus == 1) {
		a.ui.menuFocus = 1
	}
	y += 76

	// Палитра (HSV picker)
	a.drawColorPicker(fx, y)
	y += 250

	// Connect
	connect := ui.Button{Rect: rl.NewRectangle(float32(fx), float32(y), fw, 42), Text: "Connect"}
	connect.Draw()
	if connect.Clicked() {
		a.startConnect()
	}
	y += 52

	// Fullscreen
	fsLabel := "Fullscreen: Off"
	if rl.IsWindowFullscreen() {
		fsLabel = "Fullscreen: On"
	}
	fs := ui.Button{Rect: rl.NewRectangle(float32(fx), float32(y), fw, 42), Text: fsLabel}
	fs.Draw()
	if fs.Clicked() {
		rl.ToggleFullscreen()
	}
	y += 52

	// Quit
	quit := ui.Button{Rect: rl.NewRectangle(float32(fx), float32(y), fw, 42), Text: "Quit"}
	quit.Draw()
	if quit.Clicked() {
		a.quit = true
	}
	y += 52

	// Ошибка
	if a.ui.menuErr != "" {
		fonts.Draw(a.ui.menuErr, fx, y+6, 16, rl.Red)
	}

	// Подсказка внизу
	hint := "Tab - switch field   Enter - connect"
	hw := fonts.Measure(hint, 14)
	fonts.Draw(hint, (sw-hw)/2, sh-30, 14, rl.DarkGray)
}

// drawColorPicker — сетка 4×4 из 16 фиксированных цветов.
// Под сеткой — preview, название выбранного цвета и hex.
func (a *App) drawColorPicker(labelX, labelY int32) {
	fonts.Draw("Color", labelX, labelY, 18, rl.LightGray)

	const (
		cols = 4
		rows = 4
		cell = int32(40)
		gap  = int32(6)
	)
	gridX := labelX
	gridY := labelY + 26

	mouse := rl.GetMousePosition()
	clicked := rl.IsMouseButtonPressed(rl.MouseLeftButton)

	selectedIdx := -1

	for i, hex := range render.Palette {
		row := int32(i / cols)
		col := int32(i % cols)
		x := gridX + col*(cell+gap)
		y := gridY + row*(cell+gap)

		c, ok := render.HexColor(hex)
		if !ok {
			continue
		}
		rl.DrawRectangle(x, y, cell, cell, c)

		// Hover
		if mouse.X >= float32(x) && mouse.X < float32(x+cell) &&
			mouse.Y >= float32(y) && mouse.Y < float32(y+cell) {
			rl.DrawRectangleLinesEx(
				rl.NewRectangle(float32(x), float32(y), float32(cell), float32(cell)),
				2, rl.White)
			if clicked {
				a.ui.menuColor = hex
			}
		}

		// Маркер выбранного
		if a.ui.menuColor == hex {
			selectedIdx = i
			rl.DrawRectangleLinesEx(
				rl.NewRectangle(float32(x)-2, float32(y)-2, float32(cell)+4, float32(cell)+4),
				3, rl.White)
			rl.DrawRectangleLinesEx(
				rl.NewRectangle(float32(x)-4, float32(y)-4, float32(cell)+8, float32(cell)+8),
				1, rl.Black)
		}
	}

	// Внешняя рамка сетки
	gridW := int32(cols)*(cell+gap) - gap
	gridH := int32(rows)*(cell+gap) - gap
	rl.DrawRectangleLinesEx(
		rl.NewRectangle(float32(gridX)-3, float32(gridY)-3, float32(gridW)+6, float32(gridH)+6),
		1, rl.Gray)

	// Preview + название + hex под сеткой
	prevX := gridX
	prevY := gridY + gridH + 14
	const prevSize = int32(32)

	pc, _ := render.HexColor(a.ui.menuColor)
	rl.DrawRectangle(prevX, prevY, prevSize, prevSize, pc)
	rl.DrawRectangleLinesEx(
		rl.NewRectangle(float32(prevX), float32(prevY), float32(prevSize), float32(prevSize)),
		1, rl.White)

	name := ""
	if selectedIdx >= 0 && selectedIdx < len(render.PaletteNames) {
		name = render.PaletteNames[selectedIdx]
	}
	label := a.ui.menuColor
	if name != "" {
		label = name + "  " + a.ui.menuColor
	}
	fonts.Draw(label, prevX+prevSize+10, prevY+9, 16, rl.LightGray)
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

func fmtHue(h float32) string {
	return fmtItoa(int(h)) + "°"
}

func fmtPct(v float32) string {
	return fmtItoa(int(v*100)) + "%"
}

func fmtItoa(v int) string {
	if v == 0 {
		return "0"
	}
	neg := v < 0
	if neg {
		v = -v
	}
	var b [12]byte
	i := len(b)
	for v > 0 {
		i--
		b[i] = byte('0' + v%10)
		v /= 10
	}
	if neg {
		i--
		b[i] = '-'
	}
	return string(b[i:])
}
