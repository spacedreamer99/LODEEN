package app

import (
	"fmt"
	"math"
	"time"

	rl "github.com/gen2brain/raylib-go/raylib"

	"github.com/spacedreamer99/lodeen/internal/client/fonts"
	"github.com/spacedreamer99/lodeen/internal/client/input"
	"github.com/spacedreamer99/lodeen/internal/client/ui"
	"github.com/spacedreamer99/lodeen/internal/shared/protocol"
)

func (a *App) drawHUD() {
	a.drawMiniStatus()

	// Индикатор привязки в креативе.
	if a.flight != nil && a.flight.Mode == input.ModeCreative {
		lbl := "FREE"
		col := rl.NewColor(200, 200, 200, 220)
		switch a.flight.AttachedBody {
		case "earth":
			lbl = "ATTACHED: EARTH"
			col = rl.NewColor(100, 180, 240, 240)
		case "sun":
			lbl = "ATTACHED: SUN"
			col = rl.NewColor(240, 180, 60, 240)
		}
		fonts.Draw("FRAME: "+lbl, 30, 210, 16, col)
		fonts.Draw("R — переключить систему отсчёта", 30, 232, 13, rl.NewColor(140, 160, 200, 200))
	}

	const pad = int32(10)

	// FPS — левый верх
	now := time.Now()
	if now.Sub(a.diag.lastFPSAt) > 500*time.Millisecond {
		// Не показываем FPS первые 2 секунды — окно ещё не стабилизировалось.
		if now.Sub(a.startAt) > 2*time.Second {
			a.diag.cachedFPS = rl.GetFPS()
		} else {
			a.diag.cachedFPS = 0
		}
		a.diag.lastFPSAt = now
	}
	if a.diag.cachedFPS > 0 {
		fonts.Draw(fmt.Sprintf("FPS: %d", a.diag.cachedFPS), pad, pad, 18, rl.RayWhite)
	}

	// Полоска голода под FPS
	barX := pad
	barY := pad + 26
	barW := int32(180)
	barH := int32(14)
	hunger := a.nc.Hunger()
	if hunger < 0 {
		hunger = 0
	} else if hunger > 100 {
		hunger = 100
	}
	rl.DrawRectangle(barX, barY, barW, barH, rl.NewColor(40, 40, 40, 220))
	fillW := int32(float32(barW) * hunger / 100)
	if fillW > 0 {
		var col rl.Color
		switch {
		case hunger > 60:
			col = rl.NewColor(90, 200, 90, 240)
		case hunger > 25:
			col = rl.NewColor(230, 180, 60, 240)
		default:
			col = rl.NewColor(210, 70, 70, 240)
		}
		rl.DrawRectangle(barX, barY, fillW, barH, col)
	}
	rl.DrawRectangleLines(barX, barY, barW, barH, rl.NewColor(120, 120, 130, 255))
	fonts.Draw(fmt.Sprintf("Hunger: %.0f", hunger), barX+barW+8, barY, 16, rl.RayWhite)

	// Полоска голода под FPS
	barX = pad
	barY = pad + 26
	barW = int32(180)
	barH = int32(14)
	hunger = a.nc.Hunger()
	if hunger < 0 {
		hunger = 0
	} else if hunger > 100 {
		hunger = 100
	}
	rl.DrawRectangle(barX, barY, barW, barH, rl.NewColor(40, 40, 40, 220))
	fillW = int32(float32(barW) * hunger / 100)
	if fillW > 0 {
		var col rl.Color
		switch {
		case hunger > 60:
			col = rl.NewColor(90, 200, 90, 240)
		case hunger > 25:
			col = rl.NewColor(230, 180, 60, 240)
		default:
			col = rl.NewColor(210, 70, 70, 240)
		}
		rl.DrawRectangle(barX, barY, fillW, barH, col)
	}
	rl.DrawRectangleLines(barX, barY, barW, barH, rl.NewColor(120, 120, 130, 255))
	fonts.Draw(fmt.Sprintf("Hunger: %.0f", hunger), barX+barW+8, barY, 16, rl.RayWhite)

	// RTT — правый верх
	rtt := a.nc.RTT().Milliseconds()
	rtt = rtt / 10 * 10 // округляем до 10 мс — иначе текстура пересоздаётся каждый кадр
	rttText := fmt.Sprintf("RTT: %d ms", rtt)
	rttW := fonts.Measure(rttText, 18)
	rx, ry := ui.Place(ui.TopRight, pad, pad, rttW, 18)
	fonts.Draw(rttText, rx, ry, 18, rl.RayWhite)

	// Players — правый верх, под RTT
	players := len(a.nc.InterpolatedSnapshot())
	plText := fmt.Sprintf("Players: %d", players)
	plW := fonts.Measure(plText, 18)
	px, py := ui.Place(ui.TopRight, pad, pad+22, plW, 18)
	fonts.Draw(plText, px, py, 18, rl.RayWhite)

	// Speed + Mode — правый верх, под Players
	if a.flight != nil {
		spdText := fmt.Sprintf("Speed: %.0f", a.flight.Speed)
		spdW := fonts.Measure(spdText, 18)
		sx, sy := ui.Place(ui.TopRight, pad, pad+44, spdW, 18)
		fonts.Draw(spdText, sx, sy, 18, rl.RayWhite)

		modeName := "Creative"
		if a.flight.Mode == input.ModeSurvival {
			modeName = "Survival"
		}
		modeText := "Mode: " + modeName + "  [F1]"
		modeW := fonts.Measure(modeText, 18)
		mx, my := ui.Place(ui.TopRight, pad, pad+66, modeW, 18)
		fonts.Draw(modeText, mx, my, 18, rl.Yellow)
	}

	// Подсказка подбора — над чатом
	pickupHint := "F - pick up resource   |   T - chat   |   Esc - pause"
	pw := fonts.Measure(pickupHint, 18)
	phx, phy := ui.Place(ui.BottomLeft, pad, pad+70, pw, 18)
	fonts.Draw(pickupHint, phx, phy, 18, rl.Yellow)

	// Help — левый низ
	help := "WASD - move - Space up - Shift down - Q/E roll - Mouse wheel slot/speed"
	hw := fonts.Measure(help, 16)
	hx, hy := ui.Place(ui.BottomLeft, pad, pad+40, hw, 16)
	fonts.Draw(help, hx, hy, 16, rl.Gray)

	// Чат — левый низ
	sw := int(rl.GetScreenWidth())
	sh := int(rl.GetScreenHeight())
	a.chat.Draw(sw, sh-30)

	if a.ui.showInventory {
		a.drawInventory()
	}
	if a.ui.showCraft {
		a.drawCraft()
	}

	// Hotbar внизу по центру — только в Survival
	if a.flight != nil && a.flight.Mode == input.ModeSurvival {
		a.drawHotbar()
	}
	// HP bar.
	hpVal := a.myHP()
	hpBarW := float32(200)
	hpBarH := float32(16)
	hpBarX := int32(20)
	hpBarY := int32(60)
	rl.DrawRectangle(hpBarX, hpBarY, int32(hpBarW), int32(hpBarH), rl.NewColor(60, 20, 20, 200))
	hpFill := hpBarW * float32(hpVal) / 100.0
	if hpFill < 0 {
		hpFill = 0
	}
	rl.DrawRectangle(hpBarX, hpBarY, int32(hpFill), int32(hpBarH), rl.NewColor(200, 40, 40, 255))
	rl.DrawRectangleLines(hpBarX, hpBarY, int32(hpBarW), int32(hpBarH), rl.Black)
	fonts.Draw(fmt.Sprintf("HP: %d", hpVal), hpBarX+int32(hpBarW)+8, hpBarY, 16, rl.RayWhite)
	a.drawDebugOverlay()
}

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

