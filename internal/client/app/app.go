package app

import (
	"fmt"
	"log/slog"
	"math"
	"os"
	"time"

	rl "github.com/gen2brain/raylib-go/raylib"

	"github.com/spacedreamer99/lodeen/internal/client/chat"
	"github.com/spacedreamer99/lodeen/internal/client/fonts"
	"github.com/spacedreamer99/lodeen/internal/client/input"
	clientnet "github.com/spacedreamer99/lodeen/internal/client/net"
	"github.com/spacedreamer99/lodeen/internal/client/render"
	"github.com/spacedreamer99/lodeen/internal/client/state"
	"github.com/spacedreamer99/lodeen/internal/client/ui"
	"github.com/spacedreamer99/lodeen/internal/shared/config"
	"github.com/spacedreamer99/lodeen/internal/shared/protocol"
)

const (
	screenW = 1280
	screenH = 720
)

type projectile struct {
	pos   rl.Vector3
	dir   rl.Vector3
	spawn time.Time
}

type App struct {
	cfg *config.Config
	log *slog.Logger

	mode state.Mode

	nc     *clientnet.Client
	flight *input.FlightController
	scene  *render.Scene
	chat   *chat.Chat

	camera rl.Camera3D

	menuNick  string
	menuAddr  string
	menuErr   string
	menuFocus int

	uiTarget  rl.RenderTexture2D
	uiTargetW int32
	uiTargetH int32

	debugFrame int
	cachedFPS  int32
	lastFPSAt  time.Time

	cursorCaptured bool
	quit           bool
	showInventory  bool
	showCraft      bool
	selectedSlot   int
	projectiles    []projectile
	startAt        time.Time

	lastKeys     string
	lastSentHeld string
	ridingID     string
	hp           int
	hpReceived   bool
	boatID       string
	lastMouse    string
	lastLogAt    time.Time
	lastPos      rl.Vector3
	contractMobID string
	contractPos   rl.Vector3
	showContract  bool
	contractOpenedAt time.Time
	factoryID     string
	showFactory   bool
	factoryOpenedAt time.Time
	invSlots     [256]string
	dragging     bool
	dragFrom     int
	rocketID      string
	rocketBoardedAt time.Time
	hudRocket     protocol.Rocket
	showOrbitMap  bool
	orbitAzimuth   float32
	orbitElevation float32
	orbitDistance  float32
	orbitInit      bool
	autoPilot     string
	rocketNoseX    float32
	rocketNoseY    float32
	rocketNoseZ    float32
	rocketNoseInit bool
}

func New(cfg *config.Config, log *slog.Logger) *App {
	return &App{
		hp:       100,
		cfg:      cfg,
		log:      log,
		mode:     state.ModeMenu,
		menuNick: cfg.Client.Nick,
		menuAddr: cfg.Client.StartAddr,
	}
}

func (a *App) Run() error {
	rl.SetConfigFlags(rl.FlagVsyncHint | rl.FlagWindowResizable)
	rl.InitWindow(screenW, screenH, "LODEEN")
	defer rl.CloseWindow()
	rl.SetTargetFPS(60)
	rl.SetExitKey(rl.KeyNull)

	a.resizeUITarget()
	a.startAt = time.Now()
	fonts.Load(28)
	if os.Getenv("LODEEN_FONT_DEBUG") == "1" {
		fonts.DebugDump()
	}

	a.scene = render.NewScene(a.cfg.Client.PlanetModel)
	defer a.scene.Unload()
	if a.scene.HasPlanet() {
		a.log.Info("planet model loaded", "path", a.cfg.Client.PlanetModel)
	} else {
		a.log.Warn("planet model not loaded, using fallback sphere",
			"path", a.cfg.Client.PlanetModel)
	}

	a.nc = clientnet.New(a.log)
	a.chat = chat.New()

	a.camera = rl.Camera3D{
		Position:   rl.NewVector3(0, 5, 40),
		Target:     rl.NewVector3(0, 0, 0),
		Up:         rl.NewVector3(0, 1, 0),
		Fovy:       70,
		Projection: rl.CameraPerspective,
	}

	for !rl.WindowShouldClose() && !a.quit {
		dt := rl.GetFrameTime()
		a.update(dt)
		a.draw()
	}
	a.setCursorCaptured(false)
	a.nc.Disconnect()
	return nil
}

func (a *App) setCursorCaptured(c bool) {
	if a.cursorCaptured == c {
		return
	}
	a.cursorCaptured = c
	if c {
		rl.DisableCursor()
		midX := rl.GetScreenWidth() / 2
		midY := rl.GetScreenHeight() / 2
		rl.SetMousePosition(midX, midY)
	} else {
		rl.EnableCursor()
		rl.ShowCursor()
	}
}

func (a *App) update(dt float32) {
	a.drainChat()

	a.updateProjectiles()

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
		a.mode = state.ModePaused
		return
	}

	if !rl.IsWindowFocused() {
		a.setCursorCaptured(false)
		return
	}

	a.setCursorCaptured(true)

	if a.flight == nil {
		return
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
		edgeMouse := ""
		if mouse != a.lastMouse {
			edgeMouse = mouse
		}

		fw := a.flight.Forward()
		rt := a.flight.Right()
		up := a.flight.Up()
		q := a.flight.Quat

		dtSec := now.Sub(a.lastLogAt).Seconds()
		vel := "0.0,0.0,0.0"
		if dtSec > 0 {
			vx := (a.flight.Pos.X - a.lastPos.X) / float32(dtSec)
			vy := (a.flight.Pos.Y - a.lastPos.Y) / float32(dtSec)
			vz := (a.flight.Pos.Z - a.lastPos.Z) / float32(dtSec)
			vel = fmt.Sprintf("%.1f,%.1f,%.1f", vx, vy, vz)
		}

		a.log.Info("6dof",
			"dmdx", fmt.Sprintf("%+.1f", md.X),
			"dmdy", fmt.Sprintf("%+.1f", md.Y),
			"edgeK", edgeKeys,
			"edgeM", edgeMouse,
			"keys", keys,
			"mouse", mouse,
			"pos", fmt.Sprintf("%.2f,%.2f,%.2f", a.flight.Pos.X, a.flight.Pos.Y, a.flight.Pos.Z),
			"vel", vel,
			"fwd", fmt.Sprintf("%.2f,%.2f,%.2f", fw.X, fw.Y, fw.Z),
			"rgt", fmt.Sprintf("%.2f,%.2f,%.2f", rt.X, rt.Y, rt.Z),
			"up", fmt.Sprintf("%.2f,%.2f,%.2f", up.X, up.Y, up.Z),
			"quat", fmt.Sprintf("%.3f,%.3f,%.3f,%.3f", q.X, q.Y, q.Z, q.W),
			"cam.pos", fmt.Sprintf("%.2f,%.2f,%.2f", a.camera.Position.X, a.camera.Position.Y, a.camera.Position.Z),
			"cam.tgt", fmt.Sprintf("%.2f,%.2f,%.2f", a.camera.Target.X, a.camera.Target.Y, a.camera.Target.Z),
			"cam.up", fmt.Sprintf("%.2f,%.2f,%.2f", a.camera.Up.X, a.camera.Up.Y, a.camera.Up.Z),
		)

		a.lastLogAt = now
		a.lastPos = a.flight.Pos
		a.lastKeys = keys
		a.lastMouse = mouse
	}
	fw := a.flight.Forward()
	a.camera.Position = a.flight.Pos
	a.camera.Target = rl.Vector3Add(a.flight.Pos, rl.Vector3Scale(fw, 100.0))
	a.camera.Up = a.flight.CameraUp()

	yawF := float32(math.Atan2(float64(-fw.X), float64(-fw.Z)))
	pitchF := float32(math.Asin(float64(fw.Y)))
	a.nc.SetState(protocol.PlayerState{
		X: a.flight.Pos.X, Y: a.flight.Pos.Y, Z: a.flight.Pos.Z,
		Yaw: yawF, Pitch: pitchF,
	})
}

func (a *App) updatePaused() {
	a.setCursorCaptured(false)

	if rl.IsKeyPressed(rl.KeyF) {
		a.tryPickup()
	}
	if rl.IsKeyPressed(rl.KeyEscape) {
		a.mode = state.ModePlaying
	}
}

func (a *App) draw() {
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
		rl.BeginMode3D(a.camera)
		a.scene.Draw()
		a.scene.DrawWater()
		render.DrawPlayers(a.nc.InterpolatedSnapshot(), a.nc.PlayerID(), a.camera)
		render.DrawResources(a.nc.Resources())
		render.DrawMammoths(a.nc.Mammoths())
		render.DrawWells(a.nc.Wells())
		render.DrawHouses(a.nc.Houses())
		render.DrawSolar(a.nc.Solar())
		render.DrawBatteries(a.nc.Batteries())
		render.DrawFactories(a.nc.Factories())
		render.DrawRockets(a.nc.Rockets())
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
					render.DrawLeash(a.camera.Position,
						rl.NewVector3(m.X, m.Y, m.Z))
				}
			}
		}
		a.drawProjectiles()
		a.drawHeldItem()
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
			if a.showOrbitMap {
				a.drawOrbitMap()
			}
			a.drawNavBall()
		}
	}

	rl.EndTextureMode()

	// Показываем буфер 1:1 — без масштабирования, поэтому резко.
	src := rl.NewRectangle(0, 0, float32(a.uiTarget.Texture.Width), -float32(a.uiTarget.Texture.Height))
	dst := rl.NewRectangle(0, 0, float32(sw), float32(sh))
	rl.DrawTexturePro(a.uiTarget.Texture, src, dst, rl.NewVector2(0, 0), 0, rl.White)
}

