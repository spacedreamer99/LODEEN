package app

import (
	"math"

	rl "github.com/gen2brain/raylib-go/raylib"

	"github.com/spacedreamer99/lodeen/internal/client/fonts"
	"github.com/spacedreamer99/lodeen/internal/shared/protocol"
)

func (a *App) drawOrbitMap() {
	r := a.hudRocket
	if r.ID == "" {
		return
	}

	const seg = 96

	// Инициализация камеры при первом открытии.
	if !a.orbitInit {
		a.orbitInit = true
		a.orbitAzimuth = 0.6
		a.orbitElevation = 0.5
		dist := r.Altitude * 2.5
		if r.PrimaryBody == "sun" {
			dist = protocol.SunDistance * 1.5
		}
		if dist < 200 {
			dist = 200
		}
		if dist > 30000 {
			dist = 30000
		}
		a.orbitDistance = dist
	}

	// Центр карты: orbitFocus или auto (primary body).
	// Позиции в helio-фрейме.
	sunH := rl.NewVector3(protocol.SunPos.X, protocol.SunPos.Y, protocol.SunPos.Z)
	earthH := rl.NewVector3(a.earthPos.X, a.earthPos.Y, a.earthPos.Z)
	rocketH := rl.NewVector3(r.X+a.earthPos.X, r.Y+a.earthPos.Y, r.Z+a.earthPos.Z)

	var center rl.Vector3
	switch a.orbitFocus {
	case "earth":
		center = earthH
	case "sun":
		center = sunH
	case "rocket":
		center = rocketH
	default:
		// Helio-карта: авто всегда на Солнце, чтобы видеть всю систему.
		center = sunH
	}

	// Масштаб: обходим far plane raylib (~1000) — рисуем в scaled-space.
	scale := float32(1.0)
	if a.orbitDistance > 900 {
		scale = a.orbitDistance / 900
	}
	distS := a.orbitDistance / scale
	centerS := rl.NewVector3(center.X/scale, center.Y/scale, center.Z/scale)

	cosEl := float32(math.Cos(float64(a.orbitElevation)))
	camPos := rl.NewVector3(
		centerS.X+distS*cosEl*float32(math.Cos(float64(a.orbitAzimuth))),
		centerS.Y+distS*float32(math.Sin(float64(a.orbitElevation))),
		centerS.Z+distS*cosEl*float32(math.Sin(float64(a.orbitAzimuth))),
	)

	mapCam := rl.Camera3D{
		Position:   camPos,
		Target:     centerS,
		Up:         rl.NewVector3(0, 1, 0),
		Fovy:       60,
		Projection: rl.CameraPerspective,
	}

	// Фон.
	sw := int32(rl.GetScreenWidth())
	sh := int32(rl.GetScreenHeight())
	rl.DrawRectangle(0, 0, sw, sh, rl.NewColor(3, 5, 15, 255))

	rl.BeginMode3D(mapCam)
	rl.PushMatrix()
	rl.Scalef(1.0/scale, 1.0/scale, 1.0/scale)

	// ── Плоскость системы: сетка на Y=0, центр в Солнце, размер ±SunDistance*2 ──
	gridCol := rl.NewColor(40, 60, 90, 100)
	const gridStep = 1000.0
	gridHalf := int(protocol.SunDistance*2/gridStep) + 2
	ext := float32(gridHalf) * gridStep
	sxg := protocol.SunPos.X
	szg := protocol.SunPos.Z
	for i := -gridHalf; i <= gridHalf; i++ {
		off := float32(i) * gridStep
		rl.DrawLine3D(
			rl.NewVector3(sxg+off, 0, szg-ext),
			rl.NewVector3(sxg+off, 0, szg+ext),
			gridCol)
		rl.DrawLine3D(
			rl.NewVector3(sxg-ext, 0, szg+off),
			rl.NewVector3(sxg+ext, 0, szg+off),
			gridCol)
	}

	// ── Орбита Земли вокруг Солнца (яркий круг радиусом SunDistance) ──
	orbitCol := rl.NewColor(80, 200, 140, 220)
	for i := 0; i < seg*2; i++ {
		a0 := float32(i) * 2 * math.Pi / (seg * 2)
		a1 := float32(i+1) * 2 * math.Pi / (seg * 2)
		x0 := sxg + protocol.SunDistance*float32(math.Cos(float64(a0)))
		z0 := szg + protocol.SunDistance*float32(math.Sin(float64(a0)))
		x1 := sxg + protocol.SunDistance*float32(math.Cos(float64(a1)))
		z1 := szg + protocol.SunDistance*float32(math.Sin(float64(a1)))
		rl.DrawLine3D(rl.NewVector3(x0, 0, z0), rl.NewVector3(x1, 0, z1), orbitCol)
		// Толще — двойная линия чуть выше.
		rl.DrawLine3D(rl.NewVector3(x0, 0.5, z0), rl.NewVector3(x1, 0.5, z1), orbitCol)
	}

	// Планета — в helio.
	rl.DrawSphere(earthH, protocol.PlanetRadius, rl.NewColor(50, 80, 130, 255))
	rl.DrawSphereWires(earthH, protocol.PlanetRadius, 16, 16, rl.NewColor(80, 120, 180, 200))
	rl.DrawSphereWires(earthH, protocol.PlanetRadius+50, 16, 16, rl.NewColor(80, 100, 140, 100))

	// Круг SOI Земли (граница patched conics).
	soiCol := rl.NewColor(120, 180, 240, 120)
	for i := 0; i < seg; i++ {
		a0 := float32(i) * 2 * math.Pi / seg
		a1 := float32(i+1) * 2 * math.Pi / seg
		x0 := earthH.X + protocol.SOIEarth*float32(math.Cos(float64(a0)))
		z0 := earthH.Z + protocol.SOIEarth*float32(math.Sin(float64(a0)))
		x1 := earthH.X + protocol.SOIEarth*float32(math.Cos(float64(a1)))
		z1 := earthH.Z + protocol.SOIEarth*float32(math.Sin(float64(a1)))
		rl.DrawLine3D(rl.NewVector3(x0, 0, z0), rl.NewVector3(x1, 0, z1), soiCol)
	}

	// Солнце — точка в мировых координатах.
	sunPos := rl.NewVector3(protocol.SunPos.X, protocol.SunPos.Y, protocol.SunPos.Z)
	rl.DrawSphere(sunPos, protocol.SunRadius, rl.NewColor(255, 220, 100, 255))
	rl.DrawSphereWires(sunPos, protocol.SunRadius, 16, 16, rl.NewColor(255, 180, 60, 220))

	// Линия Солнце → Земля (пунктиром). Следит за Землёй.
	dashCol := rl.NewColor(180, 180, 100, 100)
	const dashes = 48
	for i := 0; i < dashes; i++ {
		t0 := float32(i) / dashes
		t1 := t0 + 0.5/dashes
		if t1 > 1 {
			t1 = 1
		}
		p0 := rl.NewVector3(
			sunPos.X+(earthH.X-sunPos.X)*t0,
			sunPos.Y+(earthH.Y-sunPos.Y)*t0,
			sunPos.Z+(earthH.Z-sunPos.Z)*t0,
		)
		p1 := rl.NewVector3(
			sunPos.X+(earthH.X-sunPos.X)*t1,
			sunPos.Y+(earthH.Y-sunPos.Y)*t1,
			sunPos.Z+(earthH.Z-sunPos.Z)*t1,
		)
		rl.DrawLine3D(p0, p1, dashCol)
	}

	// Оси (для ориентации).
	rl.DrawLine3D(rl.NewVector3(0, 0, 0), rl.NewVector3(protocol.PlanetRadius*2, 0, 0), rl.NewColor(120, 50, 50, 180))
	rl.DrawLine3D(rl.NewVector3(0, 0, 0), rl.NewVector3(0, protocol.PlanetRadius*2, 0), rl.NewColor(50, 120, 50, 180))
	rl.DrawLine3D(rl.NewVector3(0, 0, 0), rl.NewVector3(0, 0, protocol.PlanetRadius*2), rl.NewColor(50, 50, 120, 180))

	// ── Траектория: прошлое + будущее ──
	epT := a.earthPos
	evT := a.earthVel
	helioPos := protocol.Vector3{X: r.X + epT.X, Y: r.Y + epT.Y, Z: r.Z + epT.Z}
	helioVel := protocol.Vector3{X: r.VX + evT.X, Y: r.VY + evT.Y, Z: r.VZ + evT.Z}

	// Будущее — жёлтое. Только для пилотируемой ракеты (иначе predict
	// даёт мусор: ракета "стоит" под огромной гравитацией Земли).
	if r.Piloted {
		dxe := helioPos.X - epT.X
		dye := helioPos.Y - epT.Y
		dze := helioPos.Z - epT.Z
		relEarth := float32(math.Sqrt(float64(dxe*dxe + dye*dye + dze*dze)))

		var dtF float32
		var stepsF int
		if relEarth < protocol.SOIEarth {
			dtF = 0.05
			stepsF = 6000 // 300 сек — орбита вокруг Земли
		} else {
			dtF = 1.0
			stepsF = 2000 // 2000 сек — полный оборот вокруг Солнца
		}
		future := predictTrajectory(helioPos, helioVel, epT, evT, dtF, stepsF)
		futureCol := rl.NewColor(230, 200, 60, 220)
		for i := 1; i < len(future); i++ {
			p1 := rl.NewVector3(future[i-1].X, future[i-1].Y, future[i-1].Z)
			p2 := rl.NewVector3(future[i].X, future[i].Y, future[i].Z)
			rl.DrawLine3D(p1, p2, futureCol)
		}
		if len(future) >= 2 {
			tip := future[len(future)-1]
			rl.DrawSphere(rl.NewVector3(tip.X, tip.Y, tip.Z), 6, futureCol)
			rl.DrawSphereWires(rl.NewVector3(tip.X, tip.Y, tip.Z), 6, 8, 8, rl.White)
		}
	}

	// Прошлое — синее. Тоже только для пилотируемой.
	if r.Piloted {
		dxe := helioPos.X - epT.X
		dye := helioPos.Y - epT.Y
		dze := helioPos.Z - epT.Z
		relEarth := float32(math.Sqrt(float64(dxe*dxe + dye*dye + dze*dze)))

		var dtP float32
		var stepsP int
		if relEarth < protocol.SOIEarth {
			dtP = -0.05
			stepsP = 3000
		} else {
			dtP = -1.0
			stepsP = 1000
		}
		past := predictTrajectory(helioPos, helioVel, epT, evT, dtP, stepsP)
		pastCol := rl.NewColor(70, 130, 220, 200)
		for i := 1; i < len(past); i++ {
			p1 := rl.NewVector3(past[i-1].X, past[i-1].Y, past[i-1].Z)
			p2 := rl.NewVector3(past[i].X, past[i].Y, past[i].Z)
			rl.DrawLine3D(p1, p2, pastCol)
		}
	}

	// Ракета — в helio.
	rp := rocketH
	rl.DrawSphere(rp, 4, rl.NewColor(100, 255, 100, 255))
	rl.DrawSphereWires(rp, 4, 8, 8, rl.White)

	// Вектор скорости (helio).
	hvx := r.VX + a.earthVel.X
	hvy := r.VY + a.earthVel.Y
	hvz := r.VZ + a.earthVel.Z
	vLen := float32(math.Sqrt(float64(hvx*hvx + hvy*hvy + hvz*hvz)))
	if vLen > 0.1 {
		vEnd := rl.NewVector3(
			rp.X+hvx/vLen*30,
			rp.Y+hvy/vLen*30,
			rp.Z+hvz/vLen*30,
		)
		rl.DrawLine3D(rp, vEnd, rl.NewColor(255, 80, 80, 255))
	}

	// Up ракеты.
	upEnd := rl.NewVector3(
		rp.X+r.DX*15,
		rp.Y+r.DY*15,
		rp.Z+r.DZ*15,
	)
	rl.DrawLine3D(rp, upEnd, rl.NewColor(80, 200, 255, 255))

	rl.PopMatrix()
	rl.EndMode3D()

	// Солнце — 2D проекция (обходит far plane).
	sunOv := rl.NewVector3(protocol.SunPos.X/scale, protocol.SunPos.Y/scale, protocol.SunPos.Z/scale)
	swF := float32(sw)
	shF := float32(sh)

	// Проверка «перед камерой»: GetWorldToScreen зеркалит точки за камерой.
	fwdXc := mapCam.Target.X - mapCam.Position.X
	fwdYc := mapCam.Target.Y - mapCam.Position.Y
	fwdZc := mapCam.Target.Z - mapCam.Position.Z
	toXs := sunOv.X - mapCam.Position.X
	toYs := sunOv.Y - mapCam.Position.Y
	toZs := sunOv.Z - mapCam.Position.Z
	sunInFront := fwdXc*toXs+fwdYc*toYs+fwdZc*toZs > 0

	if sunInFront {
		sunScreen := rl.GetWorldToScreen(sunOv, mapCam)
		if sunScreen.X > -200 && sunScreen.X < swF+200 &&
			sunScreen.Y > -200 && sunScreen.Y < shF+200 {
			cx := int32(sunScreen.X)
			cy := int32(sunScreen.Y)
			rl.DrawCircle(cx, cy, 16, rl.NewColor(255, 220, 100, 255))
			rl.DrawCircleLines(cx, cy, 16, rl.NewColor(255, 180, 60, 255))
			rl.DrawCircleLines(cx, cy, 22, rl.NewColor(255, 220, 100, 140))
			rl.DrawCircleLines(cx, cy, 28, rl.NewColor(255, 220, 100, 80))
			fonts.Draw("SUN", cx+20, cy-10, 16, rl.NewColor(255, 220, 100, 255))
			a.sunScrX = float32(cx)
			a.sunScrY = float32(cy)
		}
	}

	// Общие forward-компоненты камеры (для проверки «перед камерой»).
	fwdX := mapCam.Target.X - mapCam.Position.X
	fwdY := mapCam.Target.Y - mapCam.Position.Y
	fwdZ := mapCam.Target.Z - mapCam.Position.Z

	// Земля — 2D проекция.
	earthOv := rl.NewVector3(a.earthPos.X/scale, a.earthPos.Y/scale, a.earthPos.Z/scale)
	toEX := earthOv.X - mapCam.Position.X
	toEY := earthOv.Y - mapCam.Position.Y
	toEZ := earthOv.Z - mapCam.Position.Z
	if fwdX*toEX+fwdY*toEY+fwdZ*toEZ > 0 {
		earthScreen := rl.GetWorldToScreen(earthOv, mapCam)
		if earthScreen.X > -800 && earthScreen.X < swF+800 &&
			earthScreen.Y > -800 && earthScreen.Y < shF+800 {
			cx := int32(earthScreen.X)
			cy := int32(earthScreen.Y)
			distE := rl.Vector3Distance(mapCam.Position, earthOv)
			if distE < 1 {
				distE = 1
			}
			// Проекция радиуса тела на экран: R/dist * (H/2) / tan(FOV/2), FOV=60.
			projR := (protocol.PlanetRadius / scale) / distE * (shF / 2) / 0.5773
			if projR < 5 {
				projR = 5
			}
			if projR > 220 {
				projR = 220
			}
			// SOI как ореол вокруг Земли.
			soiR := (protocol.SOIEarth / scale) / distE * (shF / 2) / 0.5773
			if soiR > projR+4 && soiR < 3000 {
				rl.DrawCircleLines(cx, cy, soiR, rl.NewColor(120, 180, 240, 90))
			}
			// Сохраняем для обработки клика.
			a.earthScrX = float32(cx)
			a.earthScrY = float32(cy)
			a.earthScrR = projR
			// Планета.
			rl.DrawCircle(cx, cy, projR, rl.NewColor(60, 130, 200, 255))
			rl.DrawCircleLines(cx, cy, projR, rl.NewColor(140, 200, 255, 255))
			rl.DrawCircleLines(cx, cy, projR+3, rl.NewColor(80, 140, 200, 160))
			// Подпись.
			lblX := cx + int32(projR) + 6
			lblY := cy - 8
			if lblX > sw-60 {
				lblX = cx - int32(projR) - 60
			}
			fonts.Draw("EARTH", lblX, lblY, 14, rl.NewColor(140, 200, 255, 255))
		}
	}

	// Ракета — 2D проекция.
	rocketOv := rl.NewVector3(
		(r.X+a.earthPos.X)/scale,
		(r.Y+a.earthPos.Y)/scale,
		(r.Z+a.earthPos.Z)/scale,
	)
	toRX := rocketOv.X - mapCam.Position.X
	toRY := rocketOv.Y - mapCam.Position.Y
	toRZ := rocketOv.Z - mapCam.Position.Z
	if fwdX*toRX+fwdY*toRY+fwdZ*toRZ > 0 {
		rocketScreen := rl.GetWorldToScreen(rocketOv, mapCam)
		if rocketScreen.X > -200 && rocketScreen.X < swF+200 &&
			rocketScreen.Y > -200 && rocketScreen.Y < shF+200 {
			cx := int32(rocketScreen.X)
			cy := int32(rocketScreen.Y)
			rl.DrawCircle(cx, cy, 6, rl.NewColor(80, 220, 100, 255))
			rl.DrawCircleLines(cx, cy, 6, rl.NewColor(20, 80, 20, 255))
			rl.DrawCircleLines(cx, cy, 10, rl.NewColor(80, 220, 100, 200))
			rl.DrawCircleLines(cx, cy, 14, rl.NewColor(80, 220, 100, 100))
			a.rocketScrX = float32(cx)
			a.rocketScrY = float32(cy)
		}
	}

	// Оверлей поверх 3D.
	fonts.Draw("ORBITAL MAP", 30, 30, 28, rl.NewColor(150, 200, 255, 255))
	focusLbl := "FOCUS: AUTO (" + r.PrimaryBody + ")"
	switch a.orbitFocus {
	case "earth":
		focusLbl = "FOCUS: EARTH"
	case "sun":
		focusLbl = "FOCUS: SUN"
	case "rocket":
		focusLbl = "FOCUS: ROCKET"
	}
	fonts.Draw(focusLbl, 30, 62, 14, rl.NewColor(150, 200, 255, 200))

	pad := int32(30)
	lineY := pad + 50
	lineH := int32(24)

	fonts.Draw("FUEL: "+itoa(r.Fuel)+" / "+itoa(r.MaxFuel), pad, lineY, 18, rl.RayWhite)
	lineY += lineH

	fonts.Draw("ALT: "+itoa(int(r.Altitude))+" m", pad, lineY, 18, rl.RayWhite)
	lineY += lineH

	fonts.Draw("VEL: "+itoa(int(r.Speed))+" m/s", pad, lineY, 18, rl.RayWhite)
	lineY += lineH

	apoColor := rl.RayWhite
	if r.Apoapsis > 0 {
		apoColor = rl.NewColor(150, 200, 255, 255)
	}
	fonts.Draw("APO: "+itoa(int(r.Apoapsis))+" m", pad, lineY, 18, apoColor)
	lineY += lineH

	periColor := rl.NewColor(220, 60, 60, 255)
	if r.Periapsis > 60 {
		periColor = rl.NewColor(80, 220, 100, 255)
	} else if r.Periapsis > 0 {
		periColor = rl.NewColor(230, 200, 60, 255)
	}
	fonts.Draw("PER: "+itoa(int(r.Periapsis))+" m", pad, lineY, 18, periColor)

	// Подсказка снизу: управление камерой.
	hint := "LMB drag: camera | LMB click: focus | Shift+Wheel: fast zoom | F: cycle focus | M: close"
	hintW := fonts.Measure(hint, 18)
	fonts.Draw(hint, (sw-hintW)/2, sh-40, 18, rl.LightGray)

	// Подсказки автопилота — внизу справа.
	apX := int32(30)
	apY := sh - int32(220)
	fonts.Draw("AUTOPILOT", apX, apY, 16, rl.NewColor(150, 200, 255, 255))
	apY += 22

	rows := []struct {
		key   string
		label string
		col   rl.Color
	}{
		{"G", "prograde", rl.NewColor(80, 220, 100, 255)},
		{"H", "retrograde", rl.NewColor(230, 140, 40, 255)},
		{"J", "radial+", rl.NewColor(230, 200, 60, 255)},
		{"K", "radial-", rl.NewColor(180, 100, 220, 255)},
		{"N", "normal", rl.NewColor(100, 180, 255, 255)},
		{"B", "antinormal", rl.NewColor(100, 180, 255, 255)},
		{"X", "off", rl.LightGray},
	}
	for _, r := range rows {
		line := r.key + "  " + r.label
		fonts.Draw(line, apX+10, apY, 14, r.col)
		apY += 20
	}

	// Текущий автопилот.
	cur := "MANUAL"
	if a.autoPilot != "" {
		cur = a.autoPilot
	}
	fonts.Draw("current: "+cur, apX, apY+8, 16, rl.RayWhite)
}
