package app

import (
	"fmt"
	"math"
	"time"

	rl "github.com/gen2brain/raylib-go/raylib"

	"github.com/spacedreamer99/lodeen/internal/client/input"
	"github.com/spacedreamer99/lodeen/internal/client/state"
	"github.com/spacedreamer99/lodeen/internal/shared/protocol"
)

func (a *App) updatePlaying(dt float32) {
	a.syncInvSlots()

	// Пилотирование ракеты блокирует всё остальное.
	if a.rocketID != "" && a.flight != nil {
		if a.updatePilotedRocket(dt) {
			return
		}
		// Ракета не найдена / не Piloted — выходим из режима.
		a.rocketID = ""
	}

	// Диалоги блокируют всё остальное.
	if a.showContract {
		a.updateContract()
		return
	}
	if a.showFactory {
		a.updateFactory()
		return
	}
	// Обновляем HP из сети.
	if a.nc != nil {
		if hp := a.nc.OwnHP(); hp > 0 || a.hpReceived {
			a.hp = hp
			a.hpReceived = true
		}
	}
	if a.chat.Open {
		a.setCursorCaptured(false)
		if text, ok := a.chat.Update(); ok && text != "" {
			if err := a.nc.SendChat(text); err != nil {
				a.log.Warn("send chat", "err", err)
			}
		}
		return
	}

	if rl.IsKeyPressed(rl.KeyF1) && a.flight != nil {
		if a.flight.Mode == input.ModeCreative {
			a.flight.Mode = input.ModeSurvival
		} else {
			a.flight.Mode = input.ModeCreative
		}
		a.log.Info("mode changed", "mode", a.flight.Mode)
	}
	// C — открыть окно крафта (всегда)
	if rl.IsKeyPressed(rl.KeyC) {
		a.showCraft = true
		a.showInventory = false
		rl.EnableCursor()
		rl.ShowCursor()
	}
	// I — инвентарь (toggle)
	if rl.IsKeyPressed(rl.KeyI) {
		a.showInventory = !a.showInventory
		if a.showInventory {
			a.showCraft = false
			rl.EnableCursor()
			rl.ShowCursor()
		} else {
			rl.DisableCursor()
		}
	}
	// Меню открыто — стоп движению, закрытие на Esc
	if a.showInventory || a.showCraft {
		if rl.IsKeyPressed(rl.KeyEscape) {
			a.showInventory = false
			a.showCraft = false
			rl.DisableCursor()
		}
		return
	}
	// В Survival колёсико выбирает слот из первой строки инвентаря.
	if a.flight != nil && a.flight.Mode == input.ModeSurvival {
		wheel := rl.GetMouseWheelMove()
		if wheel > 0 {
			a.selectedSlot = (a.selectedSlot + 1) % 16
		} else if wheel < 0 {
			a.selectedSlot = (a.selectedSlot + 15) % 16
		}
	}
	// Отправляем на сервер, что в руке — при смене слота.
	if held := a.heldItem(); held != a.lastSentHeld {
		a.lastSentHeld = held
		if err := a.nc.SelectItem(held); err != nil {
			a.log.Warn("select item", "err", err)
		}
	}
	if rl.IsKeyPressed(rl.KeyT) {
		a.chat.Begin()
		return
	}
	if rl.IsKeyPressed(rl.KeySlash) {
		a.chat.BeginWith("/")
		return
	}
	if rl.IsKeyPressed(rl.KeyF) {
		a.tryPickup()
	}

	// E — сесть/встать с мамонта или лодки.
	if rl.IsKeyPressed(rl.KeyE) && a.flight != nil {
		if a.ridingID != "" {
			if err := a.nc.RideMammoth(""); err != nil {
				a.log.Warn("ride", "err", err)
			}
			a.ridingID = ""
			a.flight.Riding = false
			a.log.Info("dismount sent")
		} else if a.boatID != "" {
			if err := a.nc.EnterBoat(""); err != nil {
				a.log.Warn("boat exit", "err", err)
			}
			a.boatID = ""
			a.flight.InBoat = false
			a.log.Info("boat exit sent")
		} else if id := a.saddledMammothNearby(); id != "" {
			if err := a.nc.RideMammoth(id); err != nil {
				a.log.Warn("ride", "err", err)
			}
			a.ridingID = id
			a.flight.Riding = true
			a.log.Info("mount sent", "id", id)
		} else if id := a.boatNearby(); id != "" {
			if err := a.nc.EnterBoat(id); err != nil {
				a.log.Warn("boat enter", "err", err)
			}
			a.boatID = id
			a.flight.InBoat = true
			a.log.Info("boat enter sent", "id", id)
		}
	}
	// ЛКМ — действие зависит от того, что в руке.
	if rl.IsMouseButtonPressed(rl.MouseLeftButton) && a.flight != nil {
		// Приоритет: источник пресной воды. Клик по нему всегда даёт воду.
		if id := a.wellInSight(); id != "" {
			if err := a.nc.TakeWater(id); err != nil {
				a.log.Warn("take water", "err", err)
			}
			a.log.Info("take water sent (LMB)", "well", id)
			// не возвращаемся — просто пропускаем switch ниже
			goto afterLMB
		}
		switch a.heldItem() {
		case "spear":
			fw := a.flight.Forward()
			dir := protocol.Vector3{X: fw.X, Y: fw.Y, Z: fw.Z}
			if err := a.nc.ThrowSpear(dir); err != nil {
				a.log.Warn("throw spear", "err", err)
			}
			a.projectiles = append(a.projectiles, projectile{
				pos:   a.camera.Position,
				dir:   fw,
				spawn: time.Now(),
			})
			a.log.Info("spear thrown (LMB)")
		case "fruit":
			// Приоритет: мамонт рядом → приручить или покормить.
			if id := a.mammothInReach(); id != "" {
				if err := a.nc.TameMammoth(id); err != nil {
					a.log.Warn("tame mammoth", "err", err)
				}
				a.log.Info("tame mammoth sent (LMB)", "id", id)
			} else {
				// Иначе — посадить семечко в 2 юнитах перед собой.
				fw := a.flight.Forward()
				pos := rl.Vector3Add(a.camera.Position, rl.Vector3Scale(fw, 2.0))
				if err := a.nc.PlantSeed(pos.X, pos.Y, pos.Z); err != nil {
					a.log.Warn("plant seed", "err", err)
				}
				a.log.Info("plant seed sent (LMB)")
			}
		case "water":
			// Полить росток, на который смотрит игрок.
			if id := a.seedInSight(); id != "" {
				if err := a.nc.WaterPlant(id); err != nil {
					a.log.Warn("water plant", "err", err)
				}
				a.log.Info("water plant sent (LMB)", "seed", id)
			} else {
				a.log.Debug("water: no seed in sight")
			}
		case "leash":
			if id := a.mammothInReach(); id != "" {
				if err := a.nc.LeashMammoth(id); err != nil {
					a.log.Warn("leash mammoth", "err", err)
				}
				a.log.Info("leash mammoth sent (LMB)", "id", id)
			} else {
				a.log.Info("leash: no mammoth in reach")
			}
		case "house":
			x, y, z, yaw := a.placeForward(5.0)
			if err := a.nc.PlaceHouse(x, y, z, yaw); err != nil {
				a.log.Warn("place house", "err", err)
			}
			a.log.Info("place house sent (LMB)")
		case "solar":
			x, y, z, yaw := a.placeForward(5.0)
			if err := a.nc.PlaceSolar(x, y, z, yaw); err != nil {
				a.log.Warn("place solar", "err", err)
			}
			a.log.Info("place solar sent (LMB)")
		case "battery":
			x, y, z, yaw := a.placeForward(5.0)
			if err := a.nc.PlaceBattery(x, y, z, yaw); err != nil {
				a.log.Warn("place battery", "err", err)
			}
			a.log.Info("place battery sent (LMB)")
		case "factory":
			x, y, z, yaw := a.placeForward(5.0)
			if err := a.nc.PlaceFactory(x, y, z, yaw); err != nil {
				a.log.Warn("place factory", "err", err)
			}
			a.log.Info("place factory sent (LMB)")
		case "rocket":
			x, y, z, _ := a.placeForward(5.0)
			if err := a.nc.PlaceRocket(x, y, z); err != nil {
				a.log.Warn("place rocket", "err", err)
			}
			a.log.Info("place rocket sent (LMB)")
		case "boat":
			// Поставить лодку в 5 юнитах перед собой (только на воду).
			fw := a.flight.Forward()
			pos := rl.Vector3Add(a.camera.Position, rl.Vector3Scale(fw, 5.0))
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
		case "":
			if id := a.rocketInSight(); id != "" {
				if err := a.nc.BoardRocket(id); err != nil {
					a.log.Warn("board rocket", "err", err)
				}
				a.rocketID = id
				a.rocketBoardedAt = time.Now()
				a.log.Info("board rocket sent", "id", id)
				return
			}
			if id := a.factoryInSight(); id != "" {
				a.factoryID = id
				a.showFactory = true
				a.factoryOpenedAt = time.Now()
				rl.EnableCursor()
				rl.ShowCursor()
				a.log.Info("factory dialog opened", "id", id)
				return
			}
			if id, pos := a.pinkMobInSight(); id != "" {
				a.contractMobID = id
				a.contractPos = pos
				a.showContract = true
				a.contractOpenedAt = time.Now()
				rl.EnableCursor()
				rl.ShowCursor()
				a.log.Info("contract dialog opened", "mob", id)
				return
			}
			if id := a.houseInSight(); id != "" {
				if err := a.nc.ToggleDoor(id); err != nil {
					a.log.Warn("toggle door", "err", err)
				}
				a.log.Info("toggle door sent (LMB)", "id", id)
			} else if id := a.saddledMammothNearby(); id != "" {
				if err := a.nc.SaddleMammoth(id); err != nil {
					a.log.Warn("unsaddle mammoth", "err", err)
				}
				a.log.Info("unsaddle mammoth sent (LMB)", "id", id)
			}
		}
	afterLMB:
	}
	// Смерть → экран смерти.
	if a.hpReceived && a.myHP() <= 0 {
		a.log.Info("player HP zero, switching to death screen")
		a.mode = state.ModeDead
		rl.EnableCursor()
		rl.ShowCursor()
		if a.nc != nil {
			a.nc.Close()
		}
		return
	}
	if rl.IsKeyPressed(rl.KeyEscape) {
		// Сохраняем относительную позицию к Земле.
		if a.flight != nil && a.flight.HelioInit {
			a.pausedRelPos = protocol.Vector3{
				X: a.flight.Pos.X - a.earthPos.X,
				Y: a.flight.Pos.Y - a.earthPos.Y,
				Z: a.flight.Pos.Z - a.earthPos.Z,
			}
			a.pausedRelVel = protocol.Vector3{
				X: a.flight.Vel.X - a.earthVel.X,
				Y: a.flight.Vel.Y - a.earthVel.Y,
				Z: a.flight.Vel.Z - a.earthVel.Z,
			}
			a.log.Info("PAUSE ENTERED",
				"helio", fmt.Sprintf("%.2f,%.2f,%.2f", a.flight.Pos.X, a.flight.Pos.Y, a.flight.Pos.Z),
				"earth", fmt.Sprintf("%.2f,%.2f,%.2f", a.earthPos.X, a.earthPos.Y, a.earthPos.Z),
				"relPos", fmt.Sprintf("%.2f,%.2f,%.2f", a.pausedRelPos.X, a.pausedRelPos.Y, a.pausedRelPos.Z),
				"relVel", fmt.Sprintf("%.3f,%.3f,%.3f", a.pausedRelVel.X, a.pausedRelVel.Y, a.pausedRelVel.Z))
		}
		a.mode = state.ModePaused
		return
	}

	if !rl.IsWindowFocused() {
		a.setCursorCaptured(false)
		// Alt+Tab = заморозка относительно Земли (как пауза).
		if a.flight != nil && a.flight.HelioInit {
			if !a.unfocusFreeze {
				a.unfocusRelPos = protocol.Vector3{
					X: a.flight.Pos.X - a.earthPos.X,
					Y: a.flight.Pos.Y - a.earthPos.Y,
					Z: a.flight.Pos.Z - a.earthPos.Z,
				}
				a.unfocusRelVel = protocol.Vector3{
					X: a.flight.Vel.X - a.earthVel.X,
					Y: a.flight.Vel.Y - a.earthVel.Y,
					Z: a.flight.Vel.Z - a.earthVel.Z,
				}
				a.unfocusFreeze = true
				a.log.Info("UNFOCUS freeze",
					"relPos", fmt.Sprintf("%.2f,%.2f,%.2f", a.unfocusRelPos.X, a.unfocusRelPos.Y, a.unfocusRelPos.Z))
			}
			a.flight.Pos = rl.NewVector3(
				a.earthPos.X+a.unfocusRelPos.X,
				a.earthPos.Y+a.unfocusRelPos.Y,
				a.earthPos.Z+a.unfocusRelPos.Z,
			)
			a.flight.Vel = rl.NewVector3(
				a.earthVel.X+a.unfocusRelVel.X,
				a.earthVel.Y+a.unfocusRelVel.Y,
				a.earthVel.Z+a.unfocusRelVel.Z,
			)
			fw := a.flight.Forward()
			a.camera.Position = a.flight.Pos
			a.camera.Target = rl.Vector3Add(a.flight.Pos, rl.Vector3Scale(fw, 100.0))
			a.camera.Up = a.flight.CameraUp()
		}
		return
	}

	if a.unfocusFreeze {
		a.unfocusFreeze = false
		a.log.Info("UNFOCUS restored",
			"helio", fmt.Sprintf("%.2f,%.2f,%.2f", a.flight.Pos.X, a.flight.Pos.Y, a.flight.Pos.Z),
			"earth", fmt.Sprintf("%.2f,%.2f,%.2f", a.earthPos.X, a.earthPos.Y, a.earthPos.Z))
	}

	a.setCursorCaptured(true)

	if a.flight == nil {
		return
	}

	// Передаём позиции тел в flight каждый кадр.
	a.flight.EarthPos = rl.NewVector3(a.earthPos.X, a.earthPos.Y, a.earthPos.Z)
	a.flight.SunPos = rl.NewVector3(protocol.SunPos.X, protocol.SunPos.Y, protocol.SunPos.Z)
	a.flight.EarthVel = rl.NewVector3(a.earthVel.X, a.earthVel.Y, a.earthVel.Z)

	// HelioInit — только когда EarthPos реально пришла и не (0,0,0).
	// Проверяем по длине вектора: |earthPos| > 100 (Земля точно не в центре).
	epLen := a.earthPos.X*a.earthPos.X + a.earthPos.Y*a.earthPos.Y + a.earthPos.Z*a.earthPos.Z
	if !a.flight.HelioInit && epLen > 100*100 {
		a.flight.Pos = rl.Vector3Add(a.flight.Pos, a.flight.EarthPos)
		a.flight.HelioInit = true
		a.log.Info("flight helio init",
			"spawnGeo", fmt.Sprintf("%.2f,%.2f,%.2f", a.flight.Pos.X-a.flight.EarthPos.X, a.flight.Pos.Y-a.flight.EarthPos.Y, a.flight.Pos.Z-a.flight.EarthPos.Z),
			"earth", fmt.Sprintf("%.2f,%.2f,%.2f", a.flight.EarthPos.X, a.flight.EarthPos.Y, a.flight.EarthPos.Z),
			"helio", fmt.Sprintf("%.2f,%.2f,%.2f", a.flight.Pos.X, a.flight.Pos.Y, a.flight.Pos.Z))
	}

	md := rl.GetMouseDelta()
	a.flight.Update(dt, md)

	a.debugFrame++
	keys := readKeysString()
	mouse := readMouseString()

	now := time.Now()
	changed := keys != a.lastKeys || mouse != a.lastMouse || md.X != 0 || md.Y != 0
	idle := now.Sub(a.lastLogAt) > time.Second
	if (changed || idle) && now.Sub(a.lastLogAt) >= 30*time.Millisecond {
		edgeKeys := ""
		if keys != a.lastKeys {
			edgeKeys = keys
		}

		// Полное состояние для отладки дёргания.
		relX := a.flight.Pos.X - a.earthPos.X
		relY := a.flight.Pos.Y - a.earthPos.Y
		relZ := a.flight.Pos.Z - a.earthPos.Z
		relDist := float32(math.Sqrt(float64(relX*relX + relY*relY + relZ*relZ)))
		surfaceR := protocol.SurfaceRadius(protocol.Vector3{X: relX, Y: relY, Z: relZ})
		minR := surfaceR + protocol.PlayerHeight
		onGround := relDist <= minR+0.5

		a.log.Info("state",
			"dt", fmt.Sprintf("%.4f", dt),
			"helioPos", fmt.Sprintf("%.2f,%.2f,%.2f", a.flight.Pos.X, a.flight.Pos.Y, a.flight.Pos.Z),
			"earthPos", fmt.Sprintf("%.2f,%.2f,%.2f", a.earthPos.X, a.earthPos.Y, a.earthPos.Z),
			"earthVel", fmt.Sprintf("%.3f,%.3f,%.3f", a.earthVel.X, a.earthVel.Y, a.earthVel.Z),
			"relDist", fmt.Sprintf("%.3f", relDist),
			"minR", fmt.Sprintf("%.3f", minR),
			"onGround", onGround,
			"vel", fmt.Sprintf("%.2f,%.2f,%.2f", a.flight.Vel.X, a.flight.Vel.Y, a.flight.Vel.Z),
			"velLen", fmt.Sprintf("%.2f", rl.Vector3Length(a.flight.Vel)),
			"edgeK", edgeKeys,
			"keys", keys,
		)

		a.lastLogAt = now
		a.lastPos = a.flight.Pos
		a.lastKeys = keys
		a.lastMouse = mouse
	}
	fw := a.flight.Forward()
	targetNew := rl.Vector3Add(a.flight.Pos, rl.Vector3Scale(fw, 100.0))

	// Сглаживание камеры — убирает мелкое визуальное дрожание.
	if !a.camSmoothInit {
		a.camSmoothPos = a.flight.Pos
		a.camSmoothTarget = targetNew
		a.camSmoothInit = true
	} else {
		const camAlpha = float32(0.5)
		a.camSmoothPos = rl.Vector3Add(a.camSmoothPos,
			rl.Vector3Scale(rl.Vector3Subtract(a.flight.Pos, a.camSmoothPos), camAlpha))
		a.camSmoothTarget = rl.Vector3Add(a.camSmoothTarget,
			rl.Vector3Scale(rl.Vector3Subtract(targetNew, a.camSmoothTarget), camAlpha))
	}
	a.camera.Position = a.camSmoothPos
	a.camera.Target = a.camSmoothTarget
	a.camera.Up = a.flight.CameraUp()

	yawF := float32(math.Atan2(float64(-fw.X), float64(-fw.Z)))
	pitchF := float32(math.Asin(float64(fw.Y)))
	// Серверу — geo-координаты (минус EarthPos).
	a.nc.SetState(protocol.PlayerState{
		X:   a.flight.Pos.X - a.earthPos.X,
		Y:   a.flight.Pos.Y - a.earthPos.Y,
		Z:   a.flight.Pos.Z - a.earthPos.Z,
		Yaw: yawF, Pitch: pitchF,
	})
}
