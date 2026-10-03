package input

import (
	rl "github.com/gen2brain/raylib-go/raylib"
)

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
	// Компенсация движения Земли — в SyncToEarthFrame(), вызывается извне
	// каждый кадр, даже когда UI открыт.

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
