package app

import (
	"time"

	"github.com/spacedreamer99/lodeen/internal/client/input"
	clientnet "github.com/spacedreamer99/lodeen/internal/client/net"
	"github.com/spacedreamer99/lodeen/internal/client/render"
	"github.com/spacedreamer99/lodeen/internal/client/state"
)

func (a *App) startConnect() {
	if a.nc.Status() == clientnet.StatusConnected {
		return
	}
	if a.ui.menuNick == "" {
		a.ui.menuErr = "nick required"
		return
	}
	if a.ui.menuAddr == "" {
		a.ui.menuErr = "server address required"
		return
	}
	a.ui.menuErr = ""
	a.log.Info("connecting", "addr", a.ui.menuAddr, "nick", a.ui.menuNick)

	start := time.Now()
	welcome, err := a.nc.Connect(a.ui.menuAddr, a.ui.menuNick)
	if err != nil {
		a.ui.menuErr = "connect failed: " + err.Error()
		a.log.Warn("connect failed", "err", err)
		return
	}
	a.player.hp = 100
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
