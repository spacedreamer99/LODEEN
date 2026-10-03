package app

import (
	"math"

	rl "github.com/gen2brain/raylib-go/raylib"

	"github.com/spacedreamer99/lodeen/internal/client/fonts"
	"github.com/spacedreamer99/lodeen/internal/shared/protocol"
)

// navballFrame — локальный ортонормированный базис навбола.
type navballFrame struct {
	fX, fY, fZ    float32 // forward = up ракеты
	rX, rY, rZ    float32 // right = cross(forward, worldUp)
	tuX, tuY, tuZ float32 // trueUp = cross(right, forward)
	wX, wY, wZ    float32 // radial out (нормаль планеты)
}

// drawNavBall — оркестратор.
func (a *App) drawNavBall() {
	r := a.rocket.hudRocket
	if r.ID == "" {
		return
	}
	a.logNavBallDebug(r)

	layout := computeNavBallLayout()
	frame, ok := buildNavBallFrame(r)
	if !ok {
		return
	}

	drawNavBallBackground(layout)

	project := makeNavBallProjector(layout, frame)

	// Центр — forward (нос).
	rl.DrawCircle(layout.cx, layout.cy, 3, rl.NewColor(80, 220, 255, 255))

	drawNavBallMarkers(r, frame, project)

	drawNavBallAutoPilotLabel(a.rocket.autoPilot, layout)
}

// --- layout ---

type navBallLayout struct {
	cx, cy int32
	radius int32
}

func computeNavBallLayout() navBallLayout {
	sw := int32(rl.GetScreenWidth())
	sh := int32(rl.GetScreenHeight())
	const radius = int32(90)
	return navBallLayout{
		cx:     sw - radius - 30,
		cy:     sh - radius - 30,
		radius: radius,
	}
}

func drawNavBallBackground(l navBallLayout) {
	rl.DrawCircle(l.cx, l.cy, float32(l.radius), rl.NewColor(20, 30, 50, 220))
	rl.DrawCircleLines(l.cx, l.cy, float32(l.radius), rl.NewColor(120, 180, 240, 255))
	rl.DrawCircleLines(l.cx, l.cy, float32(l.radius)*0.66, rl.NewColor(80, 100, 140, 180))
	rl.DrawCircleLines(l.cx, l.cy, float32(l.radius)*0.33, rl.NewColor(80, 100, 140, 180))
	rl.DrawLine(
		l.cx-int32(float32(l.radius)*0.9), l.cy,
		l.cx+int32(float32(l.radius)*0.9), l.cy,
		rl.NewColor(120, 140, 180, 220),
	)
}

// --- frame ---

// buildNavBallFrame строит локальный базис из forward ракеты и радиальной нормали.
func buildNavBallFrame(r protocol.Rocket) (navballFrame, bool) {
	var f navballFrame

	f.fX, f.fY, f.fZ = r.DX, r.DY, r.DZ

	pLen := float32(math.Sqrt(float64(r.X*r.X + r.Y*r.Y + r.Z*r.Z)))
	if pLen < 0.01 {
		return f, false
	}
	f.wX, f.wY, f.wZ = r.X/pLen, r.Y/pLen, r.Z/pLen

	// right = cross(forward, worldUp)
	f.rX = f.fY*f.wZ - f.fZ*f.wY
	f.rY = f.fZ*f.wX - f.fX*f.wZ
	f.rZ = f.fX*f.wY - f.fY*f.wX
	rLen := float32(math.Sqrt(float64(f.rX*f.rX + f.rY*f.rY + f.rZ*f.rZ)))
	if rLen < 0.01 {
		return f, false
	}
	f.rX /= rLen
	f.rY /= rLen
	f.rZ /= rLen

	// trueUp = cross(right, forward)
	f.tuX = f.rY*f.fZ - f.rZ*f.fY
	f.tuY = f.rZ*f.fX - f.rX*f.fZ
	f.tuZ = f.rX*f.fY - f.rY*f.fX

	return f, true
}

// --- projector ---

// navBallProjector возвращает screen-координаты вектора на навболе.
// ok=false если вектор "за" навболом (направление почти против forward).
type navBallProjector func(vx, vy, vz float32) (px, py int32, ok bool)

