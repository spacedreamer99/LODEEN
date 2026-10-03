package input

import (
	rl "github.com/gen2brain/raylib-go/raylib"

	"github.com/spacedreamer99/lodeen/internal/shared/protocol"
)

// SyncToEarthFrame обновляет Pos из текущей позиции Земли.
// Вызывается каждый кадр (даже когда открыт UI) — иначе
// игрок отстаёт от планеты, пока UI блокирует физику.
func (f *FlightController) SyncToEarthFrame(dt float32) {
	switch f.Mode {
	case ModeSurvival:
		if f.RelInit {
			f.Pos = rl.Vector3Add(f.EarthPos, f.RelPos)
			f.Vel = rl.Vector3Add(f.EarthVel, f.RelVel)
		}
	case ModeCreative:
		if f.AttachedBody == "earth" {
			f.Pos = rl.Vector3Add(f.Pos, rl.Vector3Scale(f.EarthVel, dt))
		}
	}
}

// TickPhysicsOnly применяет гравитацию и коллизию без ввода.
// Используется на паузе / при открытом UI, чтобы персонаж не зависал в воздухе.
func (f *FlightController) TickPhysicsOnly(dt float32) {
	if f.Mode != ModeSurvival || !f.RelInit {
		return
	}
	oldPos := f.Pos
	rel := f.RelPos
	relVel := f.RelVel

	dist := rl.Vector3Length(rel)
	if dist < 0.01 {
		dist = 0.01
	}
	up := rl.Vector3Scale(rel, 1/dist)

	const gravity = 30.0
	const groundEps = 0.5

	surfaceR := protocol.SurfaceRadius(protocol.Vector3{X: up.X, Y: up.Y, Z: up.Z})
	minR := surfaceR + protocol.PlayerHeight

	onGround := dist <= minR+groundEps

	if !onGround {
		relVel = rl.Vector3Subtract(relVel, rl.Vector3Scale(up, gravity*dt))
	}

	// Трение касательной (чтобы не улетел далеко по инерции).
	velTan := rl.Vector3Subtract(relVel, rl.Vector3Scale(up, rl.Vector3DotProduct(relVel, up)))
	velTan = rl.Vector3Scale(velTan, 0.9)
	velRad := rl.Vector3Scale(up, rl.Vector3DotProduct(relVel, up))
	relVel = rl.Vector3Add(velRad, velTan)

	rel = rl.Vector3Add(rel, rl.Vector3Scale(relVel, dt))

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

	f.RelPos = rel
	f.RelVel = relVel
	f.Pos = rl.Vector3Add(f.EarthPos, rel)
	f.Vel = rl.Vector3Add(f.EarthVel, relVel)

	f.parallelTransport(oldPos, f.Pos)
	f.TangentForward = f.projectToTangent(f.TangentForward)
}
