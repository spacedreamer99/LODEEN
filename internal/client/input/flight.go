package input

import (
	"math"

	rl "github.com/gen2brain/raylib-go/raylib"
)

type FlightController struct {
	Pos   rl.Vector3
	Yaw   float32
	Pitch float32
	Roll  float32

	Speed       float32
	Sensitivity float32
}

func New(pos rl.Vector3) *FlightController {
	return &FlightController{
		Pos:         pos,
		Speed:       40,
		Sensitivity: 0.0025,
	}
}

func (f *FlightController) Update(dt float32, mouseDelta rl.Vector2) {
	f.Yaw -= mouseDelta.X * f.Sensitivity
	f.Pitch -= mouseDelta.Y * f.Sensitivity

	maxPitch := float32(math.Pi/2) - 0.01
	if f.Pitch > maxPitch {
		f.Pitch = maxPitch
	}
	if f.Pitch < -maxPitch {
		f.Pitch = -maxPitch
	}

	if rl.IsKeyDown(rl.KeyQ) {
		f.Roll += 2.0 * dt
	}
	if rl.IsKeyDown(rl.KeyE) {
		f.Roll -= 2.0 * dt
	}

	fw := f.Forward()
	rt := f.Right()
	up := rl.Vector3CrossProduct(rt, fw)

	speed := f.Speed
	if rl.IsKeyDown(rl.KeyLeftShift) {
		speed *= 4
	}
	if rl.IsKeyDown(rl.KeyLeftControl) {
		speed *= 0.25
	}

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
	if rl.IsKeyDown(rl.KeyLeftAlt) {
		move = rl.Vector3Subtract(move, up)
	}

	if l := rl.Vector3Length(move); l > 0 {
		move = rl.Vector3Scale(move, 1.0/l)
		f.Pos = rl.Vector3Add(f.Pos, rl.Vector3Scale(move, speed*dt))
	}
}

func (f *FlightController) Forward() rl.Vector3 {
	cp := float32(math.Cos(float64(f.Pitch)))
	sp := float32(math.Sin(float64(f.Pitch)))
	cy := float32(math.Cos(float64(f.Yaw)))
	sy := float32(math.Sin(float64(f.Yaw)))
	return rl.NewVector3(-sy*cp, sp, -cy*cp)
}

func (f *FlightController) Right() rl.Vector3 {
	cy := float32(math.Cos(float64(f.Yaw)))
	sy := float32(math.Sin(float64(f.Yaw)))
	return rl.NewVector3(cy, 0, -sy)
}

func (f *FlightController) CameraUp() rl.Vector3 {
	fw := f.Forward()
	rt := f.Right()
	up := rl.Vector3CrossProduct(rt, fw)
	if f.Roll != 0 {
		up = rl.Vector3RotateByAxisAngle(up, fw, f.Roll)
	}
	return up
}
