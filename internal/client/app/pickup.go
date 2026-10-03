package app

import (
	"math"

	rl "github.com/gen2brain/raylib-go/raylib"

	"github.com/spacedreamer99/lodeen/internal/shared/protocol"
)

func (a *App) tryPickup() {
	if a.flight == nil {
		return
	}
	const pickupRange = 5.0
	ep := a.nc.EarthPos()
	pos := rl.NewVector3(
		a.flight.Pos.X-ep.X,
		a.flight.Pos.Y-ep.Y,
		a.flight.Pos.Z-ep.Z,
	)
	var closest *protocol.Resource
	closestDist := float32(pickupRange)
	for _, r := range a.nc.Resources() {
		dx := r.X - pos.X
		dy := r.Y - pos.Y
		dz := r.Z - pos.Z
		d := dx*dx + dy*dy + dz*dz
		if d < closestDist*closestDist {
			closestDist = float32(math.Sqrt(float64(d)))
			rr := r
			closest = &rr
		}
	}
	if closest == nil {
		a.log.Info("pickup: no resource in range")
		return
	}
	a.log.Info("pickup: sending", "id", closest.ID, "type", closest.Type, "dist", closestDist)
	_ = a.nc.PickupItem(closest.ID)
}
