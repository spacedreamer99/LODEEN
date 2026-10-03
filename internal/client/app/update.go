package app

import (
	"fmt"
	"time"

	rl "github.com/gen2brain/raylib-go/raylib"

	"github.com/spacedreamer99/lodeen/internal/client/state"
	"github.com/spacedreamer99/lodeen/internal/client/ui"
	"github.com/spacedreamer99/lodeen/internal/shared/protocol"
)

// update — оркестратор кадра. Каждый helper делает одну вещь.
func (a *App) update(dt float32) {
	a.handleTeleport()
	a.updateEarthHistory(dt)
	a.updatePlanet2()

	if dt > 0.05 {
		a.log.Warn("frame spike", "dt", fmt.Sprintf("%.4f", dt))
	}

	a.handleDebugKeys()
	a.drainChat()
	a.updateProjectiles()

	a.updatePausePhysics()

	a.dispatchMode(dt)
	a.updateDiag(dt)
}

// --- Телепорт ---

func (a *App) handleTeleport() {
	select {
	case tp := <-a.nc.Teleport():
		a.flight.Pos = rl.NewVector3(tp.X, tp.Y, tp.Z)
		a.flight.Vel = rl.NewVector3(0, 0, 0)
		a.flight.RelInit = false
		a.flight.HelioInit = false
		a.camera.camSmoothInit = false
		a.world.earthPosInit = false
		a.world.earthHistory = nil
		a.log.Info("teleport", "x", tp.X, "y", tp.Y, "z", tp.Z)
	default:
	}
}

// --- Earth buffer + renderTick + интерполяция ---

// updateEarthHistory буферизует снапшоты по тикам сервера, двигает renderTick
// и интерполирует earthPos/earthVel.
func (a *App) updateEarthHistory(dt float32) {
	serverEP := a.nc.EarthPos()
	serverEV := a.nc.EarthVel()
	serverTick := a.nc.LastSnapshotTick()

	if a.camera.tickRate == 0 {
		a.camera.tickRate = 20.0
	}

	a.pushEarthSnapshot(serverTick, serverEP, serverEV)
	a.initRenderTick()
	a.advanceRenderTick(dt)

	renderEP, renderEV := a.interpolateEarth(serverEP, serverEV)
	a.world.earthPos = renderEP
	a.world.earthVel = renderEV
	a.scene.SetEarthPos(a.world.earthPos)
}

// pushEarthSnapshot добавляет снапшот в буфер только при новом тике.
func (a *App) pushEarthSnapshot(tick uint64, ep, ev protocol.Vector3) {
	if tick == 0 {
		return
	}
	if len(a.world.earthHistory) > 0 && tick == a.world.earthHistory[len(a.world.earthHistory)-1].tick {
		return
	}
	a.world.earthHistory = append(a.world.earthHistory, earthSnap{
		tick: tick,
		pos:  ep,
		vel:  ev,
		at:   time.Now(),
	})
	if len(a.world.earthHistory) > 30 {
		a.world.earthHistory = a.world.earthHistory[len(a.world.earthHistory)-30:]
	}
}

// initRenderTick ставит renderTick на 3 тика назад от последнего (запас для интерполяции).
func (a *App) initRenderTick() {
	if a.camera.renderTickInit || len(a.world.earthHistory) == 0 {
		return
	}
	a.camera.renderTick = float64(a.world.earthHistory[len(a.world.earthHistory)-1].tick) - 3.0
	a.camera.renderTickInit = true
}

// advanceRenderTick продвигает renderTick и клэмпит его в границах буфера.
func (a *App) advanceRenderTick(dt float32) {
	if !a.camera.renderTickInit {
		return
	}
	a.camera.renderTick += float64(dt) * a.camera.tickRate

	if len(a.world.earthHistory) < 2 {
		return
	}
	oldest := float64(a.world.earthHistory[0].tick)
	newest := float64(a.world.earthHistory[len(a.world.earthHistory)-1].tick)
	if a.camera.renderTick < oldest {
		a.camera.renderTick = oldest
	}
	if a.camera.renderTick > newest-1.0 {
		a.camera.renderTick = newest - 1.0
	}
}