// resizeUITarget пересоздаёт UI-буфер под текущий размер окна.
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

// updateDead обрабатывает горячие клавиши на экране смерти.
// factoryRecipeUI — рецепты завода для UI.
type factoryRecipeUI struct {
	id     string
	name   string
	need   map[string]int
	energy int
}

var factoryRecipeList = []factoryRecipeUI{
	{id: "steel", name: "Steel", need: map[string]int{"ore": 5}, energy: 10},
	{id: "gear", name: "Gear", need: map[string]int{"stone": 2, "wood": 2}, energy: 20},
	{id: "circuit", name: "Circuit", need: map[string]int{"ore": 3, "liana": 1}, energy: 50},
	{id: "drone", name: "Drone", need: map[string]int{"gear": 1, "circuit": 1}, energy: 100},
	{id: "rocket", name: "Rocket", need: map[string]int{"circuit": 3, "steel": 5, "gear": 2}, energy: 500},
}

func (a *App) updateFactory() {
	if time.Since(a.factoryOpenedAt) < 250*time.Millisecond {
		return
	}
	if rl.IsKeyPressed(rl.KeyEscape) {
		a.showFactory = false
		a.factoryID = ""
		rl.DisableCursor()
	}
}

func (a *App) drawFactory() {
	ignoreClicks := time.Since(a.factoryOpenedAt) < 250*time.Millisecond

	sw := int32(rl.GetScreenWidth())
	sh := int32(rl.GetScreenHeight())

	rl.DrawRectangle(0, 0, sw, sh, rl.Fade(rl.Black, 0.6))

	pw := int32(560)
	ph := int32(int32(len(factoryRecipeList))*80 + 100)
	px := (sw - pw) / 2
	py := (sh - ph) / 2

	panel := rl.NewRectangle(float32(px), float32(py), float32(pw), float32(ph))
	rl.DrawRectangleRec(panel, rl.NewColor(40, 45, 60, 245))
	rl.DrawRectangleLinesEx(panel, 2, rl.NewColor(120, 180, 240, 255))

	title := "FACTORY"
	tw := fonts.Measure(title, 36)
	fonts.Draw(title, px+(pw-tw)/2, py+18, 36, rl.NewColor(150, 200, 255, 255))

	const bw = 520
	const bh = 60
	bx := px + (pw - bw) / 2

	for i, r := range factoryRecipeList {
		y := py + 80 + int32(i)*80

		// Текст с требованиями.
		req := r.name + "  ["
		first := true
		for item, q := range r.need {
			if !first {
				req += ", "
			}
			req += item + " x" + itoa(q)
			first = false
		}
		req += "]  E:" + itoa(r.energy)

		btn := ui.Button{
			Rect: rl.NewRectangle(float32(bx), float32(y), bw, bh),
			Text: req,
		}
		btn.Draw()
		if !ignoreClicks && btn.Clicked() {
			if err := a.nc.CraftFactory(a.factoryID, r.id); err != nil {
				a.log.Warn("factory craft", "err", err)
			}
			a.log.Info("factory craft sent", "recipe", r.id)
		}
	}

	esc := "Esc - close"
	escW := fonts.Measure(esc, 16)
	fonts.Draw(esc, px+(pw-escW)/2, py+ph-28, 16, rl.Gray)
}

// itoa — простой int→string (без зависимостей).
func itoa(n int) string {
	if n == 0 {
		return "0"
	}
	neg := n < 0
	if neg {
		n = -n
	}
	var buf [20]byte
	i := len(buf)
	for n > 0 {
		i--
		buf[i] = byte('0' + n%10)
		n /= 10
	}
	if neg {
		i--
		buf[i] = '-'
	}
	return string(buf[i:])
}

// updatePilotedRocket — если игрок в ракете, обновляет её состояние на клиенте.
// Возвращает true если мы по-прежнему в ракете.
func (a *App) updatePilotedRocket(dt float32) bool {
	rs := a.nc.Rockets()

	var found *protocol.Rocket
	for i := range rs {
		if rs[i].ID == a.rocketID {
			found = &rs[i]
			break
		}
	}

	// Grace period: первые 3 секунды после board снапшот может ещё не прийти.
	// Не сбрасываем rocketID, просто ждём.
	graceful := time.Since(a.rocketBoardedAt) < 3*time.Second

	if found == nil {
		if graceful {
			// ещё ждём подтверждения от сервера
			return true
		}
		a.log.Info("rocket lost, leaving", "id", a.rocketID)
		return false
	}

	// Синхронизация позиции игрока с ракетой.
	a.flight.Pos = rl.NewVector3(
		found.X+found.DX*1.5,
		found.Y+found.DY*1.5,
		found.Z+found.DZ*1.5,
	)
	a.flight.Vel = rl.NewVector3(0, 0, 0)

	// Сохранить данные для HUD.
	a.hudRocket = *found

	// Инициализация носа ракеты при первом кадре.
	// Считаем нормаль от центра планеты — не доверяем found.DX/DY/DZ.
	if !a.rocketNoseInit {
		rLen := float32(math.Sqrt(float64(found.X*found.X + found.Y*found.Y + found.Z*found.Z)))
		if rLen > 0.01 {
			a.rocketNoseX = found.X / rLen
			a.rocketNoseY = found.Y / rLen
			a.rocketNoseZ = found.Z / rLen
		} else {
			a.rocketNoseX = 0
			a.rocketNoseY = 1
			a.rocketNoseZ = 0
		}
		a.rocketNoseInit = true
		a.log.Info("rocket nose initialized",
			"x", a.rocketNoseX,
			"y", a.rocketNoseY,
			"z", a.rocketNoseZ)
	}

	// Тяга по Space.
	var thrust float32
	if rl.IsKeyDown(rl.KeySpace) {
		thrust = 1.0
	}
	if rl.IsKeyDown(rl.KeyLeftControl) {
		thrust = -1.0
	}

	// WASD — вращение носа ракеты в локальной системе камеры.
	{
		upLocal := a.camera.Up
		fwLocal := a.flight.Forward()
		rightLocal := rl.Vector3Normalize(rl.Vector3CrossProduct(fwLocal, upLocal))
		upTrue := rl.Vector3Normalize(rl.Vector3CrossProduct(rightLocal, fwLocal))

		const noseRate = 1.8
		angleStep := noseRate * dt

		nose := rl.NewVector3(a.rocketNoseX, a.rocketNoseY, a.rocketNoseZ)
		rotated := false
		if rl.IsKeyDown(rl.KeyW) {
			nose = rotateAroundAxis(nose, rightLocal, -angleStep)
			rotated = true
		}
		if rl.IsKeyDown(rl.KeyS) {
			nose = rotateAroundAxis(nose, rightLocal, angleStep)
			rotated = true
		}
		if rl.IsKeyDown(rl.KeyA) {
			nose = rotateAroundAxis(nose, upTrue, angleStep)
			rotated = true
		}
		if rl.IsKeyDown(rl.KeyD) {
			nose = rotateAroundAxis(nose, upTrue, -angleStep)
			rotated = true
		}
		if rotated {
			nose = rl.Vector3Normalize(nose)
			a.rocketNoseX = nose.X
			a.rocketNoseY = nose.Y
			a.rocketNoseZ = nose.Z
			a.log.Info("rocket nose rotated",
				"x", nose.X, "y", nose.Y, "z", nose.Z)
		}
	}

	// Отправляем желаемое направление носа ракеты.
	_ = a.nc.RocketInput(thrust, a.rocketNoseX, a.rocketNoseY, a.rocketNoseZ, a.autoPilot)

	// Автопилот: G prograde, H retrograde, J radial-out, K radial-in,
	// N normal, B antinormal, X off.
	if rl.IsKeyPressed(rl.KeyG) {
		a.autoPilot = "prograde"
	}
	if rl.IsKeyPressed(rl.KeyH) {
		a.autoPilot = "retrograde"
	}
	if rl.IsKeyPressed(rl.KeyJ) {
		a.autoPilot = "radial_out"
	}
	if rl.IsKeyPressed(rl.KeyK) {
		a.autoPilot = "radial_in"
	}
	if rl.IsKeyPressed(rl.KeyN) {
		a.autoPilot = "normal"
	}
	if rl.IsKeyPressed(rl.KeyB) {
		a.autoPilot = "antinormal"
	}
	if rl.IsKeyPressed(rl.KeyX) {
		a.autoPilot = ""
	}

	// M — карта орбиты.
	if rl.IsKeyPressed(rl.KeyM) {
		a.showOrbitMap = !a.showOrbitMap
		if a.showOrbitMap {
			a.orbitInit = false
			rl.EnableCursor()
			rl.ShowCursor()
		} else {
			rl.DisableCursor()
		}
	}

	// E — выйти.
	if rl.IsKeyPressed(rl.KeyE) {
		_ = a.nc.ExitRocket()
		a.rocketID = ""
		a.showOrbitMap = false
		a.rocketNoseInit = false
		a.log.Info("exit rocket sent")
		return false
	}

	// Если карта открыта — ракета НЕ крутится мышью.
	if a.showOrbitMap {
		a.updateOrbitMapInput()
	} else {
		mdRocket := rl.GetMouseDelta()
		a.flight.UpdateLookOnly(dt, mdRocket)
	}

	// Камера следует за ракетой.
	fw := a.flight.Forward()
	a.camera.Position = a.flight.Pos
	a.camera.Target = rl.Vector3Add(a.flight.Pos, rl.Vector3Scale(fw, 100.0))
	a.camera.Up = a.flight.CameraUp()

	a.log.Info("rocket pilot tick",
		"id", a.rocketID,
		"x", found.X, "y", found.Y, "z", found.Z,
		"fuel", found.Fuel,
		"thrust", thrust)
	return true
}

