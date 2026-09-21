package app

import (
	"fmt"
	"log/slog"
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

	cursorCaptured bool
}

func New(cfg *config.Config, log *slog.Logger) *App {
	return &App{
		cfg:      cfg,
		log:      log,
		mode:     state.ModeMenu,
		menuNick: "pilot",
		menuAddr: cfg.Client.StartAddr,
	}
}

func (a *App) Run() error {
	rl.SetConfigFlags(rl.FlagMsaa4xHint | rl.FlagVsyncHint | rl.FlagWindowResizable)
	rl.InitWindow(screenW, screenH, "LODEEN")
	defer rl.CloseWindow()
	rl.SetTargetFPS(60)
	rl.SetExitKey(rl.KeyNull)

	fonts.Load(28)

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

	for !rl.WindowShouldClose() {
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

	// Wayland fallback: если DisableCursor не центрует — центруем вручную.
	midX := rl.GetScreenWidth() / 2
	midY := rl.GetScreenHeight() / 2
	if rl.GetMouseX() != int32(midX) || rl.GetMouseY() != int32(midY) {
		rl.SetMousePosition(midX, midY)
	}

	fw := a.flight.Forward()
	a.camera.Position = a.flight.Pos
	a.camera.Target = rl.Vector3Add(a.flight.Pos, fw)
	a.camera.Up = a.flight.CameraUp()

	a.nc.SetState(protocol.PlayerState{
		X: a.flight.Pos.X, Y: a.flight.Pos.Y, Z: a.flight.Pos.Z,
		Yaw: a.flight.Yaw, Pitch: a.flight.Pitch,
	})
}

func (a *App) updatePaused() {
	a.setCursorCaptured(false)

	if rl.IsKeyPressed(rl.KeyEscape) {
		a.mode = state.ModePlaying
	}
}

func (a *App) draw() {
	rl.BeginDrawing()
	defer rl.EndDrawing()

	switch a.mode {
	case state.ModeMenu:
		rl.ClearBackground(rl.NewColor(12, 12, 22, 255))
		a.drawMenu()
	default:
		rl.ClearBackground(rl.NewColor(4, 4, 12, 255))
		rl.BeginMode3D(a.camera)
		a.scene.Draw()
		rl.EndMode3D()

		render.DrawPlayers(a.nc.Snapshot(), a.nc.PlayerID(), a.camera)

		a.drawHUD()
		if a.mode == state.ModePaused {
			a.drawPause()
		}
	}
}

func (a *App) drawMenu() {
	title := "LODEEN"
	tw := fonts.Measure(title, 64)
	fonts.Draw(title, (screenW-tw)/2, 70, 64, rl.RayWhite)

	sub := "federated cooperative multiplayer - pve"
	sw := fonts.Measure(sub, 20)
	fonts.Draw(sub, (screenW-sw)/2, 145, 20, rl.Gray)

	const fx = 440
	const fw = 400

	fonts.Draw("Nick", fx, 220, 18, rl.LightGray)
	nickRect := rl.NewRectangle(fx, 245, fw, 36)
	if ui.TextField(nickRect, a.menuNick, a.menuFocus == 0) {
		a.menuFocus = 0
	}

	fonts.Draw("Server", fx, 300, 18, rl.LightGray)
	addrRect := rl.NewRectangle(fx, 325, fw, 36)
	if ui.TextField(addrRect, a.menuAddr, a.menuFocus == 1) {
		a.menuFocus = 1
	}

	connect := ui.Button{Rect: rl.NewRectangle(fx, 400, fw, 46), Text: "Connect"}
	connect.Draw()
	if connect.Clicked() {
		a.startConnect()
	}

	quit := ui.Button{Rect: rl.NewRectangle(fx, 460, fw, 46), Text: "Quit"}
	quit.Draw()
	if quit.Clicked() {
		rl.CloseWindow()
	}

	if a.menuErr != "" {
		fonts.Draw(a.menuErr, fx, 530, 18, rl.Red)
	}

	hint := "Tab - switch field - Enter - connect"
	hw := fonts.Measure(hint, 16)
	fonts.Draw(hint, (screenW-hw)/2, screenH-40, 16, rl.DarkGray)
}

func (a *App) drawPause() {
	rl.DrawRectangle(0, 0, screenW, screenH, rl.Fade(rl.Black, 0.55))

	const pw, ph = 400, 300
	px := float32(screenW)/2 - pw/2
	py := float32(screenH)/2 - ph/2

	panel := rl.NewRectangle(px, py, pw, ph)
	rl.DrawRectangleRec(panel, rl.NewColor(18, 18, 28, 240))
	rl.DrawRectangleLinesEx(panel, 2, rl.Gray)

	title := "Paused"
	tw := fonts.Measure(title, 36)
	fonts.Draw(title, int32(px)+(pw-tw)/2, int32(py)+20, 36, rl.White)

	resume := ui.Button{Rect: rl.NewRectangle(px+40, py+100, pw-80, 44), Text: "Resume"}
	resume.Draw()
	if resume.Clicked() {
		a.mode = state.ModePlaying
	}

	disc := ui.Button{Rect: rl.NewRectangle(px+40, py+160, pw-80, 44), Text: "Disconnect"}
	disc.Draw()
	if disc.Clicked() {
		a.disconnect()
	}

	quit := ui.Button{Rect: rl.NewRectangle(px+40, py+220, pw-80, 44), Text: "Quit"}
	quit.Draw()
	if quit.Clicked() {
		rl.CloseWindow()
	}
}

func (a *App) drawHUD() {
	rl.DrawFPS(10, 10)

	rtt := a.nc.RTT().Milliseconds()
	fonts.Draw(fmt.Sprintf("RTT: %d ms", rtt), screenW-160, 10, 18, rl.RayWhite)

	help := "WASD - move - Space up - Shift down - Q/E roll - Ctrl boost - T chat - Esc pause"
	fonts.Draw(help, 10, screenH-52, 16, rl.Gray)

	a.chat.Draw(screenW, screenH-30)
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
