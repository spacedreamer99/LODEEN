package input

import (
	"math"

	rl "github.com/gen2brain/raylib-go/raylib"

	"github.com/spacedreamer99/lodeen/internal/shared/protocol"
)

type Mode int

const (
	ModeCreative Mode = iota
	ModeSurvival
)

type FlightController struct {
	Pos rl.Vector3
	Vel rl.Vector3

	// Creative
	Quat rl.Quaternion

	// Survival
	TangentForward rl.Vector3 // куда идёт тело. Всегда в касательной плоскости.
	Pitch          float32    // наклон камеры, ±89°. На движение не влияет.

	Speed       float32
	Sensitivity float32
	RollSpeed   float32
	Riding      bool
	InBoat      bool

	Mode Mode

	// Bodies — для гравитации и «верха». Устанавливаются app.go каждый кадр.
	// Pos живёт в helio (мировой фрейм). EarthPos — текущая позиция Земли.
	EarthPos   rl.Vector3
	SunPos     rl.Vector3
	EarthVel   rl.Vector3
	HelioInit  bool
	smoothMinR float32

	// Основное состояние игрока в системе Земли.
	// f.Pos/f.Vel = earthPos/earthVel + RelPos/RelVel (для рендера).
	RelPos  rl.Vector3
	RelVel  rl.Vector3
	RelInit bool

	AttachedBody string

	// Debug
	lastOnGround bool
	lastJumpAt   int64
}

func New(pos rl.Vector3) *FlightController {
	fc := &FlightController{
		Pos:         pos,
		Quat:        rl.NewQuaternion(0, 0, 0, 1),
		Speed:       40,
		Sensitivity: 0.0025,
		RollSpeed:   1.0,
		Mode:        ModeSurvival,
	}
	// Инициализируем tangent-forward произвольным касательным вектором.
	up := rl.Vector3Normalize(pos)
	worldZ := rl.NewVector3(0, 0, 1)
	fw := rl.Vector3Subtract(worldZ, rl.Vector3Scale(up, rl.Vector3DotProduct(worldZ, up)))
	if rl.Vector3Length(fw) < 0.001 {
		worldY := rl.NewVector3(0, 1, 0)
		fw = rl.Vector3Subtract(worldY, rl.Vector3Scale(up, rl.Vector3DotProduct(worldY, up)))
	}
	fc.TangentForward = rl.Vector3Normalize(fw)
	return fc
}

// UpdateLookOnly — только вращение камеры, без физики и движения.
// Используется для режима пилотирования ракеты.
func (f *FlightController) UpdateLookOnly(dt float32, mouseDelta rl.Vector2) {
	if f.Mode == ModeSurvival {
		f.updateSurvivalLook(mouseDelta)
	} else {
		f.updateCreativeLook(mouseDelta, dt)
	}
}

func (f *FlightController) Update(dt float32, mouseDelta rl.Vector2) {
	// R (только в креативе) — цикл привязки к телу.
	if f.Mode == ModeCreative && rl.IsKeyPressed(rl.KeyR) {
		switch f.AttachedBody {
		case "":
			f.AttachedBody = "earth"
		case "earth":
			f.AttachedBody = "sun"
		default:
			f.AttachedBody = ""
		}
	}

	if f.Mode == ModeSurvival {
		f.updateSurvivalLook(mouseDelta)
		f.updateSurvival(dt)
	} else {
		f.updateCreativeLook(mouseDelta, dt)
		f.updateCreative(dt)
	}

	// Колёсико: в Creative — скорость, в Survival — выбор предмета (в app.go).
	if f.Mode == ModeCreative {
		wheel := rl.GetMouseWheelMove()
		if wheel != 0 {
			f.Speed *= 1.0 + wheel*0.3
			if f.Speed < 10 {
				f.Speed = 10
			}
			if f.Speed > 5000 {
				f.Speed = 5000
			}
		}
	}
}

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

func (f *FlightController) updateSurvival(dt float32) {
	const (
		gravity   = 30.0
		walkSpeed = 14.0
		jumpSpeed = 16.0
		groundEps = 0.5
	)
	oldPos := f.Pos

	// Инициализация RelPos/RelVel — один раз.
	if !f.RelInit {
		f.RelPos = rl.Vector3Subtract(f.Pos, f.EarthPos)
		f.RelVel = rl.Vector3Subtract(f.Vel, f.EarthVel)
		f.RelInit = true
	}

	// Работаем ТОЛЬКО с rel-состоянием. f.Pos/f.Vel — производные.
	rel := f.RelPos
	relVel := f.RelVel

	dist := rl.Vector3Length(rel)
	if dist < 0.01 {
		dist = 0.01
	}
	dir := rl.Vector3Scale(rel, 1/dist)

	th := protocol.TerrainHeight(dir.X, dir.Y, dir.Z)
	overWater := th < protocol.SeaLevel
	swimming := overWater && !f.InBoat

	ws := float32(walkSpeed)
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

	up := dir

	// Гравитация.
	relVel = rl.Vector3Subtract(relVel, rl.Vector3Scale(up, gravity*dt))

	onGround := dist <= minR+groundEps

	// WASD.
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
		wish = rl.Vector3Scale(wish, ws/l)
	}

	velTan := rl.Vector3Subtract(relVel, rl.Vector3Scale(up, rl.Vector3DotProduct(relVel, up)))
	if onGround {
		velTan = rl.Vector3Add(velTan, rl.Vector3Scale(rl.Vector3Subtract(wish, velTan), 0.25))
		velTan = rl.Vector3Scale(velTan, 0.75)
	} else {
		velTan = rl.Vector3Add(velTan, rl.Vector3Scale(rl.Vector3Subtract(wish, velTan), 0.05))
	}
	velRad := rl.Vector3Scale(up, rl.Vector3DotProduct(relVel, up))
	relVel = rl.Vector3Add(velRad, velTan)

	if onGround && rl.IsKeyDown(rl.KeySpace) {
		relVel = rl.Vector3Add(relVel, rl.Vector3Scale(up, jumpSpeed))
		onGround = false
	}

	// Интеграция rel-позиции.
	rel = rl.Vector3Add(rel, rl.Vector3Scale(relVel, dt))

	// Коллизия в rel-координатах.
	newDist := rl.Vector3Length(rel)
	if newDist < minR {
		scale := minR / newDist
		rel = rl.Vector3Scale(rel, scale)
		nr := rl.Vector3Normalize(rel)
		vr := rl.Vector3DotProduct(relVel, nr)
		if vr < 0 {
			relVel = rl.Vector3Subtract(relVel, rl.Vector3Scale(nr, vr))
		}
	}

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

