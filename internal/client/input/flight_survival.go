package input

import (
	"math"

	rl "github.com/gen2brain/raylib-go/raylib"

	"github.com/spacedreamer99/lodeen/internal/shared/protocol"
)

// --- Survival ---

func (f *FlightController) updateSurvivalLook(mouseDelta rl.Vector2) {
	up := f.survivalUp()

	// Yaw — вокруг up. Крутит только тело (TangentForward).
	if mouseDelta.X != 0 {
		q := rl.QuaternionFromAxisAngle(up, -mouseDelta.X*f.Sensitivity)
		f.TangentForward = rl.Vector3RotateByQuaternion(f.TangentForward, q)
		f.TangentForward = f.projectToTangent(f.TangentForward)
	}

	// Pitch — только наклон камеры. На тело не влияет.
	f.Pitch -= mouseDelta.Y * f.Sensitivity
	const maxPitch = 89 * math.Pi / 180
	if f.Pitch > maxPitch {
		f.Pitch = maxPitch
	}
	if f.Pitch < -maxPitch {
		f.Pitch = -maxPitch
	}
}

func (f *FlightController) survivalUp() rl.Vector3 {
	rel := rl.Vector3Subtract(f.Pos, f.EarthPos)
	d := rl.Vector3Length(rel)
	if d < 0.01 {
		d = 0.01
	}
	return rl.Vector3Scale(rel, 1/d)
}

func (f *FlightController) projectToTangent(v rl.Vector3) rl.Vector3 {
	up := f.survivalUp()
	out := rl.Vector3Subtract(v, rl.Vector3Scale(up, rl.Vector3DotProduct(v, up)))
	if l := rl.Vector3Length(out); l > 0.001 {
		return rl.Vector3Scale(out, 1/l)
	}
	// Резерв
	wZ := rl.NewVector3(0, 0, 1)
	out = rl.Vector3Subtract(wZ, rl.Vector3Scale(up, rl.Vector3DotProduct(wZ, up)))
	if l := rl.Vector3Length(out); l > 0.001 {
		return rl.Vector3Scale(out, 1/l)
	}
	wY := rl.NewVector3(0, 1, 0)
	out = rl.Vector3Subtract(wY, rl.Vector3Scale(up, rl.Vector3DotProduct(wY, up)))
	return rl.Vector3Normalize(out)
}

func (f *FlightController) parallelTransport(oldPos, newPos rl.Vector3) {
	oldRel := rl.Vector3Subtract(oldPos, f.EarthPos)
	newRel := rl.Vector3Subtract(newPos, f.EarthPos)
	oldUp := rl.Vector3Normalize(oldRel)
	newUp := rl.Vector3Normalize(newRel)
	if rl.Vector3Distance(oldUp, newUp) < 0.0001 {
		return
	}
	q := rl.QuaternionFromVector3ToVector3(oldUp, newUp)
	f.TangentForward = rl.Vector3RotateByQuaternion(f.TangentForward, q)
}

// initRelIfNeeded инициализирует rel-состояние из helio один раз.
func (f *FlightController) initRelIfNeeded() {
	if f.RelInit {
		return
	}
	f.RelPos = rl.Vector3Subtract(f.Pos, f.EarthPos)
	f.RelVel = rl.Vector3Subtract(f.Vel, f.EarthVel)
	f.RelInit = true
}

// survivalGeom — геометрия для одного тика Survival.
type survivalGeom struct {
	dir       rl.Vector3
	dist      float32
	minR      float32
	onGround  bool
	walkSpeed float32
}

// computeSurvivalGeom считает нормаль, дистанцию до центра, мин. радиус
// и эффективную скорость ходьбы (с учётом езды / лодки / плавания).
func (f *FlightController) computeSurvivalGeom(rel rl.Vector3, groundEps, baseWalkSpeed float32) survivalGeom {
	dist := rl.Vector3Length(rel)
	if dist < 0.01 {
		dist = 0.01
	}
	dir := rl.Vector3Scale(rel, 1/dist)

	th := protocol.TerrainHeight(dir.X, dir.Y, dir.Z)
	overWater := th < protocol.SeaLevel
	swimming := overWater && !f.InBoat

	ws := baseWalkSpeed
	switch {
	case f.Riding:
		ws *= 2.5
	case f.InBoat && overWater:
		ws *= 2.0
	case f.InBoat && !overWater:
		ws *= 0.15
	case swimming:
		ws *= 0.35
	}

	surfaceR := protocol.SurfaceRadius(protocol.Vector3{X: dir.X, Y: dir.Y, Z: dir.Z})
	minR := surfaceR + protocol.PlayerHeight

	return survivalGeom{
		dir:       dir,
		dist:      dist,
		minR:      minR,
		onGround:  dist <= minR+groundEps,
		walkSpeed: ws,
	}
}

