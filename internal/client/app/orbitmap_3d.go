package app

import (
	"math"

	rl "github.com/gen2brain/raylib-go/raylib"

	"github.com/spacedreamer99/lodeen/internal/shared/protocol"
)

// drawOrbit3DScene — единая точка входа для отрисовки 3D-сцены карты.
func (a *App) drawOrbit3DScene(r protocol.Rocket, haveRocket bool, sunH, earthH, star2H, planet2H, rocketH rl.Vector3) {
	a.drawOrbitGrids(sunH, star2H)
	drawOrbitCircle(sunH, protocol.SunDistance, rl.NewColor(80, 200, 140, 220), true)
	drawOrbitCircle(star2H, protocol.Planet2OrbitRadius, rl.NewColor(255, 140, 60, 180), true)
	drawOrbitCircle(earthH, protocol.SOIEarth, rl.NewColor(120, 180, 240, 120), false)
	drawOrbitCircle(planet2H, protocol.SOIPlanet2, rl.NewColor(120, 200, 220, 120), false)

	a.drawOrbitBodies(sunH, earthH, star2H, planet2H)
	drawOrbitSunEarthLine(sunH, earthH)
	drawOrbitAxes()

	if haveRocket {
		a.drawOrbitTrajectory(r)
		drawOrbitRocket3D(r, rocketH, a.world.earthVel)
	}
	a.drawOrbitPlayers()
}

// drawOrbitGrids — две координатные сетки: вокруг Солнца и вокруг Star2.
func (a *App) drawOrbitGrids(sunH, star2H rl.Vector3) {
	// Система 2 (Star2): мелкая сетка.
	gridCol2 := rl.NewColor(80, 50, 30, 100)
	const gridStep2 = 500.0
	gridHalf2 := int(protocol.Planet2OrbitRadius*2/gridStep2) + 2
	ext2 := float32(gridHalf2) * gridStep2
	for i := -gridHalf2; i <= gridHalf2; i++ {
		off := float32(i) * gridStep2
		rl.DrawLine3D(
			rl.NewVector3(star2H.X+off, 0, star2H.Z-ext2),
			rl.NewVector3(star2H.X+off, 0, star2H.Z+ext2),
			gridCol2)
		rl.DrawLine3D(
			rl.NewVector3(star2H.X-ext2, 0, star2H.Z+off),
			rl.NewVector3(star2H.X+ext2, 0, star2H.Z+off),
			gridCol2)
	}
	// Система 1 (Солнце): основная сетка.
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
}

// drawOrbitCircle — круг вокруг центра в плоскости Y=0.
// double=true → двойная линия (для орбит), false → одна (для SOI).
func drawOrbitCircle(center rl.Vector3, radius float32, col rl.Color, double bool) {
	const seg = 96
	steps := seg
	if double {
		steps = seg * 2
	}
	for i := 0; i < steps; i++ {
		a0 := float32(i) * 2 * math.Pi / float32(steps)
		a1 := float32(i+1) * 2 * math.Pi / float32(steps)
		x0 := center.X + radius*float32(math.Cos(float64(a0)))
		z0 := center.Z + radius*float32(math.Sin(float64(a0)))
		x1 := center.X + radius*float32(math.Cos(float64(a1)))
		z1 := center.Z + radius*float32(math.Sin(float64(a1)))
		rl.DrawLine3D(rl.NewVector3(x0, 0, z0), rl.NewVector3(x1, 0, z1), col)
		if double {
			rl.DrawLine3D(rl.NewVector3(x0, 0.5, z0), rl.NewVector3(x1, 0.5, z1), col)
		}
	}
}

// drawOrbitBodies — сферы всех тел в helio.
func (a *App) drawOrbitBodies(sunH, earthH, star2H, planet2H rl.Vector3) {
	// Земля — синяя + ореол SOI.
	rl.DrawSphere(earthH, protocol.PlanetRadius, rl.NewColor(50, 80, 130, 255))
	rl.DrawSphereWires(earthH, protocol.PlanetRadius, 16, 16, rl.NewColor(80, 120, 180, 200))
	rl.DrawSphereWires(earthH, protocol.PlanetRadius+50, 16, 16, rl.NewColor(80, 100, 140, 100))

	// Солнце — жёлтая точка.
	rl.DrawSphere(sunH, protocol.SunRadius, rl.NewColor(255, 220, 100, 255))
	rl.DrawSphereWires(sunH, protocol.SunRadius, 16, 16, rl.NewColor(255, 180, 60, 220))

	// Star2 — оранжевый карлик.
	rl.DrawSphere(star2H, protocol.Star2Radius, rl.NewColor(255, 140, 60, 255))
	rl.DrawSphereWires(star2H, protocol.Star2Radius, 16, 16, rl.NewColor(200, 80, 30, 220))

	// Planet2 — голубая.
	rl.DrawSphere(planet2H, protocol.Planet2Radius, rl.NewColor(80, 180, 200, 255))
	rl.DrawSphereWires(planet2H, protocol.Planet2Radius, 16, 16, rl.NewColor(40, 120, 160, 220))
}

