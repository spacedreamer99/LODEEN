package input

import (
	"math"

	rl "github.com/gen2brain/raylib-go/raylib"
)

// Mode — режим управления: Creative (свободный полёт) или Survival (ходьба).
type Mode int

const (
	ModeCreative Mode = iota
	ModeSurvival
)

// FlightController — управление телом игрока в двух режимах.
// Survival: rel-состояние (RelPos/RelVel) относительно Земли.
// Creative: helio (Pos/Vel) в мировом фрейме.
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

	// Bodies — для гравитации и "верха". Устанавливаются app.go каждый кадр.
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

// SwitchMode переключает режим с конверсией состояния между фреймами.
// Creative живёт в helio (Pos/Vel), Survival — в geo (RelPos/RelVel).
func (f *FlightController) SwitchMode(newMode Mode) {
	if f.Mode == newMode {
		return
	}
	switch newMode {
	case ModeSurvival:
		// Creative -> Survival: из helio в geo
		f.RelPos = rl.Vector3Subtract(f.Pos, f.EarthPos)
		f.RelVel = rl.Vector3Subtract(f.Vel, f.EarthVel)
		f.RelInit = true

		// TangentForward из текущего направления камеры (Quat)
		fw := rl.Vector3RotateByQuaternion(rl.NewVector3(0, 0, -1), f.Quat)
		rel := f.RelPos
		d := rl.Vector3Length(rel)
		if d < 0.01 {
			d = 0.01
		}
		up := rl.Vector3Scale(rel, 1/d)
		tf := rl.Vector3Subtract(fw, rl.Vector3Scale(up, rl.Vector3DotProduct(fw, up)))
		if l := rl.Vector3Length(tf); l > 0.001 {
			f.TangentForward = rl.Vector3Scale(tf, 1/l)
		}
		// Pitch из dot(fw, up)
		sin := rl.Vector3DotProduct(fw, up)
		if sin > 1 {
			sin = 1
		} else if sin < -1 {
			sin = -1
		}
		f.Pitch = float32(math.Asin(float64(sin)))

	case ModeCreative:
		// Survival -> Creative: из geo в helio
		if f.RelInit {
			f.Pos = rl.Vector3Add(f.EarthPos, f.RelPos)
			f.Vel = rl.Vector3Add(f.EarthVel, f.RelVel)
		}
		// Quat из cameraForward() и up
		fw := f.cameraForward()
		rel := f.RelPos
		d := rl.Vector3Length(rel)
		if d < 0.01 {
			d = 0.01
		}
		up := rl.Vector3Scale(rel, 1/d)
		f.Quat = lookRotation(fw, up)

		// Автопривязка к Земле при входе в Creative.
		// Игрок сразу летит вместе с планетой — не нужно жать R.
		// R всё ещё работает и может переключить на sun / none.
		if f.AttachedBody == "" {
			f.AttachedBody = "earth"
		}
	}
	f.Mode = newMode
}

// lookRotation строит quaternion, который смотрит в forward с верхом up.
func lookRotation(forward, up rl.Vector3) rl.Quaternion {
	forward = rl.Vector3Normalize(forward)
	up = rl.Vector3Normalize(up)
	// Шаг 1: -Z → forward
	q1 := rl.QuaternionFromVector3ToVector3(rl.NewVector3(0, 0, -1), forward)
	// Куда смотрит up после первого вращения
	upAfter := rl.Vector3RotateByQuaternion(rl.NewVector3(0, 1, 0), q1)
	// Шаг 2: довернуть вокруг forward чтобы up совпал
	q2 := rl.QuaternionFromVector3ToVector3(upAfter, up)
	return rl.QuaternionNormalize(rl.QuaternionMultiply(q2, q1))
}
