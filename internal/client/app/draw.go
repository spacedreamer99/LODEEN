package app

import (
	"time"

	rl "github.com/gen2brain/raylib-go/raylib"

	"github.com/spacedreamer99/lodeen/internal/client/render"
	"github.com/spacedreamer99/lodeen/internal/client/state"
)

func (a *App) draw() {

	// Камера уже в helio (flight.Pos в helio) — без сдвига.
	epV := rl.NewVector3(a.earthPos.X, a.earthPos.Y, a.earthPos.Z)
	camRender := a.camera

	// Проверяем, изменился ли размер окна — пересоздаём UI-буфер.
	sw := int32(rl.GetScreenWidth())
	sh := int32(rl.GetScreenHeight())
	if sw != a.uiTargetW || sh != a.uiTargetH {
		a.resizeUITarget()
		a.startAt = time.Now()
	}

	rl.BeginDrawing()
	defer rl.EndDrawing()

	// 3D-слой — напрямую в окно
	switch a.mode {
	case state.ModeMenu:
		rl.ClearBackground(rl.NewColor(12, 12, 22, 255))
	default:
		rl.ClearBackground(rl.NewColor(4, 4, 12, 255))

		// Звёзды — 2D-точки на экране.
		render.DrawSkybox(camRender, sw, sh)

		rl.BeginMode3D(camRender)

		// ── Geo-объекты: сдвигаем всю пачку на +earthPos ──
		rl.PushMatrix()
		rl.Translatef(epV.X, epV.Y, epV.Z)

		a.scene.Draw()
		a.scene.DrawWater()
		a.scene.DrawClouds(camRender)
		a.scene.DrawAtmosphere(camRender)
		render.DrawPlayers(a.nc.InterpolatedSnapshot(), a.nc.PlayerID(), a.camera)
		render.DrawResources(a.nc.Resources())
		render.DrawMammoths(a.nc.Mammoths())
		render.DrawWells(a.nc.Wells())
		render.DrawHouses(a.nc.Houses())
		render.DrawSolar(a.nc.Solar())
		render.DrawBatteries(a.nc.Batteries())
		render.DrawFactories(a.nc.Factories())
		render.DrawBoats(a.nc.Boats())
		render.DrawMobs(a.nc.Mobs())
		projs := a.nc.Projectiles()
		if len(projs) > 0 {
			a.log.Debug("proj debug", "count", len(projs),
				"first", projs[0].X, projs[0].Y, projs[0].Z)
		}
		render.DrawProjectiles(projs)
		me := a.nc.PlayerID()
		if me != "" {
			for _, m := range a.nc.Mammoths() {
				if m.LeashedTo == me {
					camGeo := rl.Vector3Subtract(a.camera.Position, epV)
					render.DrawLeash(camGeo,
						rl.NewVector3(m.X, m.Y, m.Z))
				}
			}
		}
		a.drawProjectiles()
		a.drawHeldItem()

		rl.PopMatrix()
		// ── Geo-объекты закончились ──

		// Ракеты — helio-объекты.
		render.DrawRockets(a.nc.Rockets())

		// Солнце — helio-объект, в реальной позиции.
		a.scene.DrawSun3D(camRender)
		a.scene.DrawStar2(camRender)
		a.scene.DrawPlanet2(camRender, a.planet2Pos)
		rl.EndMode3D()
	}

	// UI — в буфер размером с окно
	rl.BeginTextureMode(a.uiTarget)
	rl.ClearBackground(rl.Blank)

	switch a.mode {
	case state.ModeMenu:
		a.drawMenu()
	case state.ModeDead:
		a.drawHUD()
		a.drawDead()
	default:
		if a.mode == state.ModePaused {
			// Затемнение
			rl.DrawRectangle(0, 0, sw, sh, rl.Fade(rl.Black, 0.55))
			a.drawPause()
		}
		a.drawHUD()
		if a.showContract {
			a.drawContract()
		}
		if a.showFactory {
			a.drawFactory()
		}
		if a.rocketID != "" {
			a.drawRocketHUD()
			a.drawNavBall()
		}
		if a.showOrbitMap {
			a.drawOrbitMap()
		}
	}

	rl.EndTextureMode()

	// Показываем буфер 1:1 — без масштабирования, поэтому резко.
	src := rl.NewRectangle(0, 0, float32(a.uiTarget.Texture.Width), -float32(a.uiTarget.Texture.Height))
	dst := rl.NewRectangle(0, 0, float32(sw), float32(sh))
	rl.DrawTexturePro(a.uiTarget.Texture, src, dst, rl.NewVector2(0, 0), 0, rl.White)
}

func (a *App) resizeUITarget() {
	if a.uiTarget.ID != 0 {
	}
	w := int32(rl.GetScreenWidth())
	h := int32(rl.GetScreenHeight())
	if w <= 0 || h <= 0 {
		w, h = screenW, screenH
	}
	a.uiTarget = rl.LoadRenderTexture(w, h)
	a.uiTargetW = w
	a.uiTargetH = h
}
