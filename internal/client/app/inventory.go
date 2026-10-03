package app

import (
	"fmt"

	rl "github.com/gen2brain/raylib-go/raylib"

	"github.com/spacedreamer99/lodeen/internal/client/fonts"
	"github.com/spacedreamer99/lodeen/internal/client/input"
)

func (a *App) drawInventory() {
	const cols, rows = 16, 16
	const cell = int32(38)
	const pad2 = int32(10)

	sw := int32(rl.GetScreenWidth())
	sh := int32(rl.GetScreenHeight())
	gridW := cols*cell + pad2*2
	gridH := rows*cell + pad2*2 + 30
	px := (sw - gridW) / 2
	py := (sh - gridH) / 2

	rl.DrawRectangle(0, 0, sw, sh, rl.Fade(rl.Black, 0.6))
	panel := rl.NewRectangle(float32(px), float32(py), float32(gridW), float32(gridH))
	rl.DrawRectangleRec(panel, rl.NewColor(20, 20, 30, 245))
	rl.DrawRectangleLinesEx(panel, 2, rl.NewColor(120, 120, 140, 255))

	fonts.Draw("Inventory   [I / Esc close]   LMB: drag item between slots",
		px+pad2, py+6, 20, rl.RayWhite)

	inv := a.nc.Inventory()
	mouse := rl.GetMousePosition()
	gridY := py + 30 + pad2

	var hoveredName string
	var hoveredSlot int = -1

	// Обработка мыши: клик по слоту.
	if rl.IsMouseButtonPressed(rl.MouseLeftButton) {
		// Найти, по какому слоту кликнули.
		for i := 0; i < cols*rows; i++ {
			col := int32(i % cols)
			row := int32(i / cols)
			cx := px + pad2 + col*cell
			cy := gridY + row*cell
			rect := rl.NewRectangle(float32(cx), float32(cy), float32(cell-2), float32(cell-2))
			if rl.CheckCollisionPointRec(mouse, rect) {
				a.ui.dragging = true
				a.ui.dragFrom = i
				break
			}
		}
	}
	if rl.IsMouseButtonReleased(rl.MouseLeftButton) && a.ui.dragging {
		// Куда отпустили.
		for i := 0; i < cols*rows; i++ {
			col := int32(i % cols)
			row := int32(i / cols)
			cx := px + pad2 + col*cell
			cy := gridY + row*cell
			rect := rl.NewRectangle(float32(cx), float32(cy), float32(cell-2), float32(cell-2))
			if rl.CheckCollisionPointRec(mouse, rect) {
				if i != a.ui.dragFrom {
					// Поменять местами.
					a.ui.invSlots[a.ui.dragFrom], a.ui.invSlots[i] = a.ui.invSlots[i], a.ui.invSlots[a.ui.dragFrom]
				}
				break
			}
		}
		a.ui.dragging = false
	}

	for i := 0; i < cols*rows; i++ {
		col := int32(i % cols)
		row := int32(i / cols)
		cx := px + pad2 + col*cell
		cy := gridY + row*cell
		rect := rl.NewRectangle(float32(cx), float32(cy), float32(cell-2), float32(cell-2))

		bg := rl.NewColor(32, 32, 44, 255)
		if row == 0 {
			bg = rl.NewColor(60, 50, 30, 255)
		}
		rl.DrawRectangleRec(rect, bg)
		rl.DrawRectangleLinesEx(rect, 1, rl.NewColor(70, 70, 90, 255))

		// Рамка выбранного слота.
		if row == 0 && int(col) == a.ui.selectedSlot {
			rl.DrawRectangleLinesEx(rect, 3, rl.NewColor(255, 220, 90, 255))
		}

		// Подсветка слота, если тащим предмет и наводим на этот слот.
		if a.ui.dragging && rl.CheckCollisionPointRec(mouse, rect) {
			rl.DrawRectangleLinesEx(rect, 2, rl.NewColor(80, 200, 120, 255))
			hoveredSlot = i
		}

		typ := a.ui.invSlots[i]
		if typ == "" {
			continue
		}
		n := inv[typ]
		col2 := itemColor(typ)
		icon := rl.NewRectangle(float32(cx+8), float32(cy+8), float32(cell-18), float32(cell-18))
		rl.DrawRectangleRec(icon, col2)
		fonts.Draw(fmt.Sprintf("%d", n), cx+3, cy+cell-18, 14, rl.White)

		if rl.CheckCollisionPointRec(mouse, rect) {
			hoveredName = fmt.Sprintf("%s  x%d", itemName(typ), n)
		}
	}

	// Призрак перетаскиваемого предмета под курсором.
	if a.ui.dragging && a.ui.invSlots[a.ui.dragFrom] != "" {
		typ := a.ui.invSlots[a.ui.dragFrom]
		col2 := itemColor(typ)
		gx := int32(mouse.X) - cell/2 + 2
		gy := int32(mouse.Y) - cell/2 + 2
		ghost := rl.NewRectangle(float32(gx), float32(gy), float32(cell-4), float32(cell-4))
		rl.DrawRectangleRec(ghost, rl.NewColor(col2.R, col2.G, col2.B, 200))
		rl.DrawRectangleLinesEx(ghost, 2, rl.White)
		fonts.Draw(itemName(typ), gx+4, gy+cell-20, 14, rl.White)
	}

	// Tooltip.
	if hoveredName != "" && !a.ui.dragging {
		tw := fonts.Measure(hoveredName, 16)
		tx := int32(mouse.X) + 18
		ty := int32(mouse.Y) + 14
		if tx+tw+20 > sw {
			tx = int32(mouse.X) - tw - 22
		}
		if ty+30 > sh {
			ty = sh - 34
		}
		box := rl.NewRectangle(float32(tx-6), float32(ty-4), float32(tw+12), 24)
		rl.DrawRectangleRec(box, rl.NewColor(10, 10, 20, 245))
		rl.DrawRectangleLinesEx(box, 1, rl.NewColor(200, 200, 220, 255))
		fonts.Draw(hoveredName, tx, ty, 16, rl.White)
	}
	_ = hoveredSlot
}

