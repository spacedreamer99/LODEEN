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

	lastKeys  string
	lastMouse string
	lastLogAt time.Time
	lastPos   rl.Vector3
}

func New(cfg *config.Config, log *slog.Logger) *App {
	return &App{
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

	switch a.mode {
	case state.ModeMenu:
		a.updateMenu()
	case state.ModePlaying:
		a.updatePlaying(dt)
	case state.ModePaused:
		a.updatePaused()
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
	if a.chat.Open {
		a.setCursorCaptured(false)
		if text, ok := a.chat.Update(); ok && text != "" {
			if err := a.nc.SendChat(text); err != nil {
				a.log.Warn("send chat", "err", err)
			}
		}
		return
	}

	if rl.IsKeyPressed(rl.KeyT) {
		a.chat.Begin()
		return
	}
	if rl.IsKeyPressed(rl.KeyF) {
		a.tryPickup()
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
		render.DrawPlayers(a.nc.InterpolatedSnapshot(), a.nc.PlayerID(), a.camera)
		render.DrawResources(a.nc.Resources())
		rl.EndMode3D()
	}

	// UI — в буфер размером с окно
	rl.BeginTextureMode(a.uiTarget)
	rl.ClearBackground(rl.Blank)

	switch a.mode {
	case state.ModeMenu:
		a.drawMenu()
	default:
		if a.mode == state.ModePaused {
			// Затемнение
			rl.DrawRectangle(0, 0, sw, sh, rl.Fade(rl.Black, 0.55))
			a.drawPause()
		}
		a.drawHUD()
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
		a.cachedFPS = rl.GetFPS()
		a.lastFPSAt = now
	}
	fonts.Draw(fmt.Sprintf("FPS: %d", a.cachedFPS), pad, pad, 18, rl.RayWhite)

	// Инвентарь — левый верх, под FPS
	inv := a.nc.Inventory()
	if len(inv) > 0 {
		y := pad + 24
		for _, item := range []string{"stone", "wood", "ore"} {
			if n, ok := inv[item]; ok {
				fonts.Draw(fmt.Sprintf("%s: %d", item, n), pad, y, 18, rl.RayWhite)
				y += 22
			}
		}
	}

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

	// Speed — правый верх, под Players
	if a.flight != nil {
		spdText := fmt.Sprintf("Speed: %.0f", a.flight.Speed)
		spdW := fonts.Measure(spdText, 18)
		sx, sy := ui.Place(ui.TopRight, pad, pad+44, spdW, 18)
		fonts.Draw(spdText, sx, sy, 18, rl.RayWhite)
	}

	// Подсказка подбора — над чатом
	pickupHint := "F - pick up resource   |   T - chat   |   Esc - pause"
	pw := fonts.Measure(pickupHint, 18)
	phx, phy := ui.Place(ui.BottomLeft, pad, pad+70, pw, 18)
	fonts.Draw(pickupHint, phx, phy, 18, rl.Yellow)

	// Help — левый низ
	help := "WASD - move - Space up - Shift down - Q/E roll - Mouse wheel speed"
	hw := fonts.Measure(help, 16)
	hx, hy := ui.Place(ui.BottomLeft, pad, pad+40, hw, 16)
	fonts.Draw(help, hx, hy, 16, rl.Gray)

	// Чат — левый низ
	sw := int(rl.GetScreenWidth())
	sh := int(rl.GetScreenHeight())
	a.chat.Draw(sw, sh-30)
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
