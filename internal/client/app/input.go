package app

import (
	"math"

	rl "github.com/gen2brain/raylib-go/raylib"
)

func readKeysString() string {
	keys := ""
	if rl.IsKeyDown(rl.KeyW) {
		keys += "W"
	}
	if rl.IsKeyDown(rl.KeyA) {
		keys += "A"
	}
	if rl.IsKeyDown(rl.KeyS) {
		keys += "S"
	}
	if rl.IsKeyDown(rl.KeyD) {
		keys += "D"
	}
	if rl.IsKeyDown(rl.KeySpace) {
		keys += "Spc"
	}
	if rl.IsKeyDown(rl.KeyLeftShift) {
		keys += "LSh"
	}
	if rl.IsKeyDown(rl.KeyRightShift) {
		keys += "RSh"
	}
	if rl.IsKeyDown(rl.KeyQ) {
		keys += "Q"
	}
	if rl.IsKeyDown(rl.KeyE) {
		keys += "E"
	}
	if rl.IsKeyDown(rl.KeyLeftControl) {
		keys += "LCt"
	}
	if rl.IsKeyDown(rl.KeyRightControl) {
		keys += "RCt"
	}
	if keys == "" {
		keys = "-"
	}
	return keys
}

func readMouseString() string {
	mouse := ""
	if rl.IsMouseButtonDown(rl.MouseLeftButton) {
		mouse += "L"
	}
	if rl.IsMouseButtonDown(rl.MouseRightButton) {
		mouse += "R"
	}
	if rl.IsMouseButtonDown(rl.MouseMiddleButton) {
		mouse += "M"
	}
	if mouse == "" {
		mouse = "-"
	}
	return mouse
}

func (a *App) wellInSight() string {
	wells := a.nc.Wells()
	cam := a.camera.Position
	fw := a.flight.Forward()
	var bestID string
	bestD2 := float32(36.0) // 6^2
	for _, w := range wells {
		dx := w.X - cam.X
		dy := w.Y - cam.Y
		dz := w.Z - cam.Z
		d2 := dx*dx + dy*dy + dz*dz
		if d2 > 36.0 {
			continue
		}
		d := float32(math.Sqrt(float64(d2)))
		if d > 0.01 {
			dot := (dx*fw.X + dy*fw.Y + dz*fw.Z) / d
			if dot < 0.5 {
				continue
			}
		}
		if d2 < bestD2 {
			bestD2 = d2
			bestID = w.ID
		}
	}
	return bestID
}

func (a *App) rocketInSight() string {
	rs := a.nc.Rockets()
	cam := a.camera.Position
	fw := a.flight.Forward()
	var bestID string
	bestD2 := float32(64.0)
	for _, r := range rs {
		dx := r.X - cam.X
		dy := r.Y - cam.Y
		dz := r.Z - cam.Z
		d2 := dx*dx + dy*dy + dz*dz
		if d2 > 64.0 {
			continue
		}
		d := float32(math.Sqrt(float64(d2)))
		if d > 0.01 {
			dot := (dx*fw.X + dy*fw.Y + dz*fw.Z) / d
			if dot < 0.3 {
				continue
			}
		}
		if d2 < bestD2 {
			bestD2 = d2
			bestID = r.ID
		}
	}
	return bestID
}

func (a *App) factoryInSight() string {
	fs := a.nc.Factories()
	cam := a.camera.Position
	fw := a.flight.Forward()
	var bestID string
	bestD2 := float32(36.0)
	for _, f := range fs {
		dx := f.X - cam.X
		dy := f.Y - cam.Y
		dz := f.Z - cam.Z
		d2 := dx*dx + dy*dy + dz*dz
		if d2 > 36.0 {
			continue
		}
		d := float32(math.Sqrt(float64(d2)))
		if d > 0.01 {
			dot := (dx*fw.X + dy*fw.Y + dz*fw.Z) / d
			if dot < 0.4 {
				continue
			}
		}
		if d2 < bestD2 {
			bestD2 = d2
			bestID = f.ID
		}
	}
	return bestID
}