func (a *App) updateContract() {
	if time.Since(a.contractOpenedAt) < 250*time.Millisecond {
		return
	}
	if rl.IsKeyPressed(rl.KeyEscape) {
		a.showContract = false
		a.contractMobID = ""
		rl.DisableCursor()
		return
	}
}

// drawRocketHUD — оверлей с орбитальной информацией, пока сидим в ракете.
// predictTrajectory — быстрая симуляция орбиты на N шагов вперёд.
// Копирует серверную физику (без атмосферы для простоты).
func predictTrajectory(pos, vel protocol.Vector3, dt float32, steps int) []protocol.Vector3 {
	const R = 50.0
	const G = 40.0
	mu := float32(G * R * R)

	p, v := pos, vel
	pts := make([]protocol.Vector3, 0, steps)
	for i := 0; i < steps; i++ {
		r2 := p.X*p.X + p.Y*p.Y + p.Z*p.Z
		r := float32(math.Sqrt(float64(r2)))
		if r < 1 {
			break
		}
		g := mu / r2
		ax := -p.X / r * g
		ay := -p.Y / r * g
		az := -p.Z / r * g
		v.X += ax * dt
		v.Y += ay * dt
		v.Z += az * dt
		p.X += v.X * dt
		p.Y += v.Y * dt
		p.Z += v.Z * dt
		r2 = p.X*p.X + p.Y*p.Y + p.Z*p.Z
		r = float32(math.Sqrt(float64(r2)))
		if r < R {
			break
		}
		pts = append(pts, p)
	}
	return pts
}

// drawOrbitMap — 2D проекция орбиты сверху.
// updateOrbitMapInput — управление орбитальной камерой (drag + zoom).
func (a *App) updateOrbitMapInput() {
	md := rl.GetMouseDelta()
	if rl.IsMouseButtonDown(rl.MouseLeftButton) {
		a.orbitAzimuth += md.X * 0.008
		a.orbitElevation += md.Y * 0.008
		if a.orbitElevation < -1.4 {
			a.orbitElevation = -1.4
		}
		if a.orbitElevation > 1.4 {
			a.orbitElevation = 1.4
		}
	}
	wheel := rl.GetMouseWheelMove()
	if wheel != 0 {
		a.orbitDistance *= 1.0 - wheel*0.15
		if a.orbitDistance < 80 {
			a.orbitDistance = 80
		}
		if a.orbitDistance > 5000 {
			a.orbitDistance = 5000
		}
	}
}

// drawOrbitMap — 3D вид на орбиту вокруг планеты.
// rotateAroundAxis — поворот вектора вокруг оси (формула Родрига).
func rotateAroundAxis(v, axis rl.Vector3, angle float32) rl.Vector3 {
	cosA := float32(math.Cos(float64(angle)))
	sinA := float32(math.Sin(float64(angle)))
	dot := v.X*axis.X + v.Y*axis.Y + v.Z*axis.Z
	cx := axis.Y*v.Z - axis.Z*v.Y
	cy := axis.Z*v.X - axis.X*v.Z
	cz := axis.X*v.Y - axis.Y*v.X
	return rl.NewVector3(
		v.X*cosA+cx*sinA+axis.X*dot*(1-cosA),
		v.Y*cosA+cy*sinA+axis.Y*dot*(1-cosA),
		v.Z*cosA+cz*sinA+axis.Z*dot*(1-cosA),
	)
}

func (a *App) drawOrbitMap() {
	r := a.hudRocket
	if r.ID == "" {
		return
	}

	// Инициализация камеры при первом открытии.
	if !a.orbitInit {
		a.orbitInit = true
		a.orbitAzimuth = 0.6
		a.orbitElevation = 0.5
		dist := r.Altitude * 2.5
		if dist < 200 {
			dist = 200
		}
		if dist > 1500 {
			dist = 1500
		}
		a.orbitDistance = dist
	}

	// Камера вокруг центра планеты.
	cosEl := float32(math.Cos(float64(a.orbitElevation)))
	camPos := rl.NewVector3(
		a.orbitDistance*cosEl*float32(math.Cos(float64(a.orbitAzimuth))),
		a.orbitDistance*float32(math.Sin(float64(a.orbitElevation))),
		a.orbitDistance*cosEl*float32(math.Sin(float64(a.orbitAzimuth))),
	)

	mapCam := rl.Camera3D{
		Position:   camPos,
		Target:     rl.NewVector3(0, 0, 0),
		Up:         rl.NewVector3(0, 1, 0),
		Fovy:       60,
		Projection: rl.CameraPerspective,
	}

	// Фон.
	sw := int32(rl.GetScreenWidth())
	sh := int32(rl.GetScreenHeight())
	rl.DrawRectangle(0, 0, sw, sh, rl.NewColor(3, 5, 15, 255))

	rl.BeginMode3D(mapCam)

	// Планета.
	rl.DrawSphere(rl.NewVector3(0, 0, 0), protocol.PlanetRadius, rl.NewColor(50, 80, 130, 255))
	rl.DrawSphereWires(rl.NewVector3(0, 0, 0), protocol.PlanetRadius, 16, 16, rl.NewColor(80, 120, 180, 200))
	// Атмосфера.
	rl.DrawSphereWires(rl.NewVector3(0, 0, 0), protocol.PlanetRadius+50, 16, 16, rl.NewColor(80, 100, 140, 100))

	// Оси (для ориентации).
	rl.DrawLine3D(rl.NewVector3(0, 0, 0), rl.NewVector3(protocol.PlanetRadius*2, 0, 0), rl.NewColor(120, 50, 50, 180))
	rl.DrawLine3D(rl.NewVector3(0, 0, 0), rl.NewVector3(0, protocol.PlanetRadius*2, 0), rl.NewColor(50, 120, 50, 180))
	rl.DrawLine3D(rl.NewVector3(0, 0, 0), rl.NewVector3(0, 0, protocol.PlanetRadius*2), rl.NewColor(50, 50, 120, 180))

	// Траектория.
	pos := protocol.Vector3{X: r.X, Y: r.Y, Z: r.Z}
	vel := protocol.Vector3{X: r.VX, Y: r.VY, Z: r.VZ}
	traj := predictTrajectory(pos, vel, 0.1, 800)
	yellow := rl.NewColor(230, 200, 60, 255)
	for i := 1; i < len(traj); i++ {
		p1 := rl.NewVector3(traj[i-1].X, traj[i-1].Y, traj[i-1].Z)
		p2 := rl.NewVector3(traj[i].X, traj[i].Y, traj[i].Z)
		rl.DrawLine3D(p1, p2, yellow)
	}

	// Ракета.
	rp := rl.NewVector3(r.X, r.Y, r.Z)
	rl.DrawSphere(rp, 4, rl.NewColor(100, 255, 100, 255))
	rl.DrawSphereWires(rp, 4, 8, 8, rl.White)

	// Вектор скорости.
	vLen := float32(math.Sqrt(float64(r.VX*r.VX + r.VY*r.VY + r.VZ*r.VZ)))
	if vLen > 0.1 {
		vEnd := rl.NewVector3(
			r.X+r.VX/vLen*30,
			r.Y+r.VY/vLen*30,
			r.Z+r.VZ/vLen*30,
		)
		rl.DrawLine3D(rp, vEnd, rl.NewColor(255, 80, 80, 255))
	}

	// Up ракеты.
	upEnd := rl.NewVector3(
		r.X+r.DX*15,
		r.Y+r.DY*15,
		r.Z+r.DZ*15,
	)
	rl.DrawLine3D(rp, upEnd, rl.NewColor(80, 200, 255, 255))

	rl.EndMode3D()

	// Оверлей поверх 3D.
	fonts.Draw("ORBITAL MAP", 30, 30, 28, rl.NewColor(150, 200, 255, 255))

	pad := int32(30)
	lineY := pad + 50
	lineH := int32(24)

	fonts.Draw("FUEL: "+itoa(r.Fuel)+" / "+itoa(r.MaxFuel), pad, lineY, 18, rl.RayWhite)
	lineY += lineH

	fonts.Draw("ALT: "+itoa(int(r.Altitude))+" m", pad, lineY, 18, rl.RayWhite)
	lineY += lineH

	fonts.Draw("VEL: "+itoa(int(r.Speed))+" m/s", pad, lineY, 18, rl.RayWhite)
	lineY += lineH

	apoColor := rl.RayWhite
	if r.Apoapsis > 0 {
		apoColor = rl.NewColor(150, 200, 255, 255)
	}
	fonts.Draw("APO: "+itoa(int(r.Apoapsis))+" m", pad, lineY, 18, apoColor)
	lineY += lineH

	periColor := rl.NewColor(220, 60, 60, 255)
	if r.Periapsis > 60 {
		periColor = rl.NewColor(80, 220, 100, 255)
	} else if r.Periapsis > 0 {
		periColor = rl.NewColor(230, 200, 60, 255)
	}
	fonts.Draw("PER: "+itoa(int(r.Periapsis))+" m", pad, lineY, 18, periColor)

	// Подсказка снизу: управление камерой.
	hint := "WASD: nose   |   LMB drag: camera   |   Wheel: zoom   |   M: close"
	hintW := fonts.Measure(hint, 18)
	fonts.Draw(hint, (sw-hintW)/2, sh-40, 18, rl.LightGray)

	// Подсказки автопилота — внизу справа.
	apX := int32(30)
	apY := sh - int32(220)
	fonts.Draw("AUTOPILOT", apX, apY, 16, rl.NewColor(150, 200, 255, 255))
	apY += 22

	rows := []struct {
		key   string
		label string
		col   rl.Color
	}{
		{"G", "prograde", rl.NewColor(80, 220, 100, 255)},
		{"H", "retrograde", rl.NewColor(230, 140, 40, 255)},
		{"J", "radial+", rl.NewColor(230, 200, 60, 255)},
		{"K", "radial-", rl.NewColor(180, 100, 220, 255)},
		{"N", "normal", rl.NewColor(100, 180, 255, 255)},
		{"B", "antinormal", rl.NewColor(100, 180, 255, 255)},
		{"X", "off", rl.LightGray},
	}
	for _, r := range rows {
		line := r.key + "  " + r.label
		fonts.Draw(line, apX+10, apY, 14, r.col)
		apY += 20
	}

	// Текущий автопилот.
	cur := "MANUAL"
	if a.autoPilot != "" {
		cur = a.autoPilot
	}
	fonts.Draw("current: "+cur, apX, apY+8, 16, rl.RayWhite)
}