func (a *App) drawDebugOverlay() {
	if !a.diag.showDebug || a.flight == nil {
		return
	}
	sw := int32(rl.GetScreenWidth())
	x := sw - 340
	y := int32(80)
	col := rl.NewColor(200, 220, 255, 220)
	colDim := rl.NewColor(140, 160, 200, 180)

	fonts.Draw("F3 DEBUG", x, y, 16, rl.NewColor(255, 200, 60, 255))
	y += 22

	helioStr := fmt.Sprintf("HELIO %.1f %.1f %.1f",
		a.flight.Pos.X, a.flight.Pos.Y, a.flight.Pos.Z)
	fonts.Draw(helioStr, x, y, 14, col)
	y += 18

	relX := a.flight.Pos.X - a.world.earthPos.X
	relY := a.flight.Pos.Y - a.world.earthPos.Y
	relZ := a.flight.Pos.Z - a.world.earthPos.Z
	relDist := float32(math.Sqrt(float64(relX*relX + relY*relY + relZ*relZ)))
	geoStr := fmt.Sprintf("GEO  %.1f %.1f %.1f  |r|=%.2f",
		relX, relY, relZ, relDist)
	fonts.Draw(geoStr, x, y, 14, col)
	y += 18

	velLen := float32(math.Sqrt(float64(
		a.flight.Vel.X*a.flight.Vel.X +
			a.flight.Vel.Y*a.flight.Vel.Y +
			a.flight.Vel.Z*a.flight.Vel.Z)))
	velStr := fmt.Sprintf("VEL  %.2f (%.2f %.2f %.2f)",
		velLen, a.flight.Vel.X, a.flight.Vel.Y, a.flight.Vel.Z)
	fonts.Draw(velStr, x, y, 14, col)
	y += 18

	surfaceR := protocol.SurfaceRadius(protocol.Vector3{X: relX, Y: relY, Z: relZ})
	minR := surfaceR + protocol.PlayerHeight
	onGround := relDist <= minR+0.5
	stateStr := "AIRBORNE"
	stateCol := rl.NewColor(240, 180, 100, 240)
	if onGround {
		stateStr = "ON GROUND"
		stateCol = rl.NewColor(100, 240, 120, 240)
	}
	stateFull := fmt.Sprintf("%s  minR=%.2f", stateStr, minR)
	fonts.Draw(stateFull, x, y, 14, stateCol)
	y += 18

	modeStr := "SURVIVAL"
	if a.flight.Mode == input.ModeCreative {
		modeStr = "CREATIVE"
		if a.flight.AttachedBody != "" {
			modeStr += "  ATTACHED:" + a.flight.AttachedBody
		}
	}
	fonts.Draw(modeStr, x, y, 14, colDim)
	y += 18

	earthStr := fmt.Sprintf("EARTH %.1f %.1f %.1f",
		a.world.earthPos.X, a.world.earthPos.Y, a.world.earthPos.Z)
	fonts.Draw(earthStr, x, y, 14, colDim)
	y += 18
	earthVelStr := fmt.Sprintf("EVEL %.2f %.2f %.2f",
		a.world.earthVel.X, a.world.earthVel.Y, a.world.earthVel.Z)
	fonts.Draw(earthVelStr, x, y, 14, colDim)
}