// applyWalkInput применяет WASD + трение в касательной плоскости.
func (f *FlightController) applyWalkInput(relVel, up rl.Vector3, g survivalGeom) rl.Vector3 {
	fwTan := f.TangentForward
	rt := rl.Vector3Normalize(rl.Vector3CrossProduct(fwTan, up))

	wish := rl.Vector3Zero()
	if rl.IsKeyDown(rl.KeyW) {
		wish = rl.Vector3Add(wish, fwTan)
	}
	if rl.IsKeyDown(rl.KeyS) {
		wish = rl.Vector3Subtract(wish, fwTan)
	}
	if rl.IsKeyDown(rl.KeyD) {
		wish = rl.Vector3Add(wish, rt)
	}
	if rl.IsKeyDown(rl.KeyA) {
		wish = rl.Vector3Subtract(wish, rt)
	}
	if l := rl.Vector3Length(wish); l > 0 {
		wish = rl.Vector3Scale(wish, g.walkSpeed/l)
	}

	velTan := rl.Vector3Subtract(relVel, rl.Vector3Scale(up, rl.Vector3DotProduct(relVel, up)))
	if g.onGround {
		velTan = rl.Vector3Add(velTan, rl.Vector3Scale(rl.Vector3Subtract(wish, velTan), 0.25))
		velTan = rl.Vector3Scale(velTan, 0.75)
	} else {
		velTan = rl.Vector3Add(velTan, rl.Vector3Scale(rl.Vector3Subtract(wish, velTan), 0.05))
	}
	velRad := rl.Vector3Scale(up, rl.Vector3DotProduct(relVel, up))
	return rl.Vector3Add(velRad, velTan)
}

// collideWithSurface прижимает rel к minR и гасит радиальную скорость внутрь.
func collideWithSurface(rel, relVel rl.Vector3, minR float32) (rl.Vector3, rl.Vector3) {
	newDist := rl.Vector3Length(rel)
	if newDist >= minR {
		return rel, relVel
	}
	scale := minR / newDist
	rel = rl.Vector3Scale(rel, scale)
	nr := rl.Vector3Normalize(rel)
	vr := rl.Vector3DotProduct(relVel, nr)
	if vr < 0 {
		relVel = rl.Vector3Subtract(relVel, rl.Vector3Scale(nr, vr))
	}
	return rel, relVel
}

// updateSurvival — оркестратор тика Survival: геометрия → гравитация →
// ввод → прыжок → интеграция → коллизия → commit.
func (f *FlightController) updateSurvival(dt float32) {
	const (
		gravity   = 30.0
		walkSpeed = 14.0
		jumpSpeed = 16.0
		groundEps = 0.5
	)
	oldPos := f.Pos

	f.initRelIfNeeded()

	rel := f.RelPos
	relVel := f.RelVel

	g := f.computeSurvivalGeom(rel, groundEps, walkSpeed)
	up := g.dir

	// Гравитация.
	relVel = rl.Vector3Subtract(relVel, rl.Vector3Scale(up, gravity*dt))

	// WASD + трение.
	relVel = f.applyWalkInput(relVel, up, g)

	// Прыжок.
	if g.onGround && rl.IsKeyDown(rl.KeySpace) {
		relVel = rl.Vector3Add(relVel, rl.Vector3Scale(up, jumpSpeed))
	}

	// Интеграция rel-позиции + коллизия.
	rel = rl.Vector3Add(rel, rl.Vector3Scale(relVel, dt))
	rel, relVel = collideWithSurface(rel, relVel, g.minR)

	// Сохраняем основное состояние.
	f.RelPos = rel
	f.RelVel = relVel

	// Производные для рендера — точные, БЕЗ накопления ошибки.
	f.Pos = rl.Vector3Add(f.EarthPos, rel)
	f.Vel = rl.Vector3Add(f.EarthVel, relVel)

	f.parallelTransport(oldPos, f.Pos)
	f.TangentForward = f.projectToTangent(f.TangentForward)
}

// cameraForward — куда смотрит камера в Survival:
// TangentForward повёрнут на Pitch вокруг right.
func (f *FlightController) cameraForward() rl.Vector3 {
	up := f.survivalUp()
	cp := float32(math.Cos(float64(f.Pitch)))
	sp := float32(math.Sin(float64(f.Pitch)))
	dir := rl.Vector3Add(rl.Vector3Scale(f.TangentForward, cp), rl.Vector3Scale(up, sp))
	return rl.Vector3Normalize(dir)
}