// drawNavBall — простой 2D навбол: показывает направление носа ракеты
// и целевые векторы (prograde/retrograde/radial/normal).
func (a *App) drawNavBall() {
	r := a.hudRocket
	if r.ID == "" {
		return
	}
	if a.debugFrame%60 == 0 {
		a.log.Info("navball draw",
			"id", r.ID,
			"dx", r.DX, "dy", r.DY, "dz", r.DZ,
			"vx", r.VX, "vy", r.VY, "vz", r.VZ)
	}

	sw := int32(rl.GetScreenWidth())
	sh := int32(rl.GetScreenHeight())

	// Навбол — в правом нижнем углу.
	const radius = int32(90)
	cx := sw - radius - 30
	cy := sh - radius - 30

	// Фон.
	rl.DrawCircle(cx, cy, float32(radius), rl.NewColor(20, 30, 50, 220))
	rl.DrawCircleLines(cx, cy, float32(radius), rl.NewColor(120, 180, 240, 255))
	rl.DrawCircleLines(cx, cy, float32(radius)*0.66, rl.NewColor(80, 100, 140, 180))
	rl.DrawCircleLines(cx, cy, float32(radius)*0.33, rl.NewColor(80, 100, 140, 180))
	// Горизонт (горизонтальная линия).
	rl.DrawLine(cx-int32(float32(radius)*0.9), cy, cx+int32(float32(radius)*0.9), cy, rl.NewColor(120, 140, 180, 220))

	// Локальная система координат ракеты:
	// forward = Up ракеты (куда смотрит нос)
	// right = cross(forward, worldUp)
	// trueUp = cross(right, forward)
	fX, fY, fZ := r.DX, r.DY, r.DZ

	// worldUp — нормаль планеты из позиции ракеты.
	pLen := float32(math.Sqrt(float64(r.X*r.X + r.Y*r.Y + r.Z*r.Z)))
	if pLen < 0.01 {
		return
	}
	wX, wY, wZ := r.X/pLen, r.Y/pLen, r.Z/pLen

	// right = cross(forward, worldUp)
	rX := fY*wZ - fZ*wY
	rY := fZ*wX - fX*wZ
	rZ := fX*wY - fY*wX
	rLen := float32(math.Sqrt(float64(rX*rX + rY*rY + rZ*rZ)))
	if rLen < 0.01 {
		return
	}
	rX /= rLen
	rY /= rLen
	rZ /= rLen

	// trueUp = cross(right, forward)
	tuX := rY*fZ - rZ*fY
	tuY := rZ*fX - rX*fZ
	tuZ := rX*fY - rY*fX

	// Функция проекции вектора на 2D навбол.
	project := func(vx, vy, vz float32) (int32, int32, bool) {
		// Компоненты в локальном фрейме.
		fwd := vx*fX + vy*fY + vz*fZ
		rgt := vx*rX + vy*rY + vz*rZ
		upv := vx*tuX + vy*tuY + vz*tuZ
		// Если вектор "за" навболом — не рисуем.
		if fwd < -0.2 {
			return 0, 0, false
		}
		px := cx + int32(rgt*float32(radius))
		py := cy - int32(upv*float32(radius))
		return px, py, true
	}

	// Текущая позиция носа — точка в центре (центр навбола = forward).
	rl.DrawCircle(cx, cy, 3, rl.NewColor(80, 220, 255, 255))

	// Хелпер: нарисовать цель на навболе.
	drawTarget := func(vx, vy, vz float32, col rl.Color, label string) {
		vLen := float32(math.Sqrt(float64(vx*vx + vy*vy + vz*vz)))
		if vLen < 0.01 {
			return
		}
		px, py, ok := project(vx/vLen, vy/vLen, vz/vLen)
		if !ok {
			return
		}
		rl.DrawCircleLines(px, py, 6, col)
		rl.DrawCircleLines(px, py, 7, col)
		if label != "" {
			fonts.Draw(label, px+8, py-8, 12, col)
		}
	}

	// Prograde (по скорости).
	vLen := float32(math.Sqrt(float64(r.VX*r.VX + r.VY*r.VY + r.VZ*r.VZ)))
	if vLen > 0.5 {
		// Prograde — зелёный
		drawTarget(r.VX, r.VY, r.VZ, rl.NewColor(80, 220, 100, 255), "PRO")
		// Retrograde — оранжевый
		drawTarget(-r.VX, -r.VY, -r.VZ, rl.NewColor(230, 140, 40, 255), "RET")
	}

	// Radial out (от планеты) — жёлтый.
	drawTarget(wX, wY, wZ, rl.NewColor(230, 200, 60, 255), "RAD+")
	// Radial in — фиолетовый.
	drawTarget(-wX, -wY, -wZ, rl.NewColor(180, 100, 220, 255), "RAD-")

	// Normal (N) / Antinormal (B) — голубой.
	nx := r.Y*r.VZ - r.Z*r.VY
	ny := r.Z*r.VX - r.X*r.VZ
	nz := r.X*r.VY - r.Y*r.VX
	nl := float32(math.Sqrt(float64(nx*nx + ny*ny + nz*nz)))
	if nl > 0.1 {
		drawTarget(nx/nl, ny/nl, nz/nl, rl.NewColor(100, 180, 255, 255), "NRM")
		drawTarget(-nx/nl, -ny/nl, -nz/nl, rl.NewColor(100, 180, 255, 255), "ANM")
	}

	// Текущий автопилот — под навболом.
	label := "MANUAL"
	col := rl.LightGray
	switch a.autoPilot {
	case "prograde":
		label = "PROGRADE"
		col = rl.NewColor(80, 220, 100, 255)
	case "retrograde":
		label = "RETROGRADE"
		col = rl.NewColor(230, 140, 40, 255)
	case "radial_out":
		label = "RADIAL+"
		col = rl.NewColor(230, 200, 60, 255)
	case "radial_in":
		label = "RADIAL-"
		col = rl.NewColor(180, 100, 220, 255)
	case "normal":
		label = "NORMAL"
		col = rl.NewColor(100, 180, 255, 255)
	case "antinormal":
		label = "ANTINORMAL"
		col = rl.NewColor(100, 180, 255, 255)
	}
	lw := fonts.Measure(label, 16)
	fonts.Draw(label, cx-lw/2, cy+radius+8, 16, col)
}