// drawOrbitSunEarthLine — пунктирная линия Солнце → Земля.
func drawOrbitSunEarthLine(sun, earth rl.Vector3) {
	col := rl.NewColor(180, 180, 100, 100)
	const dashes = 48
	for i := 0; i < dashes; i++ {
		t0 := float32(i) / dashes
		t1 := t0 + 0.5/dashes
		if t1 > 1 {
			t1 = 1
		}
		p0 := rl.NewVector3(
			sun.X+(earth.X-sun.X)*t0,
			sun.Y+(earth.Y-sun.Y)*t0,
			sun.Z+(earth.Z-sun.Z)*t0,
		)
		p1 := rl.NewVector3(
			sun.X+(earth.X-sun.X)*t1,
			sun.Y+(earth.Y-sun.Y)*t1,
			sun.Z+(earth.Z-sun.Z)*t1,
		)
		rl.DrawLine3D(p0, p1, col)
	}
}

// drawOrbitAxes — 3 оси для ориентации.
func drawOrbitAxes() {
	r := protocol.PlanetRadius * 2
	rl.DrawLine3D(rl.NewVector3(0, 0, 0), rl.NewVector3(r, 0, 0), rl.NewColor(120, 50, 50, 180))
	rl.DrawLine3D(rl.NewVector3(0, 0, 0), rl.NewVector3(0, r, 0), rl.NewColor(50, 120, 50, 180))
	rl.DrawLine3D(rl.NewVector3(0, 0, 0), rl.NewVector3(0, 0, r), rl.NewColor(50, 50, 120, 180))
}

// drawOrbitTrajectory — предсказание траектории ракеты (future + past).
// Только для Piloted (иначе predict даёт мусор).
func (a *App) drawOrbitTrajectory(r protocol.Rocket) {
	if !r.Piloted {
		return
	}
	epT := a.world.earthPos
	evT := a.world.earthVel
	helioPos := protocol.Vector3{X: r.X + epT.X, Y: r.Y + epT.Y, Z: r.Z + epT.Z}
	helioVel := protocol.Vector3{X: r.VX + evT.X, Y: r.VY + evT.Y, Z: r.VZ + evT.Z}

	// Будущее — жёлтое.
	{
		dxe := helioPos.X - epT.X
		dye := helioPos.Y - epT.Y
		dze := helioPos.Z - epT.Z
		relEarth := float32(math.Sqrt(float64(dxe*dxe + dye*dye + dze*dze)))

		var dtF float32
		var stepsF int
		if relEarth < protocol.SOIEarth {
			dtF = 0.05
			stepsF = 6000
		} else {
			dtF = 1.0
			stepsF = 2000
		}
		future := predictTrajectory(helioPos, helioVel, epT, evT, dtF, stepsF)
		col := rl.NewColor(230, 200, 60, 220)
		for i := 1; i < len(future); i++ {
			p1 := rl.NewVector3(future[i-1].X, future[i-1].Y, future[i-1].Z)
			p2 := rl.NewVector3(future[i].X, future[i].Y, future[i].Z)
			rl.DrawLine3D(p1, p2, col)
		}
		if len(future) >= 2 {
			tip := future[len(future)-1]
			rl.DrawSphere(rl.NewVector3(tip.X, tip.Y, tip.Z), 6, col)
			rl.DrawSphereWires(rl.NewVector3(tip.X, tip.Y, tip.Z), 6, 8, 8, rl.White)
		}
	}

	// Прошлое — синее.
	{
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
		col := rl.NewColor(70, 130, 220, 200)
		for i := 1; i < len(past); i++ {
			p1 := rl.NewVector3(past[i-1].X, past[i-1].Y, past[i-1].Z)
			p2 := rl.NewVector3(past[i].X, past[i].Y, past[i].Z)
			rl.DrawLine3D(p1, p2, col)
		}
	}
}

// drawOrbitRocket3D — ракета точкой + вектора velocity и up.
func drawOrbitRocket3D(r protocol.Rocket, rocketH rl.Vector3, earthVel protocol.Vector3) {
	rl.DrawSphere(rocketH, 4, rl.NewColor(100, 255, 100, 255))
	rl.DrawSphereWires(rocketH, 4, 8, 8, rl.White)

	hvx := r.VX + earthVel.X
	hvy := r.VY + earthVel.Y
	hvz := r.VZ + earthVel.Z
	vLen := float32(math.Sqrt(float64(hvx*hvx + hvy*hvy + hvz*hvz)))
	if vLen > 0.1 {
		vEnd := rl.NewVector3(
			rocketH.X+hvx/vLen*30,
			rocketH.Y+hvy/vLen*30,
			rocketH.Z+hvz/vLen*30,
		)
		rl.DrawLine3D(rocketH, vEnd, rl.NewColor(255, 80, 80, 255))
	}

	upEnd := rl.NewVector3(
		rocketH.X+r.DX*15,
		rocketH.Y+r.DY*15,
		rocketH.Z+r.DZ*15,
	)
	rl.DrawLine3D(rocketH, upEnd, rl.NewColor(80, 200, 255, 255))
}

// drawOrbitPlayers — точки игроков рядом с Землёй.
func (a *App) drawOrbitPlayers() {
	for _, p := range a.nc.InterpolatedSnapshot() {
		ph := rl.NewVector3(
			p.X+a.world.earthPos.X,
			p.Y+a.world.earthPos.Y,
			p.Z+a.world.earthPos.Z,
		)
		rl.DrawSphere(ph, 3, rl.NewColor(255, 220, 80, 255))
		rl.DrawSphereWires(ph, 3, 6, 6, rl.NewColor(120, 80, 0, 255))
	}
}
