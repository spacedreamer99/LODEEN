package app

import (
	"fmt"
	"math"
	"time"

	rl "github.com/gen2brain/raylib-go/raylib"

	"github.com/spacedreamer99/lodeen/internal/shared/protocol"
)

// handleHeldItemClick — клик с предметом в руке.
func (a *App) handleHeldItemClick(held string) {
	switch held {
	case "spear":
		a.throwSpear()
	case "fruit":
		a.useFruit()
	case "water":
		a.useWater()
	case "leash":
		a.useLeash()
	case "house", "solar", "battery", "factory":
		a.placeStructure(held)
	case "rocket":
		x, y, z, _ := a.placeForward(5.0)
		if err := a.nc.PlaceRocket(x, y, z); err != nil {
			a.log.Warn("place rocket", "err", err)
		}
		a.log.Info("place rocket sent (LMB)")
	case "boat":
		fw := a.flight.Forward()
		pos := rl.Vector3Add(a.camera.camera.Position, rl.Vector3Scale(fw, 5.0))
		yaw := float32(math.Atan2(float64(fw.X), float64(fw.Z)))
		if err := a.nc.PlaceBoat(pos.X, pos.Y, pos.Z, yaw); err != nil {
			a.log.Warn("place boat", "err", err)
		}
		a.log.Info("place boat sent (LMB)")
	case "saddle":
		if id := a.mammothInReach(); id != "" {
			if err := a.nc.SaddleMammoth(id); err != nil {
				a.log.Warn("saddle mammoth", "err", err)
			}
			a.log.Info("saddle mammoth sent (LMB)", "id", id)
		} else {
			a.log.Info("saddle: no mammoth in reach")
		}
	}
}

// handleEmptyHandClick — клик пустой рукой: ракета, фабрика, контракт, дом, сёдла.
// Возвращает true если клик открыл UI и надо выйти из updatePlaying.
func (a *App) handleEmptyHandClick() bool {
	if id := a.rocketInSight(); id != "" {
		if err := a.nc.BoardRocket(id); err != nil {
			a.log.Warn("board rocket", "err", err)
		}
		a.player.rocketID = id
		a.player.rocketBoardedAt = time.Now()
		a.log.Info("board rocket sent", "id", id)
		return true
	}
	if id := a.factoryInSight(); id != "" {
		a.ui.factoryID = id
		a.ui.showFactory = true
		a.ui.factoryOpenedAt = time.Now()
		rl.EnableCursor()
		rl.ShowCursor()
		a.log.Info("factory dialog opened", "id", id)
		return true
	}
	if id, pos := a.pinkMobInSight(); id != "" {
		a.ui.contractMobID = id
		a.ui.contractPos = pos
		a.ui.showContract = true
		a.ui.contractOpenedAt = time.Now()
		rl.EnableCursor()
		rl.ShowCursor()
		a.log.Info("contract dialog opened", "mob", id)
		return true
	}
	if id := a.houseInSight(); id != "" {
		if err := a.nc.ToggleDoor(id); err != nil {
			a.log.Warn("toggle door", "err", err)
		}
		a.log.Info("toggle door sent (LMB)", "id", id)
		return false
	}
	if id := a.saddledMammothNearby(); id != "" {
		if err := a.nc.SaddleMammoth(id); err != nil {
			a.log.Warn("saddle mammoth", "err", err)
		}
		a.log.Info("unsaddle mammoth sent (LMB)", "id", id)
	}
	return false
}

// throwSpear — метает копьё + локальный «трейсер» снаряда.
func (a *App) throwSpear() {
	fw := a.flight.Forward()
	dir := protocol.Vector3{X: fw.X, Y: fw.Y, Z: fw.Z}
	if err := a.nc.ThrowSpear(dir); err != nil {
		a.log.Warn("throw spear", "err", err)
	}
	ep := a.nc.EarthPos()
	a.player.projectiles = append(a.player.projectiles, projectile{
		pos: rl.NewVector3(
			a.camera.camera.Position.X-ep.X,
			a.camera.camera.Position.Y-ep.Y,
			a.camera.camera.Position.Z-ep.Z,
		),
		dir:   fw,
		spawn: time.Now(),
	})
	a.log.Info("spear thrown (LMB)")
}

// useFruit — приручить мамонта рядом, иначе посадить семечко в 2 юнитах перед собой.
func (a *App) useFruit() {
	if id := a.mammothInReach(); id != "" {
		if err := a.nc.TameMammoth(id); err != nil {
			a.log.Warn("tame mammoth", "err", err)
		}
		a.log.Info("tame mammoth sent (LMB)", "id", id)
		return
	}
	fw := a.flight.Forward()
	helio := rl.Vector3Add(a.camera.camera.Position, rl.Vector3Scale(fw, 2.0))
	ep := a.nc.EarthPos()
	pos := rl.NewVector3(helio.X-ep.X, helio.Y-ep.Y, helio.Z-ep.Z)
	if err := a.nc.PlantSeed(pos.X, pos.Y, pos.Z); err != nil {
		a.log.Warn("plant seed", "err", err)
	}
	a.log.Info("plant seed sent (LMB)",
		"geo", fmt.Sprintf("%.1f,%.1f,%.1f", pos.X, pos.Y, pos.Z))
}

// useWater — полить росток под прицелом.
func (a *App) useWater() {
	if id := a.seedInSight(); id != "" {
		if err := a.nc.WaterPlant(id); err != nil {
			a.log.Warn("water plant", "err", err)
		}
		a.log.Info("water plant sent (LMB)", "seed", id)
	} else {
		a.log.Debug("water: no seed in sight")
	}
}

// useLeash — накинуть поводок на мамонта рядом.
func (a *App) useLeash() {
	if id := a.mammothInReach(); id != "" {
		if err := a.nc.LeashMammoth(id); err != nil {
			a.log.Warn("leash mammoth", "err", err)
		}
		a.log.Info("leash mammoth sent (LMB)", "id", id)
	} else {
		a.log.Info("leash: no mammoth in reach")
	}
}

// placeStructure — поставить дом/панель/батарею/фабрику в 5 юнитах перед собой.
func (a *App) placeStructure(kind string) {
	x, y, z, yaw := a.placeForward(5.0)
	var err error
	switch kind {
	case "house":
		err = a.nc.PlaceHouse(x, y, z, yaw)
	case "solar":
		err = a.nc.PlaceSolar(x, y, z, yaw)
	case "battery":
		err = a.nc.PlaceBattery(x, y, z, yaw)
	case "factory":
		err = a.nc.PlaceFactory(x, y, z, yaw)
	}
	if err != nil {
		a.log.Warn("place "+kind, "err", err)
	}
	a.log.Info("place " + kind + " sent (LMB)")
}
