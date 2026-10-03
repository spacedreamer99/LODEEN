package app

import (
	"fmt"
	"time"

	rl "github.com/gen2brain/raylib-go/raylib"

	"github.com/spacedreamer99/lodeen/internal/client/state"
	"github.com/spacedreamer99/lodeen/internal/client/ui"
	"github.com/spacedreamer99/lodeen/internal/shared/protocol"
)

func (a *App) update(dt float32) {
	// Проверка телепорта (чит /tp).
	select {
	case tp := <-a.nc.Teleport():
		newPos := rl.NewVector3(tp.X, tp.Y, tp.Z)
		a.flight.Pos = newPos
		a.flight.Vel = rl.NewVector3(0, 0, 0)
		a.flight.RelInit = false
		a.flight.HelioInit = false
		a.camSmoothInit = false
		a.earthPosInit = false
		a.earthHistory = nil
		a.log.Info("teleport", "x", tp.X, "y", tp.Y, "z", tp.Z)
	default:
	}

	// Синк тел: буферизация по ТИКАМ сервера (не по времени прибытия).
	// Устраняет джиттер снапшотов при рендере.
	serverEP := a.nc.EarthPos()
	serverEV := a.nc.EarthVel()
	serverTick := a.nc.LastSnapshotTick()

	if a.tickRate == 0 {
		a.tickRate = 20.0
	}

	// Записываем снапшот только при новом тике.
	if serverTick > 0 {
		shouldPush := len(a.earthHistory) == 0 ||
			serverTick != a.earthHistory[len(a.earthHistory)-1].tick
		if shouldPush {
			a.earthHistory = append(a.earthHistory, earthSnap{
				tick: serverTick,
				pos:  serverEP,
				vel:  serverEV,
				at:   time.Now(),
			})
			if len(a.earthHistory) > 30 {
				a.earthHistory = a.earthHistory[len(a.earthHistory)-30:]
			}
		}
	}

	// Инициализация renderTick — 3 тика назад от последнего.
	if !a.renderTickInit && len(a.earthHistory) > 0 {
		a.renderTick = float64(a.earthHistory[len(a.earthHistory)-1].tick) - 3.0
		a.renderTickInit = true
	}

	// Продвигаем renderTick на dt * tickRate.
	if a.renderTickInit {
		a.renderTick += float64(dt) * a.tickRate
	}

	// Клэмп в границах буфера.
	if len(a.earthHistory) >= 2 {
		oldest := float64(a.earthHistory[0].tick)
		newest := float64(a.earthHistory[len(a.earthHistory)-1].tick)
		minRT := oldest
		maxRT := newest - 1.0
		if a.renderTick < minRT {
			a.renderTick = minRT
		}
		if a.renderTick > maxRT {
			a.renderTick = maxRT
		}
	}

	// Интерполяция по тикам.
	var renderEP, renderEV protocol.Vector3
	if len(a.earthHistory) >= 2 {
		var s1, s2 *earthSnap
		for i := range a.earthHistory {
			if float64(a.earthHistory[i].tick) <= a.renderTick {
				s1 = &a.earthHistory[i]
				if i+1 < len(a.earthHistory) {
					s2 = &a.earthHistory[i+1]
				}
			}
		}
		if s1 == nil {
			s1 = &a.earthHistory[0]
			if len(a.earthHistory) > 1 {
				s2 = &a.earthHistory[1]
			}
		}
		if s2 == nil {
			renderEP = s1.pos
			renderEV = s1.vel
		} else {
			span := float64(s2.tick - s1.tick)
			t := 0.0
			if span > 0 {
				t = (a.renderTick - float64(s1.tick)) / span
			}
			if t < 0 {
				t = 0
			} else if t > 1 {
				t = 1
			}
			ft := float32(t)
			renderEP = protocol.Vector3{
				X: s1.pos.X + (s2.pos.X-s1.pos.X)*ft,
				Y: s1.pos.Y + (s2.pos.Y-s1.pos.Y)*ft,
				Z: s1.pos.Z + (s2.pos.Z-s1.pos.Z)*ft,
			}
			renderEV = protocol.Vector3{
				X: s1.vel.X + (s2.vel.X-s1.vel.X)*ft,
				Y: s1.vel.Y + (s2.vel.Y-s1.vel.Y)*ft,
				Z: s1.vel.Z + (s2.vel.Z-s1.vel.Z)*ft,
			}
		}
	} else if len(a.earthHistory) == 1 {
		renderEP = a.earthHistory[0].pos
		renderEV = a.earthHistory[0].vel
	} else {
		renderEP = serverEP
		renderEV = serverEV
	}
	a.earthPos = renderEP
	a.earthVel = renderEV
	a.scene.SetEarthPos(a.earthPos)

	// Planet2 — без интерполяции пока (прямо из снапшота).
	a.planet2Pos = a.nc.Planet2Pos()
	a.planet2Vel = a.nc.Planet2Vel()

	// Frame spike detection.
	if dt > 0.05 {
		a.log.Warn("frame spike", "dt", fmt.Sprintf("%.4f", dt))
	}

	if rl.IsKeyPressed(rl.KeyF3) {
		a.showDebug = !a.showDebug
	}
	if rl.IsKeyPressed(rl.KeyF8) {
		a.diagFrames = 120 // 2 секунды @ 60 FPS
		a.log.Info("DIAG START")
	}
	a.drainChat()

	a.updateProjectiles()

	// На паузе игрок жёстко привязан к Земле по relative-координатам.
	if a.mode == state.ModePaused && a.flight != nil && a.flight.HelioInit {
		// Обновляем Earth — чтобы flight знал текущую Землю.
		a.flight.EarthPos = rl.NewVector3(a.earthPos.X, a.earthPos.Y, a.earthPos.Z)
		a.flight.EarthVel = rl.NewVector3(a.earthVel.X, a.earthVel.Y, a.earthVel.Z)

		// Применяем физику (гравитация + коллизия) — персонаж падает на грунт,
		// не висит в воздухе на паузе.
		a.flight.TickPhysicsOnly(rl.GetFrameTime())

		// Синхронизируем pausedRel* из flight — release использует их.
		a.pausedRelPos = protocol.Vector3{
			X: a.flight.RelPos.X,
			Y: a.flight.RelPos.Y,
			Z: a.flight.RelPos.Z,
		}
		a.pausedRelVel = protocol.Vector3{
			X: a.flight.RelVel.X,
			Y: a.flight.RelVel.Y,
			Z: a.flight.RelVel.Z,
		}

		// Камера едет с Землёй.
		fw := a.flight.Forward()
		a.camera.Position = a.flight.Pos
		a.camera.Target = rl.Vector3Add(a.flight.Pos, rl.Vector3Scale(fw, 100.0))
		a.camera.Up = a.flight.CameraUp()
	}

	switch a.mode {
	case state.ModeMenu:
		a.updateMenu()
	case state.ModePlaying:
		a.updatePlaying(dt)
	case state.ModePaused:
		a.updatePaused()
	case state.ModeDead:
		a.updateDead()
	}

	// Диагностика — пишем каждый кадр 2 секунды.
	if a.diagFrames > 0 && a.flight != nil {
		a.diagFrames--
		rel := rl.Vector3Subtract(a.flight.Pos, a.earthPosAsRl())
		a.log.Info("DIAG",
			"flightPos", fmt.Sprintf("%.4f,%.4f,%.4f", a.flight.Pos.X, a.flight.Pos.Y, a.flight.Pos.Z),
			"camPos", fmt.Sprintf("%.4f,%.4f,%.4f", a.camSmoothPos.X, a.camSmoothPos.Y, a.camSmoothPos.Z),
			"earthPos", fmt.Sprintf("%.4f,%.4f,%.4f", a.earthPos.X, a.earthPos.Y, a.earthPos.Z),
			"earthVel", fmt.Sprintf("%.5f,%.5f,%.5f", a.earthVel.X, a.earthVel.Y, a.earthVel.Z),
			"rel", fmt.Sprintf("%.4f,%.4f,%.4f", rel.X, rel.Y, rel.Z),
			"relLen", fmt.Sprintf("%.4f", rl.Vector3Length(rel)),
			"vel", fmt.Sprintf("%.4f,%.4f,%.4f", a.flight.Vel.X, a.flight.Vel.Y, a.flight.Vel.Z),
			"renderTick", fmt.Sprintf("%.4f", a.renderTick),
			"dt", fmt.Sprintf("%.5f", dt))
		if a.diagFrames == 0 {
			a.log.Info("DIAG END")
		}
	}
}