// interpolateEarth возвращает позицию и скорость Земли для текущего renderTick.
func (a *App) interpolateEarth(fallbackEP, fallbackEV protocol.Vector3) (protocol.Vector3, protocol.Vector3) {
	if len(a.world.earthHistory) == 0 {
		return fallbackEP, fallbackEV
	}
	if len(a.world.earthHistory) == 1 {
		return a.world.earthHistory[0].pos, a.world.earthHistory[0].vel
	}

	s1, s2 := a.findEarthSurrounding()
	if s1 == nil {
		return fallbackEP, fallbackEV
	}
	if s2 == nil {
		return s1.pos, s1.vel
	}

	span := float64(s2.tick - s1.tick)
	t := 0.0
	if span > 0 {
		t = (a.camera.renderTick - float64(s1.tick)) / span
	}
	if t < 0 {
		t = 0
	} else if t > 1 {
		t = 1
	}
	ft := float32(t)

	return protocol.Vector3{
		X: s1.pos.X + (s2.pos.X-s1.pos.X)*ft,
		Y: s1.pos.Y + (s2.pos.Y-s1.pos.Y)*ft,
		Z: s1.pos.Z + (s2.pos.Z-s1.pos.Z)*ft,
	}, protocol.Vector3{
		X: s1.vel.X + (s2.vel.X-s1.vel.X)*ft,
		Y: s1.vel.Y + (s2.vel.Y-s1.vel.Y)*ft,
		Z: s1.vel.Z + (s2.vel.Z-s1.vel.Z)*ft,
	}
}

// findEarthSurrounding находит два снапшота вокруг renderTick.
func (a *App) findEarthSurrounding() (*earthSnap, *earthSnap) {
	var s1, s2 *earthSnap
	for i := range a.world.earthHistory {
		if float64(a.world.earthHistory[i].tick) <= a.camera.renderTick {
			s1 = &a.world.earthHistory[i]
			if i+1 < len(a.world.earthHistory) {
				s2 = &a.world.earthHistory[i+1]
			}
		}
	}
	if s1 == nil {
		s1 = &a.world.earthHistory[0]
		if len(a.world.earthHistory) > 1 {
			s2 = &a.world.earthHistory[1]
		}
	}
	return s1, s2
}

// --- Planet2 ---

func (a *App) updatePlanet2() {
	a.world.planet2Pos = a.nc.Planet2Pos()
	a.world.planet2Vel = a.nc.Planet2Vel()
}

// --- Debug keys + chat ---

