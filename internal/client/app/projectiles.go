package app

import (
	"math"
	"time"

	rl "github.com/gen2brain/raylib-go/raylib"
)

func (a *App) updateProjectiles() {
	if len(a.projectiles) == 0 {
		return
	}
	now := time.Now()
	dt := rl.GetFrameTime()
	const speed = 60.0
	const ttl = 1.5
	const mobR = 2.5     // радиус моба (куб 2x2x2 + запас)
	const mammothR = 3.0 // радиус мамонта
	const spearR = 0.5

	mobs := a.nc.Mobs()
	mammoths := a.nc.Mammoths()

	alive := a.projectiles[:0]
	for _, p := range a.projectiles {
		if now.Sub(p.spawn).Seconds() > ttl {
			continue
		}
		prev := p.pos
		p.pos = rl.Vector3Add(p.pos, rl.Vector3Scale(p.dir, speed*dt))

		// 1. Сначала мобы — приоритетнее.
		var hitMobID string
		for _, m := range mobs {
			c := rl.NewVector3(m.X, m.Y, m.Z)
			if segSphereHit(prev, p.pos, c, mobR+spearR) {
				hitMobID = m.ID
				break
			}
		}
		if hitMobID != "" {
			_ = a.nc.HitMob(hitMobID)
			a.log.Info("hit mob (LMB throw)", "mob", hitMobID)
			continue
		}

		// 2. Потом мамонты.
		var hitID string
		for _, m := range mammoths {
			c := rl.NewVector3(m.X, m.Y, m.Z)
			if segSphereHit(prev, p.pos, c, mammothR+spearR) {
				hitID = m.ID
				break
			}
		}
		if hitID != "" {
			_ = a.nc.HitMammoth(hitID)
			a.log.Info("local hit detected", "mammoth", hitID)
			continue
		}
		alive = append(alive, p)
	}
	a.projectiles = alive
}

func segSphereHit(a, b, c rl.Vector3, r float32) bool {
	ab := rl.Vector3Subtract(b, a)
	ac := rl.Vector3Subtract(c, a)
	abLen2 := rl.Vector3DotProduct(ab, ab)
	if abLen2 < 1e-6 {
		d := rl.Vector3Subtract(a, c)
		return rl.Vector3DotProduct(d, d) < r*r
	}
	t := rl.Vector3DotProduct(ac, ab) / abLen2
	if t < 0 {
		t = 0
	} else if t > 1 {
		t = 1
	}
	closest := rl.Vector3Add(a, rl.Vector3Scale(ab, t))
	d := rl.Vector3Subtract(closest, c)
	return rl.Vector3DotProduct(d, d) < r*r
}

func (a *App) drawProjectiles() {
	shaftCol := rl.NewColor(140, 100, 60, 255)
	tipCol := rl.NewColor(220, 220, 200, 255)

	for _, p := range a.projectiles {
		dir := rl.Vector3Normalize(p.dir)
		yaw := float32(math.Atan2(float64(dir.X), float64(dir.Z))) * 180 / math.Pi
		pitch := -float32(math.Asin(float64(dir.Y))) * 180 / math.Pi

		rl.PushMatrix()
		rl.Translatef(p.pos.X, p.pos.Y, p.pos.Z)
		rl.Rotatef(yaw, 0, 1, 0)
		rl.Rotatef(pitch, 1, 0, 0)

		// Древко (цилиндр вдоль +Z).
		rl.PushMatrix()
		rl.Rotatef(90, 1, 0, 0)
		rl.DrawCylinder(rl.NewVector3(0, 0, 0), 0.025, 0.025, 1.0, 8, shaftCol)
		rl.PopMatrix()

		// Наконечник — конус.
		rl.PushMatrix()
		rl.Translatef(0, 0, 0.6)
		rl.Rotatef(-90, 1, 0, 0)
		rl.DrawCylinder(rl.NewVector3(0, 0, 0), 0.06, 0.001, 0.25, 8, tipCol)
		rl.PopMatrix()

		rl.PopMatrix()
	}
}