func makeNavBallProjector(l navBallLayout, f navballFrame) navBallProjector {
	return func(vx, vy, vz float32) (int32, int32, bool) {
		fwd := vx*f.fX + vy*f.fY + vz*f.fZ
		rgt := vx*f.rX + vy*f.rY + vz*f.rZ
		upv := vx*f.tuX + vy*f.tuY + vz*f.tuZ
		if fwd < -0.2 {
			return 0, 0, false
		}
		px := l.cx + int32(rgt*float32(l.radius))
		py := l.cy - int32(upv*float32(l.radius))
		return px, py, true
	}
}

// --- markers ---

func drawNavBallMarkers(r protocol.Rocket, f navballFrame, project navBallProjector) {
	// Prograde / retrograde (по скорости).
	vLen := float32(math.Sqrt(float64(r.VX*r.VX + r.VY*r.VY + r.VZ*r.VZ)))
	if vLen > 0.5 {
		drawNavBallTarget(project, r.VX, r.VY, r.VZ,
			rl.NewColor(80, 220, 100, 255), "PRO")
		drawNavBallTarget(project, -r.VX, -r.VY, -r.VZ,
			rl.NewColor(230, 140, 40, 255), "RET")
	}

	// Radial out / in.
	drawNavBallTarget(project, f.wX, f.wY, f.wZ,
		rl.NewColor(230, 200, 60, 255), "RAD+")
	drawNavBallTarget(project, -f.wX, -f.wY, -f.wZ,
		rl.NewColor(180, 100, 220, 255), "RAD-")

	// Normal / antinormal = cross(radial, velocity).
	nx := r.Y*r.VZ - r.Z*r.VY
	ny := r.Z*r.VX - r.X*r.VZ
	nz := r.X*r.VY - r.Y*r.VX
	nl := float32(math.Sqrt(float64(nx*nx + ny*ny + nz*nz)))
	if nl > 0.1 {
		drawNavBallTarget(project, nx/nl, ny/nl, nz/nl,
			rl.NewColor(100, 180, 255, 255), "NRM")
		drawNavBallTarget(project, -nx/nl, -ny/nl, -nz/nl,
			rl.NewColor(100, 180, 255, 255), "ANM")
	}
}

// drawNavBallTarget рисует кружок цели по направлению вектора.
func drawNavBallTarget(project navBallProjector, vx, vy, vz float32, col rl.Color, label string) {
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

// --- labels ---

func drawNavBallAutoPilotLabel(mode string, l navBallLayout) {
	label := "MANUAL"
	col := rl.LightGray
	switch mode {
	case "prograde":
		label, col = "PROGRADE", rl.NewColor(80, 220, 100, 255)
	case "retrograde":
		label, col = "RETROGRADE", rl.NewColor(230, 140, 40, 255)
	case "radial_out":
		label, col = "RADIAL+", rl.NewColor(230, 200, 60, 255)
	case "radial_in":
		label, col = "RADIAL-", rl.NewColor(180, 100, 220, 255)
	case "normal":
		label, col = "NORMAL", rl.NewColor(100, 180, 255, 255)
	case "antinormal":
		label, col = "ANTINORMAL", rl.NewColor(100, 180, 255, 255)
	}
	w := fonts.Measure(label, 16)
	fonts.Draw(label, l.cx-w/2, l.cy+l.radius+8, 16, col)
}

// --- debug ---

func (a *App) logNavBallDebug(r protocol.Rocket) {
	if a.diag.debugFrame%60 != 0 {
		return
	}
	a.log.Info("navball draw",
		"id", r.ID,
		"dx", r.DX, "dy", r.DY, "dz", r.DZ,
		"vx", r.VX, "vy", r.VY, "vz", r.VZ)
}

// ============================================================
// drawRocketHUD
// ============================================================

// drawRocketHUD — оркестратор панели телеметрии ракеты.
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

	escaping := r.Apoapsis == 0 && r.Altitude > 0 && r.Speed > 5
	leftEarth := r.PrimaryBody == "sun"

	drawRocketHUDPanel(px, py, panelW, panelH, r, escaping, leftEarth)
	drawRocketHUDTelemetry(px, py, r, escaping)
	drawRocketHUDHints(px, py, panelH, escaping)
}