func (a *App) drawRocketHUD() {
	if a.rocketID == "" {
		return
	}
	r := a.hudRocket
	if r.ID != a.rocketID {
		return
	}

	sw := int32(rl.GetScreenWidth())
	const panelW = int32(360)
	const panelH = int32(200)
	px := sw - panelW - 20
	py := int32(20)

	// Отдельно: проверка ESCAPE — когда apo/peri = 0 и alt выше орбиты.
	escaping := r.Apoapsis == 0 && r.Altitude > 60 && r.Speed > 20

	// Панель.
	bg := rl.NewColor(10, 15, 30, 220)
	if r.InOrbit {
		bg = rl.NewColor(15, 30, 15, 230)
	} else if escaping {
		bg = rl.NewColor(40, 15, 15, 230)
	}
	panel := rl.NewRectangle(float32(px), float32(py), float32(panelW), float32(panelH))
	rl.DrawRectangleRec(panel, bg)
	border := rl.NewColor(120, 180, 240, 255)
	if r.InOrbit {
		border = rl.NewColor(80, 220, 100, 255)
	} else if escaping {
		border = rl.NewColor(220, 60, 60, 255)
	}
	rl.DrawRectangleLinesEx(panel, 2, border)

	title := "ROCKET"
	if r.InOrbit {
		title = "ORBIT ACHIEVED"
	} else if escaping {
		title = "ESCAPE TRAJECTORY"
	}
	fonts.Draw(title, px+12, py+8, 20, border)

	// Строки.
	pad := int32(12)
	lineY := py + 40
	lineH := int32(20)

	fuelStr := "FUEL: " + itoa(r.Fuel) + " / " + itoa(r.MaxFuel)
	fonts.Draw(fuelStr, px+pad, lineY, 16, rl.RayWhite)
	lineY += lineH

	altStr := "ALT: " + itoa(int(r.Altitude)) + " m"
	fonts.Draw(altStr, px+pad, lineY, 16, rl.RayWhite)
	lineY += lineH

	spdStr := "VEL: " + itoa(int(r.Speed)) + " m/s"
	spdColor := rl.RayWhite
	if r.TargetVelocity > 0 {
		spdStr += "  (orbit: " + itoa(int(r.TargetVelocity)) + ")"
		// Цвет: зелёный если 85-115% от target, жёлтый если 50-85% или 115-150%, красный иначе.
		ratio := r.Speed / r.TargetVelocity
		switch {
		case ratio >= 0.85 && ratio <= 1.15:
			spdColor = rl.NewColor(80, 220, 100, 255) // точно
		case ratio >= 0.5 && ratio <= 1.5:
			spdColor = rl.NewColor(230, 200, 60, 255) // близко
		default:
			spdColor = rl.NewColor(220, 60, 60, 255) // не то
		}
	}
	fonts.Draw(spdStr, px+pad, lineY, 16, spdColor)
	lineY += lineH

	apoStr := "APO: " + itoa(int(r.Apoapsis)) + " m"
	periStr := "PER: " + itoa(int(r.Periapsis)) + " m"
	apoColor := rl.LightGray
	periColor := rl.LightGray
	if r.Apoapsis > 0 {
		apoColor = rl.RayWhite
	}
	if r.Periapsis > 60.0 { // atmosphereHeight (dup on client)
		periColor = rl.NewColor(80, 220, 100, 255)
	} else if r.Periapsis > 0 {
		periColor = rl.NewColor(230, 200, 60, 255)
	} else {
		periColor = rl.NewColor(220, 60, 60, 255)
	}
	fonts.Draw(apoStr, px+pad, lineY, 16, apoColor)
	fonts.Draw(periStr, px+pad+140, lineY, 16, periColor)
	lineY += lineH

	// Подсказка.
	hint := "WASD nose | Space thrust | Ctrl retro | M map | E exit"
	hint2 := "Auto: G=prograde  H=retro  J=radial+  K=radial-  N=normal  B=anti  X=off"
	if escaping {
		hint = "TOO FAST — turn sideways, release Space"
	}
	fonts.Draw(hint2, px+pad, py+panelH-40, 11, rl.Gray)
	fonts.Draw(hint, px+pad, py+panelH-24, 12, rl.Gray)
}

func (a *App) drawContract() {
	ignoreClicks := time.Since(a.contractOpenedAt) < 250*time.Millisecond
	sw := int32(rl.GetScreenWidth())
	sh := int32(rl.GetScreenHeight())

	// Затемнение
	rl.DrawRectangle(0, 0, sw, sh, rl.Fade(rl.Black, 0.55))

	// Панель
	pw := int32(520)
	ph := int32(300)
	px := (sw - pw) / 2
	py := (sh - ph) / 2
	panel := rl.NewRectangle(float32(px), float32(py), float32(pw), float32(ph))
	rl.DrawRectangleRec(panel, rl.NewColor(40, 40, 50, 245))
	rl.DrawRectangleLinesEx(panel, 2, rl.NewColor(240, 130, 175, 255))

	title := "HELPER"
	tw := fonts.Measure(title, 36)
	fonts.Draw(title, px+(pw-tw)/2, py+20, 36, rl.NewColor(240, 130, 175, 255))

	// Кнопки
	const bw = 460
	const bh = 60
	bx := px + (pw - bw) / 2

	g1 := ui.Button{
		Rect: rl.NewRectangle(float32(bx), float32(py+100), bw, bh),
		Text: "Give 1 fruit -> brings 4 resources",
	}
	g1.Draw()
	if !ignoreClicks && g1.Clicked() {
		_ = a.nc.AcceptContract(a.contractMobID, "gather4")
		a.showContract = false
		a.contractMobID = ""
		rl.DisableCursor()
	}

	g2 := ui.Button{
		Rect: rl.NewRectangle(float32(bx), float32(py+170), bw, bh),
		Text: "Give 10 spears -> guard me",
	}
	g2.Draw()
	if !ignoreClicks && g2.Clicked() {
		_ = a.nc.AcceptContract(a.contractMobID, "guard")
		a.showContract = false
		a.contractMobID = ""
		rl.DisableCursor()
	}


}

func (a *App) updateDead() {
	if rl.IsKeyPressed(rl.KeyR) {
		a.respawn()
	}
	if rl.IsKeyPressed(rl.KeyEscape) {
		a.returnToMenu()
	}
}

// respawn переподключается и возвращает в игру с полным HP.
func (a *App) respawn() {
	a.log.Info("respawn")
	a.hp = 100
	a.hpReceived = false
	a.startConnect()
	rl.DisableCursor()
}

// returnToMenu возвращает в главное меню.
func (a *App) returnToMenu() {
	a.log.Info("return to menu")
	a.hp = 100
	a.hpReceived = false
	a.mode = state.ModeMenu
	rl.EnableCursor()
	rl.ShowCursor()
}

// drawDead рисует экран смерти с двумя кнопками.
func (a *App) drawDead() {
	sw := int32(rl.GetScreenWidth())
	sh := int32(rl.GetScreenHeight())

	// Затемнение на весь экран.
	rl.DrawRectangle(0, 0, sw, sh, rl.Fade(rl.Black, 0.78))

	// Заголовок.
	title := "YOUR CHARACTER DIED"
	tw := fonts.Measure(title, 72)
	fonts.Draw(title, (sw-tw)/2, sh/3, 72, rl.NewColor(220, 40, 40, 255))

	// Кнопки.
	const bw = 300
	const bh = 54
	bx := (sw - bw) / 2

	respRect := rl.NewRectangle(float32(bx), float32(sh/2+60), bw, bh)
	respBtn := ui.Button{Rect: respRect, Text: "Respawn  [R]"}
	respBtn.Draw()
	if respBtn.Clicked() {
		a.respawn()
	}

	menuRect := rl.NewRectangle(float32(bx), float32(sh/2+130), bw, bh)
	menuBtn := ui.Button{Rect: menuRect, Text: "Main Menu  [Esc]"}
	menuBtn.Draw()
	if menuBtn.Clicked() {
		a.returnToMenu()
	}
}

func (a *App) drawMenu() {
	sw := int32(rl.GetScreenWidth())
	sh := int32(rl.GetScreenHeight())

	title := "LODEEN"
	tw := fonts.Measure(title, 64)
	fonts.Draw(title, (sw-tw)/2, sh/6, 64, rl.RayWhite)

	sub := "federated cooperative multiplayer - pve"
	subW := fonts.Measure(sub, 20)
	fonts.Draw(sub, (sw-subW)/2, sh/6+80, 20, rl.Gray)

	// Вертикальный центр для формы
	centerY := sh/2 - 50
	const fw = 400
	fx := (sw - fw) / 2

	// Ник
	fonts.Draw("Nick", fx, centerY, 18, rl.LightGray)
	nickRect := rl.NewRectangle(float32(fx), float32(centerY+22), fw, 36)
	if ui.TextField(nickRect, a.menuNick, a.menuFocus == 0) {
		a.menuFocus = 0
	}

	// Server
	fonts.Draw("Server", fx, centerY+76, 18, rl.LightGray)
	addrRect := rl.NewRectangle(float32(fx), float32(centerY+98), fw, 36)
	if ui.TextField(addrRect, a.menuAddr, a.menuFocus == 1) {
		a.menuFocus = 1
	}

	// Connect
	connect := ui.Button{Rect: rl.NewRectangle(float32(fx), float32(centerY+160), fw, 46), Text: "Connect"}
	connect.Draw()
	if connect.Clicked() {
		a.startConnect()
	}

	// Fullscreen
	fsLabel := "Fullscreen: Off"
	if rl.IsWindowFullscreen() {
		fsLabel = "Fullscreen: On"
	}
	fs := ui.Button{Rect: rl.NewRectangle(float32(fx), float32(centerY+220), fw, 46), Text: fsLabel}
	fs.Draw()
	if fs.Clicked() {
		rl.ToggleFullscreen()
	}

	// Quit
	quit := ui.Button{Rect: rl.NewRectangle(float32(fx), float32(centerY+280), fw, 46), Text: "Quit"}
	quit.Draw()
	if quit.Clicked() {
		a.quit = true
	}

	// Ошибка
	if a.menuErr != "" {
		fonts.Draw(a.menuErr, fx, centerY+350, 18, rl.Red)
	}

	// Подсказка внизу по центру
	hint := "Tab - switch field - Enter - connect"
	hw := fonts.Measure(hint, 16)
	fonts.Draw(hint, (sw-hw)/2, sh-40, 16, rl.DarkGray)
}

