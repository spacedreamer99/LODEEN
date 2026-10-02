package app

import (
	"time"

	rl "github.com/gen2brain/raylib-go/raylib"

	"github.com/spacedreamer99/lodeen/internal/client/fonts"
	"github.com/spacedreamer99/lodeen/internal/client/ui"
)

func (a *App) updateFactory() {
	if time.Since(a.factoryOpenedAt) < 250*time.Millisecond {
		return
	}
	if rl.IsKeyPressed(rl.KeyEscape) {
		a.showFactory = false
		a.factoryID = ""
		rl.DisableCursor()
	}
}

func (a *App) drawFactory() {
	ignoreClicks := time.Since(a.factoryOpenedAt) < 250*time.Millisecond

	sw := int32(rl.GetScreenWidth())
	sh := int32(rl.GetScreenHeight())

	rl.DrawRectangle(0, 0, sw, sh, rl.Fade(rl.Black, 0.6))

	pw := int32(560)
	ph := int32(int32(len(factoryRecipeList))*80 + 100)
	px := (sw - pw) / 2
	py := (sh - ph) / 2

	panel := rl.NewRectangle(float32(px), float32(py), float32(pw), float32(ph))
	rl.DrawRectangleRec(panel, rl.NewColor(40, 45, 60, 245))
	rl.DrawRectangleLinesEx(panel, 2, rl.NewColor(120, 180, 240, 255))

	title := "FACTORY"
	tw := fonts.Measure(title, 36)
	fonts.Draw(title, px+(pw-tw)/2, py+18, 36, rl.NewColor(150, 200, 255, 255))

	const bw = 520
	const bh = 60
	bx := px + (pw-bw)/2

	for i, r := range factoryRecipeList {
		y := py + 80 + int32(i)*80

		// Текст с требованиями.
		req := r.name + "  ["
		first := true
		for item, q := range r.need {
			if !first {
				req += ", "
			}
			req += item + " x" + itoa(q)
			first = false
		}
		req += "]  E:" + itoa(r.energy)

		btn := ui.Button{
			Rect: rl.NewRectangle(float32(bx), float32(y), bw, bh),
			Text: req,
		}
		btn.Draw()
		if !ignoreClicks && btn.Clicked() {
			if err := a.nc.CraftFactory(a.factoryID, r.id); err != nil {
				a.log.Warn("factory craft", "err", err)
			}
			a.log.Info("factory craft sent", "recipe", r.id)
		}
	}

	esc := "Esc - close"
	escW := fonts.Measure(esc, 16)
	fonts.Draw(esc, px+(pw-escW)/2, py+ph-28, 16, rl.Gray)
}

func itoa(n int) string {
	if n == 0 {
		return "0"
	}
	neg := n < 0
	if neg {
		n = -n
	}
	var buf [20]byte
	i := len(buf)
	for n > 0 {
		i--
		buf[i] = byte('0' + n%10)
		n /= 10
	}
	if neg {
		i--
		buf[i] = '-'
	}
	return string(buf[i:])
}
