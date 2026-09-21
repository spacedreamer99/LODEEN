package main

import (
	"fmt"
	"os"
	"runtime"

	"github.com/spacedreamer99/lodeen/internal/client/app"
	"github.com/spacedreamer99/lodeen/internal/shared/config"
	"github.com/spacedreamer99/lodeen/internal/shared/logger"
)

func main() {
	runtime.LockOSThread()
	if err := run(); err != nil {
		fmt.Fprintln(os.Stderr, "fatal:", err)
		os.Exit(1)
	}
}

func run() error {
	cfg, err := config.Load()
	if err != nil {
		return fmt.Errorf("config: %w", err)
	}
	log := logger.New(cfg.LogLevel, "client")
	a := app.New(cfg, log)
	return a.Run()
}
