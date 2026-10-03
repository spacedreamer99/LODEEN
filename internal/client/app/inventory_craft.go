package app

import (
	"fmt"

	rl "github.com/gen2brain/raylib-go/raylib"

	"github.com/spacedreamer99/lodeen/internal/client/fonts"
	"github.com/spacedreamer99/lodeen/internal/client/ui"
)

func (a *App) drawCraft() {
	sw := int32(rl.GetScreenWidth())
	sh := int32(rl.GetScreenHeight())
	pw := int32(660)
	ph := int32(120 + 90*len(craftRecipes))
	px := (sw - pw) / 2
	py := (sh - ph) / 2

	rl.DrawRectangle(0, 0, sw, sh, rl.Fade(rl.Black, 0.6))
	panel := rl.NewRectangle(float32(px), float32(py), float32(pw), float32(ph))
	rl.DrawRectangleRec(panel, rl.NewColor(20, 20, 30, 245))
	rl.DrawRectangleLinesEx(panel, 2, rl.NewColor(120, 120, 140, 255))

	fonts.Draw("Crafting   [C / Esc to close]", px+14, py+10, 22, rl.RayWhite)

	inv := a.nc.Inventory()
	rowH := int32(90)
	for i, rec := range craftRecipes {
		cardY := py + 50 + int32(i)*rowH
		card := rl.NewRectangle(float32(px+12), float32(cardY), float32(pw-24), float32(rowH-8))
		rl.DrawRectangleRec(card, rl.NewColor(30, 30, 42, 255))
		rl.DrawRectangleLinesEx(card, 1, rl.NewColor(70, 70, 90, 255))

		// иконка
		var col rl.Color
		switch rec.out {
		case "spear":
			col = rl.NewColor(180, 160, 120, 255)
		case "torch":
			col = rl.NewColor(240, 180, 80, 255)
		default:
			col = rl.White
		}
		rl.DrawRectangleRec(rl.NewRectangle(float32(px+22), float32(cardY+9), 56, 56), col)

		// имя
		fonts.Draw(rec.name, px+94, cardY+8, 22, rl.RayWhite)

		// требования
		reqText := ""
		canCraft := true
		for _, k := range []string{"stone", "wood", "ore", "fruit", "meat", "spear", "torch", "water", "liana", "leash", "house", "saddle", "boat", "solar", "battery", "factory", "steel", "gear", "circuit", "drone", "rocket"} {
			if need, ok := rec.req[k]; ok {
				have := inv[k]
				reqText += fmt.Sprintf("%s %d/%d   ", k, have, need)
				if have < need {
					canCraft = false
				}
			}
		}
		reqCol := rl.NewColor(90, 220, 90, 255)
		if !canCraft {
			reqCol = rl.NewColor(230, 90, 90, 255)
		}
		fonts.Draw(reqText, px+94, cardY+42, 16, reqCol)

		// кнопка
		btn := ui.Button{
			Rect: rl.NewRectangle(float32(px+pw-150), float32(cardY+22), 130, 42),
			Text: "Craft",
		}
		btn.Draw()
		if btn.Clicked() && canCraft {
			_ = a.nc.CraftItem(rec.id)
		}
	}
}