// --- Creative ---

func (f *FlightController) updateCreativeLook(mouseDelta rl.Vector2, dt float32) {
	if mouseDelta.X != 0 {
		q := rl.QuaternionFromAxisAngle(rl.NewVector3(0, 1, 0), -mouseDelta.X*f.Sensitivity)
		f.Quat = rl.QuaternionNormalize(rl.QuaternionMultiply(f.Quat, q))
	}
	if mouseDelta.Y != 0 {
		q := rl.QuaternionFromAxisAngle(rl.NewVector3(1, 0, 0), -mouseDelta.Y*f.Sensitivity)
		f.Quat = rl.QuaternionNormalize(rl.QuaternionMultiply(f.Quat, q))
	}
	if rl.IsKeyDown(rl.KeyQ) {
		q := rl.QuaternionFromAxisAngle(rl.NewVector3(0, 0, -1), -f.RollSpeed*dt)
		f.Quat = rl.QuaternionNormalize(rl.QuaternionMultiply(f.Quat, q))
	}
	if rl.IsKeyDown(rl.KeyE) {
		q := rl.QuaternionFromAxisAngle(rl.NewVector3(0, 0, -1), f.RollSpeed*dt)
		f.Quat = rl.QuaternionNormalize(rl.QuaternionMultiply(f.Quat, q))
	}
}

func (f *FlightController) updateCreative(dt float32) {
	// Если привязан к Земле — компенсируем её движение.
	if f.AttachedBody == "earth" {
		f.Pos = rl.Vector3Add(f.Pos, rl.Vector3Scale(f.EarthVel, dt))
	}

	fw := f.creativeForward()
	rt := f.creativeRight()
	up := f.creativeUp()
	move := rl.Vector3Zero()
	if rl.IsKeyDown(rl.KeyW) {
		move = rl.Vector3Add(move, fw)
	}
	if rl.IsKeyDown(rl.KeyS) {
		move = rl.Vector3Subtract(move, fw)
	}
	if rl.IsKeyDown(rl.KeyD) {
		move = rl.Vector3Add(move, rt)
	}
	if rl.IsKeyDown(rl.KeyA) {
		move = rl.Vector3Subtract(move, rt)
	}
	if rl.IsKeyDown(rl.KeySpace) {
		move = rl.Vector3Add(move, up)
	}
	if rl.IsKeyDown(rl.KeyLeftShift) {
		move = rl.Vector3Subtract(move, up)
	}
	if l := rl.Vector3Length(move); l > 0 {
		move = rl.Vector3Scale(move, 1.0/l)
		f.Pos = rl.Vector3Add(f.Pos, rl.Vector3Scale(move, f.Speed*dt))
	}
	f.Vel = rl.Vector3Zero()
}

func (f *FlightController) creativeForward() rl.Vector3 {
	return rl.Vector3RotateByQuaternion(rl.NewVector3(0, 0, -1), f.Quat)
}
func (f *FlightController) creativeRight() rl.Vector3 {
	return rl.Vector3RotateByQuaternion(rl.NewVector3(1, 0, 0), f.Quat)
}
func (f *FlightController) creativeUp() rl.Vector3 {
	return rl.Vector3RotateByQuaternion(rl.NewVector3(0, 1, 0), f.Quat)
}

// --- Публичные геттеры (используются в app.go) ---

func (f *FlightController) Forward() rl.Vector3 {
	if f.Mode == ModeCreative {
		return f.creativeForward()
	}
	return f.cameraForward()
}

func (f *FlightController) Right() rl.Vector3 {
	if f.Mode == ModeCreative {
		return f.creativeRight()
	}
	up := f.survivalUp()
	return rl.Vector3Normalize(rl.Vector3CrossProduct(f.cameraForward(), up))
}

func (f *FlightController) Up() rl.Vector3 {
	if f.Mode == ModeCreative {
		return f.creativeUp()
	}
	return f.survivalUp()
}

func (f *FlightController) CameraUp() rl.Vector3 { return f.Up() }