func drawRocketHUDPanel(px, py, w, h int32, r protocol.Rocket, escaping, leftEarth bool) {
	bg := rl.NewColor(10, 15, 30, 220)
	border := rl.NewColor(120, 180, 240, 255)
	switch {
	case leftEarth:
		bg = rl.NewColor(30, 15, 45, 230)
		border = rl.NewColor(200, 120, 240, 255)
	case r.InOrbit:
		bg = rl.NewColor(15, 30, 15, 230)
		border = rl.NewColor(80, 220, 100, 255)
	case escaping:
		bg = rl.NewColor(40, 15, 15, 230)
		border = rl.NewColor(220, 60, 60, 255)
	}
	panel := rl.NewRectangle(float32(px), float32(py), float32(w), float32(h))
	rl.DrawRectangleRec(panel, bg)
	rl.DrawRectangleLinesEx(panel, 2, border)

	title := "ROCKET"
	switch {
	case leftEarth:
		title = "HELIOCENTRIC ORBIT"
	case r.InOrbit:
		title = "ORBIT ACHIEVED"
	case escaping:
		title = "ESCAPE TRAJECTORY"
	}
	fonts.Draw(title, px+12, py+8, 20, border)
}

func drawRocketHUDTelemetry(px, py int32, r protocol.Rocket, escaping bool) {
	const pad = int32(12)
	lineY := py + 40
	const lineH = int32(20)

	// FUEL + ORBIT.
	fonts.Draw("FUEL: "+itoa(r.Fuel)+" / "+itoa(r.MaxFuel), px+pad, lineY, 16, rl.RayWhite)
	body := "EARTH"
	bodyCol := rl.NewColor(100, 180, 240, 255)
	if r.PrimaryBody == "sun" {
		body = "SUN"
		bodyCol = rl.NewColor(240, 180, 60, 255)
	}
	fonts.Draw("ORBIT: "+body, px+pad+150, lineY, 16, bodyCol)
	lineY += lineH

	// ALT.
	fonts.Draw("ALT: "+itoa(int(r.Altitude))+" m", px+pad, lineY, 16, rl.RayWhite)
	lineY += lineH

	// VEL + orbit target.
	fonts.Draw(rocketVelocityLine(r), px+pad, lineY, 16, rocketVelocityColor(r))
	lineY += lineH

	// APO + PER.
	apoCol := rl.LightGray
	if r.Apoapsis > 0 {
		apoCol = rl.RayWhite
	}
	fonts.Draw("APO: "+itoa(int(r.Apoapsis))+" m", px+pad, lineY, 16, apoCol)
	fonts.Draw("PER: "+itoa(int(r.Periapsis))+" m", px+pad+140, lineY, 16, rocketPeriapsisColor(r))
}

func rocketVelocityLine(r protocol.Rocket) string {
	s := "VEL: " + itoa(int(r.Speed)) + " m/s"
	if r.TargetVelocity > 0 {
		s += "  (orbit: " + itoa(int(r.TargetVelocity)) + ")"
	}
	return s
}

func rocketVelocityColor(r protocol.Rocket) rl.Color {
	if r.TargetVelocity <= 0 {
		return rl.RayWhite
	}
	ratio := r.Speed / r.TargetVelocity
	switch {
	case ratio >= 0.85 && ratio <= 1.15:
		return rl.NewColor(80, 220, 100, 255) // точно
	case ratio >= 0.5 && ratio <= 1.5:
		return rl.NewColor(230, 200, 60, 255) // близко
	default:
		return rl.NewColor(220, 60, 60, 255) // не то
	}
}

func rocketPeriapsisColor(r protocol.Rocket) rl.Color {
	switch {
	case r.Periapsis > 60.0:
		return rl.NewColor(80, 220, 100, 255)
	case r.Periapsis > 0:
		return rl.NewColor(230, 200, 60, 255)
	default:
		return rl.NewColor(220, 60, 60, 255)
	}
}

func drawRocketHUDHints(px, py, panelH int32, escaping bool) {
	const pad = int32(12)
	hint := "WASD nose | Space thrust | Ctrl retro | M map | E exit"
	if escaping {
		hint = "TOO FAST — turn sideways, release Space"
	}
	hint2 := "Auto: G=prograde  H=retro  J=radial+  K=radial-  N=normal  B=anti  X=off"
	fonts.Draw(hint2, px+pad, py+panelH-40, 11, rl.Gray)
	fonts.Draw(hint, px+pad, py+panelH-24, 12, rl.Gray)
}