func (a *App) pinkMobInSight() (string, rl.Vector3) {
	mobs := a.nc.Mobs()
	cam := a.camera.Position
	fw := a.flight.Forward()
	var bestID string
	var bestPos rl.Vector3
	bestD2 := float32(36.0)
	for _, m := range mobs {
		if m.Kind != "pink" {
			continue
		}
		dx := m.X - cam.X
		dy := m.Y - cam.Y
		dz := m.Z - cam.Z
		d2 := dx*dx + dy*dy + dz*dz
		if d2 > 36.0 {
			continue
		}
		d := float32(math.Sqrt(float64(d2)))
		if d > 0.01 {
			dot := (dx*fw.X + dy*fw.Y + dz*fw.Z) / d
			if dot < 0.4 {
				continue
			}
		}
		if d2 < bestD2 {
			bestD2 = d2
			bestID = m.ID
			bestPos = rl.NewVector3(m.X, m.Y, m.Z)
		}
	}
	return bestID, bestPos
}

func (a *App) houseInSight() string {
	hs := a.nc.Houses()
	cam := a.camera.Position
	var bestID string
	bestD2 := float32(36.0)
	for _, h := range hs {
		dx := h.X - cam.X
		dy := h.Y - cam.Y
		dz := h.Z - cam.Z
		d2 := dx*dx + dy*dy + dz*dz
		if d2 < bestD2 {
			bestD2 = d2
			bestID = h.ID
		}
	}
	return bestID
}

func (a *App) placeForward(dist float32) (x, y, z, yaw float32) {
	fw := a.flight.Forward()
	pos := rl.Vector3Add(a.camera.Position, rl.Vector3Scale(fw, dist))
	yaw = float32(math.Atan2(float64(fw.X), float64(fw.Z)))
	return pos.X, pos.Y, pos.Z, yaw
}

func (a *App) boatNearby() string {
	bs := a.nc.Boats()
	cam := a.camera.Position
	me := a.nc.PlayerID()
	var bestID string
	bestD2 := float32(64.0)
	for _, b := range bs {
		if b.RiderID != "" && b.RiderID != me {
			continue
		}
		dx := b.X - cam.X
		dy := b.Y - cam.Y
		dz := b.Z - cam.Z
		d2 := dx*dx + dy*dy + dz*dz
		if d2 < bestD2 {
			bestD2 = d2
			bestID = b.ID
		}
	}
	return bestID
}

func (a *App) saddledMammothNearby() string {
	ms := a.nc.Mammoths()
	cam := a.camera.Position
	me := a.nc.PlayerID()
	var bestID string
	bestD2 := float32(36.0)
	for _, m := range ms {
		if !m.Saddle {
			continue
		}
		if m.RiderID != "" && m.RiderID != me {
			continue
		}
		dx := m.X - cam.X
		dy := m.Y - cam.Y
		dz := m.Z - cam.Z
		d2 := dx*dx + dy*dy + dz*dz
		if d2 < bestD2 {
			bestD2 = d2
			bestID = m.ID
		}
	}
	return bestID
}

func (a *App) mammothInReach() string {
	ms := a.nc.Mammoths()
	cam := a.camera.Position
	fw := a.flight.Forward()
	var bestID string
	bestD2 := float32(25.0)
	for _, m := range ms {
		dx := m.X - cam.X
		dy := m.Y - cam.Y
		dz := m.Z - cam.Z
		d2 := dx*dx + dy*dy + dz*dz
		if d2 > 25.0 {
			continue
		}
		d := float32(math.Sqrt(float64(d2)))
		if d > 0.01 {
			dot := (dx*fw.X + dy*fw.Y + dz*fw.Z) / d
			if dot < 0.4 {
				continue
			}
		}
		if d2 < bestD2 {
			bestD2 = d2
			bestID = m.ID
		}
	}
	return bestID
}

func (a *App) seedInSight() string {
	res := a.nc.Resources()
	cam := a.camera.Position
	fw := a.flight.Forward()
	var bestID string
	bestD2 := float32(25.0) // 5^2
	for _, r := range res {
		if r.Type != "seed" {
			continue
		}
		dx := r.X - cam.X
		dy := r.Y - cam.Y
		dz := r.Z - cam.Z
		d2 := dx*dx + dy*dy + dz*dz
		if d2 > 25.0 {
			continue
		}
		d := float32(math.Sqrt(float64(d2)))
		if d > 0.01 {
			dot := (dx*fw.X + dy*fw.Y + dz*fw.Z) / d
			if dot < 0.6 {
				continue
			}
		}
		if d2 < bestD2 {
			bestD2 = d2
			bestID = r.ID
		}
	}
	return bestID
}