func (a *App) drainChat() {
	for {
		select {
		case m := <-a.nc.Chat():
			a.chat.Push(m)
		default:
			return
		}
	}
}

func (a *App) updateMouseScale() {
	// UI рисуется напрямую в окно — мышь 1:1, ничего пересчитывать не надо.
}

func (a *App) updateMenu() {
	a.setCursorCaptured(false)

	ui.EditField(&a.menuNick, a.menuFocus == 0, 32)
	ui.EditField(&a.menuAddr, a.menuFocus == 1, 64)

	if rl.IsKeyPressed(rl.KeyTab) {
		a.menuFocus = (a.menuFocus + 1) % 2
	}
	if rl.IsKeyPressed(rl.KeyEnter) {
		a.startConnect()
	}
}

func (a *App) updatePaused() {
	a.setCursorCaptured(false)

	if rl.IsKeyPressed(rl.KeyF) {
		a.tryPickup()
	}
	if rl.IsKeyPressed(rl.KeyEscape) {
		// Возвращаем velocity из сохранённой relative.
		if a.flight != nil && a.flight.HelioInit {
			a.flight.Vel = rl.NewVector3(
				a.earthVel.X+a.pausedRelVel.X,
				a.earthVel.Y+a.pausedRelVel.Y,
				a.earthVel.Z+a.pausedRelVel.Z,
			)
			a.log.Info("PAUSE EXITED",
				"earth", fmt.Sprintf("%.2f,%.2f,%.2f", a.earthPos.X, a.earthPos.Y, a.earthPos.Z),
				"flightPos", fmt.Sprintf("%.2f,%.2f,%.2f", a.flight.Pos.X, a.flight.Pos.Y, a.flight.Pos.Z),
				"expectedPos", fmt.Sprintf("%.2f,%.2f,%.2f", a.earthPos.X+a.pausedRelPos.X, a.earthPos.Y+a.pausedRelPos.Y, a.earthPos.Z+a.pausedRelPos.Z),
				"flightVel", fmt.Sprintf("%.3f,%.3f,%.3f", a.flight.Vel.X, a.flight.Vel.Y, a.flight.Vel.Z))
		}
		a.mode = state.ModePlaying
	}
}