func (a *App) drawPause() {
	sw := int32(rl.GetScreenWidth())
	sh := int32(rl.GetScreenHeight())

	const pw, ph = 400, 360
	px := float32(sw)/2 - pw/2
	py := float32(sh)/2 - ph/2

	panel := rl.NewRectangle(px, py, pw, ph)
	rl.DrawRectangleRec(panel, rl.NewColor(18, 18, 28, 240))
	rl.DrawRectangleLinesEx(panel, 2, rl.Gray)

	title := "Paused"
	tw := fonts.Measure(title, 36)
	fonts.Draw(title, int32(px)+(pw-tw)/2, int32(py)+20, 36, rl.White)

	resume := ui.Button{Rect: rl.NewRectangle(px+40, py+90, pw-80, 44), Text: "Resume"}
	resume.Draw()
	if resume.Clicked() {
		a.mode = state.ModePlaying
	}

	fsLabel := "Fullscreen: Off"
	if rl.IsWindowFullscreen() {
		fsLabel = "Fullscreen: On"
	}
	fs := ui.Button{Rect: rl.NewRectangle(px+40, py+150, pw-80, 44), Text: fsLabel}
	fs.Draw()
	if fs.Clicked() {
		rl.ToggleFullscreen()
	}

	disc := ui.Button{Rect: rl.NewRectangle(px+40, py+210, pw-80, 44), Text: "Disconnect"}
	disc.Draw()
	if disc.Clicked() {
		a.disconnect()
	}

	quit := ui.Button{Rect: rl.NewRectangle(px+40, py+270, pw-80, 44), Text: "Quit"}
	quit.Draw()
	if quit.Clicked() {
		a.quit = true
	}
}

func (a *App) drawHUD() {
	const pad = int32(10)

	// FPS — левый верх
	now := time.Now()
	if now.Sub(a.lastFPSAt) > 500*time.Millisecond {
		// Не показываем FPS первые 2 секунды — окно ещё не стабилизировалось.
		if now.Sub(a.startAt) > 2*time.Second {
			a.cachedFPS = rl.GetFPS()
		} else {
			a.cachedFPS = 0
		}
		a.lastFPSAt = now
	}
	if a.cachedFPS > 0 {
		fonts.Draw(fmt.Sprintf("FPS: %d", a.cachedFPS), pad, pad, 18, rl.RayWhite)
	}

	// Полоска голода под FPS
	barX := pad
	barY := pad + 26
	barW := int32(180)
	barH := int32(14)
	hunger := a.nc.Hunger()
	if hunger < 0 {
		hunger = 0
	} else if hunger > 100 {
		hunger = 100
	}
	rl.DrawRectangle(barX, barY, barW, barH, rl.NewColor(40, 40, 40, 220))
	fillW := int32(float32(barW) * hunger / 100)
	if fillW > 0 {
		var col rl.Color
		switch {
		case hunger > 60:
			col = rl.NewColor(90, 200, 90, 240)
		case hunger > 25:
			col = rl.NewColor(230, 180, 60, 240)
		default:
			col = rl.NewColor(210, 70, 70, 240)
		}
		rl.DrawRectangle(barX, barY, fillW, barH, col)
	}
	rl.DrawRectangleLines(barX, barY, barW, barH, rl.NewColor(120, 120, 130, 255))
	fonts.Draw(fmt.Sprintf("Hunger: %.0f", hunger), barX+barW+8, barY, 16, rl.RayWhite)

	// Полоска голода под FPS
	barX = pad
	barY = pad + 26
	barW = int32(180)
	barH = int32(14)
	hunger = a.nc.Hunger()
	if hunger < 0 {
		hunger = 0
	} else if hunger > 100 {
		hunger = 100
	}
	rl.DrawRectangle(barX, barY, barW, barH, rl.NewColor(40, 40, 40, 220))
	fillW = int32(float32(barW) * hunger / 100)
	if fillW > 0 {
		var col rl.Color
		switch {
		case hunger > 60:
			col = rl.NewColor(90, 200, 90, 240)
		case hunger > 25:
			col = rl.NewColor(230, 180, 60, 240)
		default:
			col = rl.NewColor(210, 70, 70, 240)
		}
		rl.DrawRectangle(barX, barY, fillW, barH, col)
	}
	rl.DrawRectangleLines(barX, barY, barW, barH, rl.NewColor(120, 120, 130, 255))
	fonts.Draw(fmt.Sprintf("Hunger: %.0f", hunger), barX+barW+8, barY, 16, rl.RayWhite)

	// RTT — правый верх
	rtt := a.nc.RTT().Milliseconds()
	rtt = rtt / 10 * 10 // округляем до 10 мс — иначе текстура пересоздаётся каждый кадр
	rttText := fmt.Sprintf("RTT: %d ms", rtt)
	rttW := fonts.Measure(rttText, 18)
	rx, ry := ui.Place(ui.TopRight, pad, pad, rttW, 18)
	fonts.Draw(rttText, rx, ry, 18, rl.RayWhite)

	// Players — правый верх, под RTT
	players := len(a.nc.InterpolatedSnapshot())
	plText := fmt.Sprintf("Players: %d", players)
	plW := fonts.Measure(plText, 18)
	px, py := ui.Place(ui.TopRight, pad, pad+22, plW, 18)
	fonts.Draw(plText, px, py, 18, rl.RayWhite)

	// Speed + Mode — правый верх, под Players
	if a.flight != nil {
		spdText := fmt.Sprintf("Speed: %.0f", a.flight.Speed)
		spdW := fonts.Measure(spdText, 18)
		sx, sy := ui.Place(ui.TopRight, pad, pad+44, spdW, 18)
		fonts.Draw(spdText, sx, sy, 18, rl.RayWhite)

		modeName := "Creative"
		if a.flight.Mode == input.ModeSurvival {
			modeName = "Survival"
		}
		modeText := "Mode: " + modeName + "  [F1]"
		modeW := fonts.Measure(modeText, 18)
		mx, my := ui.Place(ui.TopRight, pad, pad+66, modeW, 18)
		fonts.Draw(modeText, mx, my, 18, rl.Yellow)
	}

	// Подсказка подбора — над чатом
	pickupHint := "F - pick up resource   |   T - chat   |   Esc - pause"
	pw := fonts.Measure(pickupHint, 18)
	phx, phy := ui.Place(ui.BottomLeft, pad, pad+70, pw, 18)
	fonts.Draw(pickupHint, phx, phy, 18, rl.Yellow)

	// Help — левый низ
	help := "WASD - move - Space up - Shift down - Q/E roll - Mouse wheel slot/speed"
	hw := fonts.Measure(help, 16)
	hx, hy := ui.Place(ui.BottomLeft, pad, pad+40, hw, 16)
	fonts.Draw(help, hx, hy, 16, rl.Gray)

	// Чат — левый низ
	sw := int(rl.GetScreenWidth())
	sh := int(rl.GetScreenHeight())
	a.chat.Draw(sw, sh-30)

	if a.showInventory {
		a.drawInventory()
	}
	if a.showCraft {
		a.drawCraft()
	}

	// Hotbar внизу по центру — только в Survival
	if a.flight != nil && a.flight.Mode == input.ModeSurvival {
		a.drawHotbar()
	}
	// HP bar.
	hpVal := a.myHP()
	hpBarW := float32(200)
	hpBarH := float32(16)
	hpBarX := int32(20)
	hpBarY := int32(60)
	rl.DrawRectangle(hpBarX, hpBarY, int32(hpBarW), int32(hpBarH), rl.NewColor(60, 20, 20, 200))
	hpFill := hpBarW * float32(hpVal) / 100.0
	if hpFill < 0 {
		hpFill = 0
	}
	rl.DrawRectangle(hpBarX, hpBarY, int32(hpFill), int32(hpBarH), rl.NewColor(200, 40, 40, 255))
	rl.DrawRectangleLines(hpBarX, hpBarY, int32(hpBarW), int32(hpBarH), rl.Black)
	fonts.Draw(fmt.Sprintf("HP: %d", hpVal), hpBarX+int32(hpBarW)+8, hpBarY, 16, rl.RayWhite)
}

