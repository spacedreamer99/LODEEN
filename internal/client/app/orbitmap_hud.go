package app

import (
	rl "github.com/gen2brain/raylib-go/raylib"

	"github.com/spacedreamer99/lodeen/internal/client/fonts"
	"github.com/spacedreamer99/lodeen/internal/shared/protocol"
)

// drawOrbitHUD — вся текстовая инфа поверх карты.
func (a *App) drawOrbitHUD(r protocol.Rocket, sw, sh int32) {
	a.drawOrbitHeader(r)
	a.drawOrbitTelemetry(r)
	a.drawOrbitHint(sw, sh)
	a.drawOrbitAutopilot(sh)
}

func (a *App) drawOrbitHeader(r protocol.Rocket) {
	fonts.Draw("ORBITAL MAP", 30, 30, 28, rl.NewColor(150, 200, 255, 255))

	focusLbl := "FOCUS: AUTO (" + r.PrimaryBody + ")"
	switch a.orbit.orbitFocus {
	case "earth":
		focusLbl = "FOCUS: EARTH"
	case "sun":
		focusLbl = "FOCUS: SUN"
	case "rocket":
		focusLbl = "FOCUS: ROCKET"
	}
	fonts.Draw(focusLbl, 30, 62, 14, rl.NewColor(150, 200, 255, 200))
}

func (a *App) drawOrbitTelemetry(r protocol.Rocket) {
	pad := int32(30)
	y := pad + 50
	const lh = int32(24)

	fonts.Draw("FUEL: "+itoa(r.Fuel)+" / "+itoa(r.MaxFuel), pad, y, 18, rl.RayWhite)
	y += lh

	fonts.Draw("ALT: "+itoa(int(r.Altitude))+" m", pad, y, 18, rl.RayWhite)
	y += lh

	fonts.Draw("VEL: "+itoa(int(r.Speed))+" m/s", pad, y, 18, rl.RayWhite)
	y += lh

	apoCol := rl.RayWhite
	if r.Apoapsis > 0 {
		apoCol = rl.NewColor(150, 200, 255, 255)
	}
	fonts.Draw("APO: "+itoa(int(r.Apoapsis))+" m", pad, y, 18, apoCol)
	y += lh

	periCol := rl.NewColor(220, 60, 60, 255)
	if r.Periapsis > 60 {
		periCol = rl.NewColor(80, 220, 100, 255)
	} else if r.Periapsis > 0 {
		periCol = rl.NewColor(230, 200, 60, 255)
	}
	fonts.Draw("PER: "+itoa(int(r.Periapsis))+" m", pad, y, 18, periCol)
}

func (a *App) drawOrbitHint(sw, sh int32) {
	hint := "LMB drag: camera | LMB click: focus | Shift+Wheel: fast zoom | F: cycle focus | M: close"
	w := fonts.Measure(hint, 18)
	fonts.Draw(hint, (sw-w)/2, sh-40, 18, rl.LightGray)
}

func (a *App) drawOrbitAutopilot(sh int32) {
	x := int32(30)
	y := sh - int32(220)

	fonts.Draw("AUTOPILOT", x, y, 16, rl.NewColor(150, 200, 255, 255))
	y += 22

	rows := []struct {
		key, label string
		col        rl.Color
	}{
		{"G", "prograde", rl.NewColor(80, 220, 100, 255)},
		{"H", "retrograde", rl.NewColor(230, 140, 40, 255)},
		{"J", "radial+", rl.NewColor(230, 200, 60, 255)},
		{"K", "radial-", rl.NewColor(180, 100, 220, 255)},
		{"N", "normal", rl.NewColor(100, 180, 255, 255)},
		{"B", "antinormal", rl.NewColor(100, 180, 255, 255)},
		{"X", "off", rl.LightGray},
	}
	for _, rr := range rows {
		fonts.Draw(rr.key+"  "+rr.label, x+10, y, 14, rr.col)
		y += 20
	}

	cur := "MANUAL"
	if a.rocket.autoPilot != "" {
		cur = a.rocket.autoPilot
	}
	fonts.Draw("current: "+cur, x, y+8, 16, rl.RayWhite)
}
