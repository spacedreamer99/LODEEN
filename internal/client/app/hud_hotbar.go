package app

import (
	"fmt"

	rl "github.com/gen2brain/raylib-go/raylib"

	"github.com/spacedreamer99/lodeen/internal/client/fonts"
)

func (a *App) drawHotbar() {
	const slots = 16
	const cell = int32(44)
	const pad3 = int32(6)

	sw := int32(rl.GetScreenWidth())
	sh := int32(rl.GetScreenHeight())
	barW := slots*cell + pad3*2
	barH := cell + pad3*2
	px := (sw - barW) / 2
	py := sh - barH - 20

	panel := rl.NewRectangle(float32(px), float32(py), float32(barW), float32(barH))
	rl.DrawRectangleRec(panel, rl.NewColor(15, 15, 25, 200))
	rl.DrawRectangleLinesEx(panel, 2, rl.NewColor(120, 120, 140, 255))

	inv := a.nc.Inventory()

	for i := 0; i < slots; i++ {
		cx := px + pad3 + int32(i)*cell
		cy := py + pad3
		rect := rl.NewRectangle(float32(cx), float32(cy), float32(cell-2), float32(cell-2))
		rl.DrawRectangleRec(rect, rl.NewColor(30, 30, 40, 255))
		rl.DrawRectangleLinesEx(rect, 1, rl.NewColor(70, 70, 90, 255))

		if i == a.ui.selectedSlot {
			rl.DrawRectangleLinesEx(rect, 3, rl.NewColor(255, 220, 90, 255))
		}

		typ := ""
		if i < len(a.ui.invSlots) {
			typ = a.ui.invSlots[i]
		}
		if typ != "" {
			col := itemColor(typ)
			rl.DrawRectangleRec(rl.NewRectangle(float32(cx+9), float32(cy+9), float32(cell-20), float32(cell-20)), col)
			if n := inv[typ]; n > 0 {
				fonts.Draw(fmt.Sprintf("%d", n), cx+4, cy+cell-20, 14, rl.White)
			}
		}
	}
}