// drawHotbar — первая строка инвентаря внизу по центру.
func (a *App) drawHotbar() {
	const slots = 16
	const cell = int32(44)
	const pad3 = int32(6)

	sw := int32(rl.GetScreenWidth())
	sh := int32(rl.GetScreenHeight())
	barW := slots*cell + pad3*2
	barH := cell + pad3*2
	px := (sw - barW) / 2
	py := sh - barH - 20

	panel := rl.NewRectangle(float32(px), float32(py), float32(barW), float32(barH))
	rl.DrawRectangleRec(panel, rl.NewColor(15, 15, 25, 200))
	rl.DrawRectangleLinesEx(panel, 2, rl.NewColor(120, 120, 140, 255))

	inv := a.nc.Inventory()

	for i := 0; i < slots; i++ {
		cx := px + pad3 + int32(i)*cell
		cy := py + pad3
		rect := rl.NewRectangle(float32(cx), float32(cy), float32(cell-2), float32(cell-2))
		rl.DrawRectangleRec(rect, rl.NewColor(30, 30, 40, 255))
		rl.DrawRectangleLinesEx(rect, 1, rl.NewColor(70, 70, 90, 255))

		if i == a.selectedSlot {
			rl.DrawRectangleLinesEx(rect, 3, rl.NewColor(255, 220, 90, 255))
		}

		typ := ""
		if i < len(a.invSlots) {
			typ = a.invSlots[i]
		}
		if typ != "" {
			col := itemColor(typ)
			rl.DrawRectangleRec(rl.NewRectangle(float32(cx+9), float32(cy+9), float32(cell-20), float32(cell-20)), col)
			if n := inv[typ]; n > 0 {
				fonts.Draw(fmt.Sprintf("%d", n), cx+4, cy+cell-20, 14, rl.White)
			}
		}
	}
}

func (a *App) startConnect() {
	if a.nc.Status() == clientnet.StatusConnected {
		return
	}
	if a.menuNick == "" {
		a.menuErr = "nick required"
		return
	}
	if a.menuAddr == "" {
		a.menuErr = "server address required"
		return
	}
	a.menuErr = ""
	a.log.Info("connecting", "addr", a.menuAddr, "nick", a.menuNick)

	start := time.Now()
	welcome, err := a.nc.Connect(a.menuAddr, a.menuNick)
	if err != nil {
		a.menuErr = "connect failed: " + err.Error()
		a.log.Warn("connect failed", "err", err)
		return
	}
	a.hp = 100
	a.log.Info("connected", "took", time.Since(start).String())

	spawn := render.SpawnFromID(welcome.PlayerID)
	a.log.Info("spawn", "x", spawn.X, "z", spawn.Z)

	a.flight = input.New(spawn)
	a.nc.StartStateLoop()
	a.mode = state.ModePlaying
}

func (a *App) disconnect() {
	a.setCursorCaptured(false)
	a.nc.Disconnect()
	a.flight = nil
	a.mode = state.ModeMenu
}

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

// seedInSight возвращает ID ближайшего seed в радиусе 5 юнитов
// и в конусе ~53° по направлению взгляда.
// wellInSight возвращает ID ближайшего источника пресной воды
// в радиусе 6 юнитов и в конусе взгляда.
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

// mammothInReach возвращает ID ближайшего мамонта в радиусе 5 юнитов
// и в конусе взгляда.
// houseInSight возвращает ID ближайшего дома в радиусе 6 юнитов.
// pinkMobInSight — ближайший розовый моб в конусе взгляда.
// factoryInSight — ближайший завод в радиусе 6 юнитов и в конусе взгляда.
// rocketInSight — ближайшая ракета в радиусе 8 юнитов и в конусе взгляда.
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

// placeForward возвращает точку перед игроком и её yaw.
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

