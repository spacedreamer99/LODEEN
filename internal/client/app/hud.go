package app

import (
	"fmt"
	"time"

	rl "github.com/gen2brain/raylib-go/raylib"

	"github.com/spacedreamer99/lodeen/internal/client/fonts"
	"github.com/spacedreamer99/lodeen/internal/client/input"
	"github.com/spacedreamer99/lodeen/internal/client/ui"
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

func (a *App) earthPosAsRl() rl.Vector3 {
	return rl.NewVector3(a.world.earthPos.X, a.world.earthPos.Y, a.world.earthPos.Z)
}
