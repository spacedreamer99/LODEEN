package app

import (
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
	for _, p := range a.projectiles {
		rl.DrawSphere(p.pos, 0.4, rl.NewColor(255, 220, 60, 255))
		rl.DrawSphereWires(p.pos, 0.4, 12, 12, rl.NewColor(180, 140, 20, 255))
	}
}