func (a *App) tryPickup() {
	if a.flight == nil {
		return
	}
	const pickupRange = 5.0
	pos := a.flight.Pos
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

// drawInventory — сетка 16×16 с подсветкой первой строки и tooltip.
func (a *App) drawInventory() {
	const cols, rows = 16, 16
	const cell = int32(38)
	const pad2 = int32(10)

	sw := int32(rl.GetScreenWidth())
	sh := int32(rl.GetScreenHeight())
	gridW := cols*cell + pad2*2
	gridH := rows*cell + pad2*2 + 30
	px := (sw - gridW) / 2
	py := (sh - gridH) / 2

	rl.DrawRectangle(0, 0, sw, sh, rl.Fade(rl.Black, 0.6))
	panel := rl.NewRectangle(float32(px), float32(py), float32(gridW), float32(gridH))
	rl.DrawRectangleRec(panel, rl.NewColor(20, 20, 30, 245))
	rl.DrawRectangleLinesEx(panel, 2, rl.NewColor(120, 120, 140, 255))

	fonts.Draw("Inventory   [I / Esc close]   LMB: drag item between slots",
		px+pad2, py+6, 20, rl.RayWhite)

	inv := a.nc.Inventory()
	mouse := rl.GetMousePosition()
	gridY := py + 30 + pad2

	var hoveredName string
	var hoveredSlot int = -1

	// Обработка мыши: клик по слоту.
	if rl.IsMouseButtonPressed(rl.MouseLeftButton) {
		// Найти, по какому слоту кликнули.
		for i := 0; i < cols*rows; i++ {
			col := int32(i % cols)
			row := int32(i / cols)
			cx := px + pad2 + col*cell
			cy := gridY + row*cell
			rect := rl.NewRectangle(float32(cx), float32(cy), float32(cell-2), float32(cell-2))
			if rl.CheckCollisionPointRec(mouse, rect) {
				a.dragging = true
				a.dragFrom = i
				break
			}
		}
	}
	if rl.IsMouseButtonReleased(rl.MouseLeftButton) && a.dragging {
		// Куда отпустили.
		for i := 0; i < cols*rows; i++ {
			col := int32(i % cols)
			row := int32(i / cols)
			cx := px + pad2 + col*cell
			cy := gridY + row*cell
			rect := rl.NewRectangle(float32(cx), float32(cy), float32(cell-2), float32(cell-2))
			if rl.CheckCollisionPointRec(mouse, rect) {
				if i != a.dragFrom {
					// Поменять местами.
					a.invSlots[a.dragFrom], a.invSlots[i] = a.invSlots[i], a.invSlots[a.dragFrom]
				}
				break
			}
		}
		a.dragging = false
	}

	for i := 0; i < cols*rows; i++ {
		col := int32(i % cols)
		row := int32(i / cols)
		cx := px + pad2 + col*cell
		cy := gridY + row*cell
		rect := rl.NewRectangle(float32(cx), float32(cy), float32(cell-2), float32(cell-2))

		bg := rl.NewColor(32, 32, 44, 255)
		if row == 0 {
			bg = rl.NewColor(60, 50, 30, 255)
		}
		rl.DrawRectangleRec(rect, bg)
		rl.DrawRectangleLinesEx(rect, 1, rl.NewColor(70, 70, 90, 255))

		// Рамка выбранного слота.
		if row == 0 && int(col) == a.selectedSlot {
			rl.DrawRectangleLinesEx(rect, 3, rl.NewColor(255, 220, 90, 255))
		}

		// Подсветка слота, если тащим предмет и наводим на этот слот.
		if a.dragging && rl.CheckCollisionPointRec(mouse, rect) {
			rl.DrawRectangleLinesEx(rect, 2, rl.NewColor(80, 200, 120, 255))
			hoveredSlot = i
		}

		typ := a.invSlots[i]
		if typ == "" {
			continue
		}
		n := inv[typ]
		col2 := itemColor(typ)
		icon := rl.NewRectangle(float32(cx+8), float32(cy+8), float32(cell-18), float32(cell-18))
		rl.DrawRectangleRec(icon, col2)
		fonts.Draw(fmt.Sprintf("%d", n), cx+3, cy+cell-18, 14, rl.White)

		if rl.CheckCollisionPointRec(mouse, rect) {
			hoveredName = fmt.Sprintf("%s  x%d", itemName(typ), n)
		}
	}

	// Призрак перетаскиваемого предмета под курсором.
	if a.dragging && a.invSlots[a.dragFrom] != "" {
		typ := a.invSlots[a.dragFrom]
		col2 := itemColor(typ)
		gx := int32(mouse.X) - cell/2 + 2
		gy := int32(mouse.Y) - cell/2 + 2
		ghost := rl.NewRectangle(float32(gx), float32(gy), float32(cell-4), float32(cell-4))
		rl.DrawRectangleRec(ghost, rl.NewColor(col2.R, col2.G, col2.B, 200))
		rl.DrawRectangleLinesEx(ghost, 2, rl.White)
		fonts.Draw(itemName(typ), gx+4, gy+cell-20, 14, rl.White)
	}

	// Tooltip.
	if hoveredName != "" && !a.dragging {
		tw := fonts.Measure(hoveredName, 16)
		tx := int32(mouse.X) + 18
		ty := int32(mouse.Y) + 14
		if tx+tw+20 > sw {
			tx = int32(mouse.X) - tw - 22
		}
		if ty+30 > sh {
			ty = sh - 34
		}
		box := rl.NewRectangle(float32(tx-6), float32(ty-4), float32(tw+12), 24)
		rl.DrawRectangleRec(box, rl.NewColor(10, 10, 20, 245))
		rl.DrawRectangleLinesEx(box, 1, rl.NewColor(200, 200, 220, 255))
		fonts.Draw(hoveredName, tx, ty, 16, rl.White)
	}
	_ = hoveredSlot
}

// itemColor возвращает цвет иконки для типа ресурса.
func itemColor(typ string) rl.Color {
	switch typ {
	case "stone":
		return rl.NewColor(140, 140, 150, 255)
	case "water":
		return rl.NewColor(60, 120, 220, 255)
	case "liana":
		return rl.NewColor(80, 160, 60, 255)
	case "leash":
		return rl.NewColor(160, 120, 70, 255)
	case "house":
		return rl.NewColor(140, 95, 55, 255)
	case "saddle":
		return rl.NewColor(90, 60, 40, 255)
	case "boat":
		return rl.NewColor(120, 80, 40, 255)
	case "solar":
		return rl.NewColor(40, 80, 140, 255)
	case "battery":
		return rl.NewColor(60, 60, 80, 255)
	case "factory":
		return rl.NewColor(110, 100, 90, 255)
	case "steel":
		return rl.NewColor(180, 180, 190, 255)
	case "gear":
		return rl.NewColor(160, 160, 130, 255)
	case "circuit":
		return rl.NewColor(80, 200, 120, 255)
	case "drone":
		return rl.NewColor(100, 150, 220, 255)
	case "rocket":
		return rl.NewColor(220, 80, 80, 255)
	case "wood":
		return rl.NewColor(120, 80, 40, 255)
	case "ore":
		return rl.NewColor(200, 170, 60, 255)
	case "fruit":
		return rl.NewColor(230, 70, 70, 255)
	case "meat":
		return rl.NewColor(200, 100, 100, 255)
	case "spear":
		return rl.NewColor(180, 160, 120, 255)
	case "torch":
		return rl.NewColor(240, 180, 80, 255)
	}
	return rl.White
}

// itemName — человекочитаемое имя.
func itemName(typ string) string {
	switch typ {
	case "stone":
		return "Stone"
	case "wood":
		return "Wood"
	case "ore":
		return "Ore"
	case "fruit":
		return "Fruit"
	case "meat":
		return "Meat"
	case "spear":
		return "Spear"
	case "torch":
		return "Torch"
	case "water":
		return "Water"
	case "liana":
		return "Liana"
	case "leash":
		return "Leash"
	case "house":
		return "House"
	case "saddle":
		return "Saddle"
	case "boat":
		return "Boat"
	case "solar":
		return "Solar Panel"
	case "battery":
		return "Battery"
	case "factory":
		return "Factory"
	case "steel":
		return "Steel"
	case "gear":
		return "Gear"
	case "circuit":
		return "Circuit"
	case "drone":
		return "Drone"
	case "rocket":
		return "Rocket"
	}
	return typ
}

// craftRecipes — клиентский список рецептов для UI.
// Сервер валидирует независимо, здесь — только отображение.
type craftRecipe struct {
	id   string
	name string
	out  string
	req  map[string]int
}

var craftRecipes = []craftRecipe{
	{id: "spear", name: "Spear", out: "spear", req: map[string]int{"stone": 2, "wood": 1}},
	{id: "torch", name: "Torch", out: "torch", req: map[string]int{"stone": 1, "wood": 1}},
	{id: "leash", name: "Leash", out: "leash", req: map[string]int{"liana": 2}},
	{id: "house", name: "House", out: "house", req: map[string]int{"wood": 50}},
	{id: "saddle", name: "Saddle", out: "saddle", req: map[string]int{"liana": 4}},
	{id: "boat", name: "Boat", out: "boat", req: map[string]int{"wood": 20}},
	{id: "solar", name: "Solar Panel", out: "solar", req: map[string]int{"ore": 10, "stone": 5}},
	{id: "battery", name: "Battery", out: "battery", req: map[string]int{"ore": 15, "stone": 10}},
	{id: "factory", name: "Factory", out: "factory", req: map[string]int{"ore": 30, "wood": 20, "stone": 20}},
}

// drawCraft — окно крафта со списком рецептов.
func (a *App) drawCraft() {
	sw := int32(rl.GetScreenWidth())
	sh := int32(rl.GetScreenHeight())
	pw := int32(660)
	ph := int32(120 + 90*len(craftRecipes))
	px := (sw - pw) / 2
	py := (sh - ph) / 2

	rl.DrawRectangle(0, 0, sw, sh, rl.Fade(rl.Black, 0.6))
	panel := rl.NewRectangle(float32(px), float32(py), float32(pw), float32(ph))
	rl.DrawRectangleRec(panel, rl.NewColor(20, 20, 30, 245))
	rl.DrawRectangleLinesEx(panel, 2, rl.NewColor(120, 120, 140, 255))

	fonts.Draw("Crafting   [C / Esc to close]", px+14, py+10, 22, rl.RayWhite)

	inv := a.nc.Inventory()
	rowH := int32(90)
	for i, rec := range craftRecipes {
		cardY := py + 50 + int32(i)*rowH
		card := rl.NewRectangle(float32(px+12), float32(cardY), float32(pw-24), float32(rowH-8))
		rl.DrawRectangleRec(card, rl.NewColor(30, 30, 42, 255))
		rl.DrawRectangleLinesEx(card, 1, rl.NewColor(70, 70, 90, 255))

		// иконка
		var col rl.Color
		switch rec.out {
		case "spear":
			col = rl.NewColor(180, 160, 120, 255)
		case "torch":
			col = rl.NewColor(240, 180, 80, 255)
		default:
			col = rl.White
		}
		rl.DrawRectangleRec(rl.NewRectangle(float32(px+22), float32(cardY+9), 56, 56), col)

		// имя
		fonts.Draw(rec.name, px+94, cardY+8, 22, rl.RayWhite)

		// требования
		reqText := ""
		canCraft := true
		for _, k := range []string{"stone", "wood", "ore", "fruit", "meat", "spear", "torch", "water", "liana", "leash", "house", "saddle", "boat", "solar", "battery", "factory", "steel", "gear", "circuit", "drone", "rocket"} {
			if need, ok := rec.req[k]; ok {
				have := inv[k]
				reqText += fmt.Sprintf("%s %d/%d   ", k, have, need)
				if have < need {
					canCraft = false
				}
			}
		}
		reqCol := rl.NewColor(90, 220, 90, 255)
		if !canCraft {
			reqCol = rl.NewColor(230, 90, 90, 255)
		}
		fonts.Draw(reqText, px+94, cardY+42, 16, reqCol)

		// кнопка
		btn := ui.Button{
			Rect: rl.NewRectangle(float32(px+pw-150), float32(cardY+22), 130, 42),
			Text: "Craft",
		}
		btn.Draw()
		if btn.Clicked() && canCraft {
			_ = a.nc.CraftItem(rec.id)
		}
	}
}

// drawHeldItem — предмет в руках: куб перед камерой.
// Работает только в Survival, если в selectedSlot есть предмет.
func (a *App) drawHeldItem() {
	if a.flight == nil || a.flight.Mode != input.ModeSurvival {
		return
	}
	typ := a.heldItem()
	if typ == "" {
		return
	}
	col := itemColor(typ)

	// Локальные оси камеры
	fw := rl.Vector3Normalize(rl.Vector3Subtract(a.camera.Target, a.camera.Position))
	right := rl.Vector3Normalize(rl.Vector3CrossProduct(fw, a.camera.Up))
	up := rl.Vector3Normalize(rl.Vector3CrossProduct(right, fw))

	// Позиция в правом нижнем углу, чуть впереди
	pos := a.camera.Position
	pos = rl.Vector3Add(pos, rl.Vector3Scale(fw, 0.7))
	pos = rl.Vector3Add(pos, rl.Vector3Scale(right, 0.4))
	pos = rl.Vector3Subtract(pos, rl.Vector3Scale(up, 0.35))

	size := float32(0.25)
	rl.DrawCube(pos, size, size, size, col)
	rl.DrawCubeWires(pos, size, size, size, rl.Black)
}

// heldItem возвращает тип предмета в selectedSlot (пустая строка — ничего).
func (a *App) myHP() int {
	return a.hp
}

func (a *App) heldItem() string {
	if a.selectedSlot < 0 || a.selectedSlot >= len(a.invSlots) {
		return ""
	}
	return a.invSlots[a.selectedSlot]
}

// syncInvSlots — разложить новые предметы из серверного инвентаря по слотам.
// Уже занятые слоты не трогает, освобождает слоты при исчезновении предмета.
func (a *App) syncInvSlots() {
	inv := a.nc.Inventory()

	// Какие типы уже лежат в слотах.
	present := make(map[string]bool, len(a.invSlots))
	for _, t := range a.invSlots {
		if t != "" {
			present[t] = true
		}
	}

	// Новые предметы → в первый пустой слот.
	for t, n := range inv {
		if n <= 0 || present[t] {
			continue
		}
		for i := 0; i < len(a.invSlots); i++ {
			if a.invSlots[i] == "" {
				a.invSlots[i] = t
				present[t] = true
				break
			}
		}
	}

	// Очистить слоты, где предмет пропал.
	for i, t := range a.invSlots {
		if t != "" && inv[t] <= 0 {
			a.invSlots[i] = ""
		}
	}
}

// updateProjectiles — локальная симуляция снарядов с детекцией коллизии.
func (a *App) updateProjectiles() {
	if len(a.projectiles) == 0 {
		return
	}
	now := time.Now()
	dt := rl.GetFrameTime()
	const speed = 60.0
	const ttl = 1.5
	const mobR = 2.5    // радиус моба (куб 2x2x2 + запас)
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

// segSphereHit — пересечение отрезка [a,b] со сферой (c, r).
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

// drawProjectiles — рисует локальные снаряды жёлтыми сферами.
func (a *App) drawProjectiles() {
	for _, p := range a.projectiles {
		rl.DrawSphere(p.pos, 0.4, rl.NewColor(255, 220, 60, 255))
		rl.DrawSphereWires(p.pos, 0.4, 12, 12, rl.NewColor(180, 140, 20, 255))
	}
}