func (a *App) handleDebugKeys() {
	if rl.IsKeyPressed(rl.KeyF3) {
		a.diag.showDebug = !a.diag.showDebug
	}
	if rl.IsKeyPressed(rl.KeyF8) {
		a.diag.diagFrames = 120 // 2 секунды @ 60 FPS
		a.log.Info("DIAG START")
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

// --- Пауза: физика + привязка к Земле ---

// updatePausePhysics на паузе жёстко привязывает игрока к Земле по rel-координатам,
// применяет гравитацию (TickPhysicsOnly) и синкает pausedRelPos/Vel для release.
func (a *App) updatePausePhysics() {
	if a.mode != state.ModePaused || a.flight == nil || !a.flight.HelioInit {
		return
	}

	a.flight.EarthPos = rl.NewVector3(a.world.earthPos.X, a.world.earthPos.Y, a.world.earthPos.Z)
	a.flight.EarthVel = rl.NewVector3(a.world.earthVel.X, a.world.earthVel.Y, a.world.earthVel.Z)

	// Гравитация + коллизия — персонаж падает на грунт, не висит в воздухе.
	a.flight.TickPhysicsOnly(rl.GetFrameTime())

	// pausedRel* используются при выходе из паузы (updatePaused).
	a.pause.pausedRelPos = protocol.Vector3{
		X: a.flight.RelPos.X,
		Y: a.flight.RelPos.Y,
		Z: a.flight.RelPos.Z,
	}
	a.pause.pausedRelVel = protocol.Vector3{
		X: a.flight.RelVel.X,
		Y: a.flight.RelVel.Y,
		Z: a.flight.RelVel.Z,
	}

	// Камера едет с Землёй.
	fw := a.flight.Forward()
	a.camera.camera.Position = a.flight.Pos
	a.camera.camera.Target = rl.Vector3Add(a.flight.Pos, rl.Vector3Scale(fw, 100.0))
	a.camera.camera.Up = a.flight.CameraUp()
}

// --- Dispatch по режиму ---

func (a *App) dispatchMode(dt float32) {
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
}

// --- Диагностика (F8) ---

func (a *App) updateDiag(dt float32) {
	if a.diag.diagFrames <= 0 || a.flight == nil {
		return
	}
	a.diag.diagFrames--

	rel := rl.Vector3Subtract(a.flight.Pos, a.earthPosAsRl())
	a.log.Info("DIAG",
		"flightPos", fmt.Sprintf("%.4f,%.4f,%.4f", a.flight.Pos.X, a.flight.Pos.Y, a.flight.Pos.Z),
		"camPos", fmt.Sprintf("%.4f,%.4f,%.4f", a.camera.camSmoothPos.X, a.camera.camSmoothPos.Y, a.camera.camSmoothPos.Z),
		"earthPos", fmt.Sprintf("%.4f,%.4f,%.4f", a.world.earthPos.X, a.world.earthPos.Y, a.world.earthPos.Z),
		"earthVel", fmt.Sprintf("%.5f,%.5f,%.5f", a.world.earthVel.X, a.world.earthVel.Y, a.world.earthVel.Z),
		"rel", fmt.Sprintf("%.4f,%.4f,%.4f", rel.X, rel.Y, rel.Z),
		"relLen", fmt.Sprintf("%.4f", rl.Vector3Length(rel)),
		"vel", fmt.Sprintf("%.4f,%.4f,%.4f", a.flight.Vel.X, a.flight.Vel.Y, a.flight.Vel.Z),
		"renderTick", fmt.Sprintf("%.4f", a.camera.renderTick),
		"dt", fmt.Sprintf("%.5f", dt))
	if a.diag.diagFrames == 0 {
		a.log.Info("DIAG END")
	}
}

// --- Меню ---

func (a *App) updateMenu() {
	a.setCursorCaptured(false)

	ui.EditField(&a.ui.menuNick, a.ui.menuFocus == 0, 32)
	ui.EditField(&a.ui.menuAddr, a.ui.menuFocus == 1, 64)

	if rl.IsKeyPressed(rl.KeyTab) {
		a.ui.menuFocus = (a.ui.menuFocus + 1) % 2
	}
	if rl.IsKeyPressed(rl.KeyEnter) {
		a.startConnect()
	}
}

// --- Пауза (обработка клавиш) ---

func (a *App) updatePaused() {
	a.setCursorCaptured(false)

	if rl.IsKeyPressed(rl.KeyF) {
		a.tryPickup()
	}
	if rl.IsKeyPressed(rl.KeyEscape) {
		a.exitPause()
	}
}

// exitPause восстанавливает velocity из pausedRelVel и возвращает в игру.
func (a *App) exitPause() {
	if a.flight != nil && a.flight.HelioInit {
		a.flight.Vel = rl.NewVector3(
			a.world.earthVel.X+a.pause.pausedRelVel.X,
			a.world.earthVel.Y+a.pause.pausedRelVel.Y,
			a.world.earthVel.Z+a.pause.pausedRelVel.Z,
		)
		a.log.Info("PAUSE EXITED",
			"earth", fmt.Sprintf("%.2f,%.2f,%.2f", a.world.earthPos.X, a.world.earthPos.Y, a.world.earthPos.Z),
			"flightPos", fmt.Sprintf("%.2f,%.2f,%.2f", a.flight.Pos.X, a.flight.Pos.Y, a.flight.Pos.Z),
			"expectedPos", fmt.Sprintf("%.2f,%.2f,%.2f", a.world.earthPos.X+a.pause.pausedRelPos.X, a.world.earthPos.Y+a.pause.pausedRelPos.Y, a.world.earthPos.Z+a.pause.pausedRelPos.Z),
			"flightVel", fmt.Sprintf("%.3f,%.3f,%.3f", a.flight.Vel.X, a.flight.Vel.Y, a.flight.Vel.Z))
	}
	a.mode = state.ModePlaying
}