func (a *App) drawHeldItem() {
	if a.flight == nil || a.flight.Mode != input.ModeSurvival {
		return
	}
	typ := a.heldItem()
	if typ == "" {
		return
	}
	col := itemColor(typ)

	// Локальные оси камеры
	fw := rl.Vector3Normalize(rl.Vector3Subtract(a.camera.camera.Target, a.camera.camera.Position))
	right := rl.Vector3Normalize(rl.Vector3CrossProduct(fw, a.camera.camera.Up))
	up := rl.Vector3Normalize(rl.Vector3CrossProduct(right, fw))

	// Позиция в правом нижнем углу, чуть впереди
	pos := a.camera.camera.Position
	pos = rl.Vector3Add(pos, rl.Vector3Scale(fw, 0.7))
	pos = rl.Vector3Add(pos, rl.Vector3Scale(right, 0.4))
	pos = rl.Vector3Subtract(pos, rl.Vector3Scale(up, 0.35))

	size := float32(0.25)
	rl.DrawCube(pos, size, size, size, col)
	rl.DrawCubeWires(pos, size, size, size, rl.Black)
}

func (a *App) myHP() int {
	return a.player.hp
}

func (a *App) heldItem() string {
	if a.ui.selectedSlot < 0 || a.ui.selectedSlot >= len(a.ui.invSlots) {
		return ""
	}
	return a.ui.invSlots[a.ui.selectedSlot]
}

func (a *App) syncInvSlots() {
	inv := a.nc.Inventory()

	// Какие типы уже лежат в слотах.
	present := make(map[string]bool, len(a.ui.invSlots))
	for _, t := range a.ui.invSlots {
		if t != "" {
			present[t] = true
		}
	}

	// Новые предметы → в первый пустой слот.
	for t, n := range inv {
		if n <= 0 || present[t] {
			continue
		}
		for i := 0; i < len(a.ui.invSlots); i++ {
			if a.ui.invSlots[i] == "" {
				a.ui.invSlots[i] = t
				present[t] = true
				break
			}
		}
	}

	// Очистить слоты, где предмет пропал.
	for i, t := range a.ui.invSlots {
		if t != "" && inv[t] <= 0 {
			a.ui.invSlots[i] = ""
		}
	}
}
