package app

import (
	"math"

	rl "github.com/gen2brain/raylib-go/raylib"

	"github.com/spacedreamer99/lodeen/internal/client/fonts"
)

func (a *App) drawNavBall() {
	r := a.rocket.hudRocket
	if r.ID == "" {
		return
	}
	if a.diag.debugFrame%60 == 0 {
		a.log.Info("navball draw",
			"id", r.ID,
			"dx", r.DX, "dy", r.DY, "dz", r.DZ,
			"vx", r.VX, "vy", r.VY, "vz", r.VZ)
	}

	sw := int32(rl.GetScreenWidth())
	sh := int32(rl.GetScreenHeight())

	// Навбол — в правом нижнем углу.
	const radius = int32(90)
	cx := sw - radius - 30
	cy := sh - radius - 30

	// Фон.
	rl.DrawCircle(cx, cy, float32(radius), rl.NewColor(20, 30, 50, 220))
	rl.DrawCircleLines(cx, cy, float32(radius), rl.NewColor(120, 180, 240, 255))
	rl.DrawCircleLines(cx, cy, float32(radius)*0.66, rl.NewColor(80, 100, 140, 180))
	rl.DrawCircleLines(cx, cy, float32(radius)*0.33, rl.NewColor(80, 100, 140, 180))
	// Горизонт (горизонтальная линия).
	rl.DrawLine(cx-int32(float32(radius)*0.9), cy, cx+int32(float32(radius)*0.9), cy, rl.NewColor(120, 140, 180, 220))

	// Локальная система координат ракеты:
	// forward = Up ракеты (куда смотрит нос)
	// right = cross(forward, worldUp)
	// trueUp = cross(right, forward)
	fX, fY, fZ := r.DX, r.DY, r.DZ

	// worldUp — нормаль планеты из позиции ракеты.
	pLen := float32(math.Sqrt(float64(r.X*r.X + r.Y*r.Y + r.Z*r.Z)))
	if pLen < 0.01 {
		return
	}
	wX, wY, wZ := r.X/pLen, r.Y/pLen, r.Z/pLen

	// right = cross(forward, worldUp)
	rX := fY*wZ - fZ*wY
	rY := fZ*wX - fX*wZ
	rZ := fX*wY - fY*wX
	rLen := float32(math.Sqrt(float64(rX*rX + rY*rY + rZ*rZ)))
	if rLen < 0.01 {
		return
	}
	rX /= rLen
	rY /= rLen
	rZ /= rLen

	// trueUp = cross(right, forward)
	tuX := rY*fZ - rZ*fY
	tuY := rZ*fX - rX*fZ
	tuZ := rX*fY - rY*fX

	// Функция проекции вектора на 2D навбол.
	project := func(vx, vy, vz float32) (int32, int32, bool) {
		// Компоненты в локальном фрейме.
		fwd := vx*fX + vy*fY + vz*fZ
		rgt := vx*rX + vy*rY + vz*rZ
		upv := vx*tuX + vy*tuY + vz*tuZ
		// Если вектор "за" навболом — не рисуем.
		if fwd < -0.2 {
			return 0, 0, false
		}
		px := cx + int32(rgt*float32(radius))
		py := cy - int32(upv*float32(radius))
		return px, py, true
	}

	// Текущая позиция носа — точка в центре (центр навбола = forward).
	rl.DrawCircle(cx, cy, 3, rl.NewColor(80, 220, 255, 255))

	// Хелпер: нарисовать цель на навболе.
	drawTarget := func(vx, vy, vz float32, col rl.Color, label string) {
		vLen := float32(math.Sqrt(float64(vx*vx + vy*vy + vz*vz)))
		if vLen < 0.01 {
			return
		}
		px, py, ok := project(vx/vLen, vy/vLen, vz/vLen)
		if !ok {
			return
		}
		rl.DrawCircleLines(px, py, 6, col)
		rl.DrawCircleLines(px, py, 7, col)
		if label != "" {
			fonts.Draw(label, px+8, py-8, 12, col)
		}
	}

	// Prograde (по скорости).
	vLen := float32(math.Sqrt(float64(r.VX*r.VX + r.VY*r.VY + r.VZ*r.VZ)))
	if vLen > 0.5 {
		// Prograde — зелёный
		drawTarget(r.VX, r.VY, r.VZ, rl.NewColor(80, 220, 100, 255), "PRO")
		// Retrograde — оранжевый
		drawTarget(-r.VX, -r.VY, -r.VZ, rl.NewColor(230, 140, 40, 255), "RET")
	}

	// Radial out (от планеты) — жёлтый.
	drawTarget(wX, wY, wZ, rl.NewColor(230, 200, 60, 255), "RAD+")
	// Radial in — фиолетовый.
	drawTarget(-wX, -wY, -wZ, rl.NewColor(180, 100, 220, 255), "RAD-")

	// Normal (N) / Antinormal (B) — голубой.
	nx := r.Y*r.VZ - r.Z*r.VY
	ny := r.Z*r.VX - r.X*r.VZ
	nz := r.X*r.VY - r.Y*r.VX
	nl := float32(math.Sqrt(float64(nx*nx + ny*ny + nz*nz)))
	if nl > 0.1 {
		drawTarget(nx/nl, ny/nl, nz/nl, rl.NewColor(100, 180, 255, 255), "NRM")
		drawTarget(-nx/nl, -ny/nl, -nz/nl, rl.NewColor(100, 180, 255, 255), "ANM")
	}

	// Текущий автопилот — под навболом.
	label := "MANUAL"
	col := rl.LightGray
	switch a.rocket.autoPilot {
	case "prograde":
		label = "PROGRADE"
		col = rl.NewColor(80, 220, 100, 255)
	case "retrograde":
		label = "RETROGRADE"
		col = rl.NewColor(230, 140, 40, 255)
	case "radial_out":
		label = "RADIAL+"
		col = rl.NewColor(230, 200, 60, 255)
	case "radial_in":
		label = "RADIAL-"
		col = rl.NewColor(180, 100, 220, 255)
	case "normal":
		label = "NORMAL"
		col = rl.NewColor(100, 180, 255, 255)
	case "antinormal":
		label = "ANTINORMAL"
		col = rl.NewColor(100, 180, 255, 255)
	}
	lw := fonts.Measure(label, 16)
	fonts.Draw(label, cx-lw/2, cy+radius+8, 16, col)
}

