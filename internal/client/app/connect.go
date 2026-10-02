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