func (a *App) drawMiniStatus() {
	if a.flight == nil {
		return
	}

	// Позиция относительно текущего тела.
	bodyName := "EARTH"
	var bodyPos protocol.Vector3
	bodyRadius := protocol.PlanetRadius
	if a.flight.AttachedBody == "sun" {
		bodyName = "SUN"
		bodyPos = protocol.SunPos
		bodyRadius = protocol.SunRadius
	} else {
		bodyPos = a.world.earthPos
	}

	relX := a.flight.Pos.X - bodyPos.X
	relY := a.flight.Pos.Y - bodyPos.Y
	relZ := a.flight.Pos.Z - bodyPos.Z
	relDist := float32(math.Sqrt(float64(relX*relX + relY*relY + relZ*relZ)))

	surfaceR := bodyRadius
	if bodyName == "EARTH" {
		surfaceR = protocol.SurfaceRadius(protocol.Vector3{X: relX, Y: relY, Z: relZ})
	}
	alt := relDist - surfaceR

	// Скорость относительно тела.
	var bodyVel protocol.Vector3
	if bodyName == "EARTH" {
		bodyVel = a.world.earthVel
	}
	relVx := a.flight.Vel.X - bodyVel.X
	relVy := a.flight.Vel.Y - bodyVel.Y
	relVz := a.flight.Vel.Z - bodyVel.Z
	relSpeed := float32(math.Sqrt(float64(relVx*relVx + relVy*relVy + relVz*relVz)))

	// Радиальная скорость (вверх/вниз).
	upX := relX / relDist
	upY := relY / relDist
	upZ := relZ / relDist
	radialVel := relVx*upX + relVy*upY + relVz*upZ

	onGround := alt <= protocol.PlayerHeight+0.5

	// Панель.
	x := int32(20)
	y := int32(120)
	pad := int32(10)
	panelW := int32(220)
	panelH := int32(70)

	bg := rl.NewColor(10, 15, 30, 200)
	rl.DrawRectangle(x, y, panelW, panelH, bg)
	rl.DrawRectangleLines(x, y, panelW, panelH, rl.NewColor(120, 180, 240, 255))

	// Строка 1: тело + статус.
	bodyCol := rl.NewColor(100, 180, 240, 255)
	if bodyName == "SUN" {
		bodyCol = rl.NewColor(240, 180, 60, 255)
	}
	fonts.Draw(bodyName, x+pad, y+pad, 16, bodyCol)

	statusStr := "AIRBORNE"
	statusCol := rl.NewColor(240, 180, 100, 255)
	if onGround {
		statusStr = "ON GROUND"
		statusCol = rl.NewColor(100, 240, 120, 255)
	}
	fonts.Draw(statusStr, x+pad+80, y+pad, 16, statusCol)

	// Строка 2: высота.
	altStr := fmt.Sprintf("ALT %.1f m", alt)
	fonts.Draw(altStr, x+pad, y+pad+22, 14, rl.RayWhite)

	// Строка 3: скорость.
	spdStr := fmt.Sprintf("SPD %.1f m/s", relSpeed)
	fonts.Draw(spdStr, x+pad, y+pad+42, 14, rl.RayWhite)

	// Если в воздухе — показать вертикальную скорость.
	if !onGround {
		vsStr := fmt.Sprintf("V/S %+.1f m/s", radialVel)
		vsCol := rl.NewColor(200, 200, 220, 220)
		if radialVel > 0.5 {
			vsCol = rl.NewColor(100, 240, 120, 240) // вверх
		} else if radialVel < -0.5 {
			vsCol = rl.NewColor(240, 120, 120, 240) // вниз
		}
		fonts.Draw(vsStr, x+pad+100, y+pad+42, 14, vsCol)
	}
}

func (a *App) earthPosAsRl() rl.Vector3 {
	return rl.NewVector3(a.world.earthPos.X, a.world.earthPos.Y, a.world.earthPos.Z)
}