func (a *App) drawRocketHUD() {
	if a.player.rocketID == "" {
		return
	}
	r := a.rocket.hudRocket
	if r.ID != a.player.rocketID {
		return
	}

	sw := int32(rl.GetScreenWidth())
	const panelW = int32(360)
	const panelH = int32(200)
	px := sw - panelW - 20
	py := int32(20)

	// Гиперболическая орбита вокруг текущего primary.
	escaping := r.Apoapsis == 0 && r.Altitude > 0 && r.Speed > 5
	leftEarth := r.PrimaryBody == "sun"

	// Панель.
	bg := rl.NewColor(10, 15, 30, 220)
	if leftEarth {
		bg = rl.NewColor(30, 15, 45, 230)
	} else if r.InOrbit {
		bg = rl.NewColor(15, 30, 15, 230)
	} else if escaping {
		bg = rl.NewColor(40, 15, 15, 230)
	}
	panel := rl.NewRectangle(float32(px), float32(py), float32(panelW), float32(panelH))
	rl.DrawRectangleRec(panel, bg)
	border := rl.NewColor(120, 180, 240, 255)
	if leftEarth {
		border = rl.NewColor(200, 120, 240, 255)
	} else if r.InOrbit {
		border = rl.NewColor(80, 220, 100, 255)
	} else if escaping {
		border = rl.NewColor(220, 60, 60, 255)
	}
	rl.DrawRectangleLinesEx(panel, 2, border)

	title := "ROCKET"
	if leftEarth {
		title = "HELIOCENTRIC ORBIT"
	} else if r.InOrbit {
		title = "ORBIT ACHIEVED"
	} else if escaping {
		title = "ESCAPE TRAJECTORY"
	}
	fonts.Draw(title, px+12, py+8, 20, border)

	// Строки.
	pad := int32(12)
	lineY := py + 40
	lineH := int32(20)

	fuelStr := "FUEL: " + itoa(r.Fuel) + " / " + itoa(r.MaxFuel)
	fonts.Draw(fuelStr, px+pad, lineY, 16, rl.RayWhite)

	bodyStr := "EARTH"
	bodyColor := rl.NewColor(100, 180, 240, 255)
	if r.PrimaryBody == "sun" {
		bodyStr = "SUN"
		bodyColor = rl.NewColor(240, 180, 60, 255)
	}
	fonts.Draw("ORBIT: "+bodyStr, px+pad+150, lineY, 16, bodyColor)
	lineY += lineH

	altStr := "ALT: " + itoa(int(r.Altitude)) + " m"
	fonts.Draw(altStr, px+pad, lineY, 16, rl.RayWhite)
	lineY += lineH

	spdStr := "VEL: " + itoa(int(r.Speed)) + " m/s"
	spdColor := rl.RayWhite
	if r.TargetVelocity > 0 {
		spdStr += "  (orbit: " + itoa(int(r.TargetVelocity)) + ")"
		// Цвет: зелёный если 85-115% от target, жёлтый если 50-85% или 115-150%, красный иначе.
		ratio := r.Speed / r.TargetVelocity
		switch {
		case ratio >= 0.85 && ratio <= 1.15:
			spdColor = rl.NewColor(80, 220, 100, 255) // точно
		case ratio >= 0.5 && ratio <= 1.5:
			spdColor = rl.NewColor(230, 200, 60, 255) // близко
		default:
			spdColor = rl.NewColor(220, 60, 60, 255) // не то
		}
	}
	fonts.Draw(spdStr, px+pad, lineY, 16, spdColor)
	lineY += lineH

	apoStr := "APO: " + itoa(int(r.Apoapsis)) + " m"
	periStr := "PER: " + itoa(int(r.Periapsis)) + " m"
	apoColor := rl.LightGray
	periColor := rl.LightGray
	if r.Apoapsis > 0 {
		apoColor = rl.RayWhite
	}
	if r.Periapsis > 60.0 { // atmosphereHeight (dup on client)
		periColor = rl.NewColor(80, 220, 100, 255)
	} else if r.Periapsis > 0 {
		periColor = rl.NewColor(230, 200, 60, 255)
	} else {
		periColor = rl.NewColor(220, 60, 60, 255)
	}
	fonts.Draw(apoStr, px+pad, lineY, 16, apoColor)
	fonts.Draw(periStr, px+pad+140, lineY, 16, periColor)
	lineY += lineH

	// Подсказка.
	hint := "WASD nose | Space thrust | Ctrl retro | M map | E exit"
	hint2 := "Auto: G=prograde  H=retro  J=radial+  K=radial-  N=normal  B=anti  X=off"
	if escaping {
		hint = "TOO FAST — turn sideways, release Space"
	}
	fonts.Draw(hint2, px+pad, py+panelH-40, 11, rl.Gray)
	fonts.Draw(hint, px+pad, py+panelH-24, 12, rl.Gray)
}
