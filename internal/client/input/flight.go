package input

import (
	rl "github.com/gen2brain/raylib-go/raylib"
)

// FlightController implements true 6DOF:
//
//	Mouse X — yaw   around WORLD up (pre-multiply, world axis)
//	Mouse Y — pitch around LOCAL right (post-multiply, local axis)
//	Q/E     — roll  around LOCAL forward (post-multiply, local axis)
//
// Pre-multiply = rotation in world frame (axis fixed in world).
// Post-multiply = rotation in local frame (axis attached to the cube).
type FlightController struct {
	Pos  rl.Vector3
	Quat rl.Quaternion

	Speed       float32
	Sensitivity float32
	RollSpeed   float32
}

func New(pos rl.Vector3) *FlightController {
	return &FlightController{
		Pos:         pos,
		Quat:        rl.NewQuaternion(0, 0, 0, 1),
		Speed:       40,
		Sensitivity: 0.0025,
		RollSpeed:   1.0,
	}
}

func (f *FlightController) Update(dt float32, mouseDelta rl.Vector2) {
	// Yaw around WORLD up — pre-multiply.
	if mouseDelta.X != 0 {
		localUp := rl.NewVector3(0, 1, 0)
		q := rl.QuaternionFromAxisAngle(localUp, -mouseDelta.X*f.Sensitivity)
		f.Quat = rl.QuaternionNormalize(rl.QuaternionMultiply(f.Quat, q))
	}

	// Pitch around LOCAL right (1,0,0) — post-multiply.
	if mouseDelta.Y != 0 {
		localRight := rl.NewVector3(1, 0, 0)
		q := rl.QuaternionFromAxisAngle(localRight, -mouseDelta.Y*f.Sensitivity)
		f.Quat = rl.QuaternionNormalize(rl.QuaternionMultiply(f.Quat, q))
	}

	// Roll around LOCAL forward (0,0,-1) — post-multiply.
	roll := float32(0)
	if rl.IsKeyDown(rl.KeyQ) {
		roll -= f.RollSpeed * dt
	}
	if rl.IsKeyDown(rl.KeyE) {
		roll += f.RollSpeed * dt
	}
	if roll != 0 {
		localForward := rl.NewVector3(0, 0, -1)
		q := rl.QuaternionFromAxisAngle(localForward, roll)
		f.Quat = rl.QuaternionNormalize(rl.QuaternionMultiply(f.Quat, q))
	}

	fw := f.Forward()
	rt := f.Right()
	up := f.Up()

	// Колёсико мыши — скорость. Одно деление ≈ ±10%.
	wheel := rl.GetMouseWheelMove()
	if wheel != 0 {
		f.Speed *= 1.0 + wheel*0.1
		if f.Speed < 5 {
			f.Speed = 5
		}
		if f.Speed > 500 {
			f.Speed = 500
		}
	}
	speed := f.Speed

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
	if rl.IsKeyDown(rl.KeyLeftShift) || rl.IsKeyDown(rl.KeyRightShift) {
		move = rl.Vector3Subtract(move, up)
	}

	if l := rl.Vector3Length(move); l > 0 {
		move = rl.Vector3Scale(move, 1.0/l)
		f.Pos = rl.Vector3Add(f.Pos, rl.Vector3Scale(move, speed*dt))
	}
}

func (f *FlightController) Forward() rl.Vector3 {
	return rl.Vector3RotateByQuaternion(rl.NewVector3(0, 0, -1), f.Quat)
}

func (f *FlightController) Right() rl.Vector3 {
	return rl.Vector3RotateByQuaternion(rl.NewVector3(1, 0, 0), f.Quat)
}

func (f *FlightController) Up() rl.Vector3 {
	return rl.Vector3RotateByQuaternion(rl.NewVector3(0, 1, 0), f.Quat)
}

func (f *FlightController) CameraUp() rl.Vector3 { return f.Up() }
