package main

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/prometheus/client_golang/prometheus"

	servernet "github.com/spacedreamer99/lodeen/internal/server/net"
	"github.com/spacedreamer99/lodeen/internal/shared/config"
	"github.com/spacedreamer99/lodeen/internal/shared/logger"
	"github.com/spacedreamer99/lodeen/internal/shared/metrics"
	"github.com/spacedreamer99/lodeen/internal/shared/version"
)

func main() {
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
	log := logger.New(cfg.LogLevel, "server")
	log.Info("starting lodeen server",
		"version", version.String(),
		"env", cfg.Env,
		"tcp", cfg.Server.TCPAddr,
		"admin", cfg.Server.AdminAddr,
		"tick_rate", cfg.Server.TickRate,
	)

	reg := metrics.New()
	srvMetrics := &servernet.Metrics{
		PlayersConnected: prometheus.NewGauge(prometheus.GaugeOpts{
			Name: "lodeen_server_players_connected",
			Help: "Current number of connected players.",
		}),
		TicksTotal: prometheus.NewCounter(prometheus.CounterOpts{
			Name: "lodeen_server_ticks_total",
			Help: "Total number of server ticks.",
		}),
		ChatMessages: prometheus.NewCounter(prometheus.CounterOpts{
			Name: "lodeen_server_chat_messages_total",
			Help: "Total number of chat messages.",
		}),
	}
	reg.MustRegister(srvMetrics.PlayersConnected, srvMetrics.TicksTotal, srvMetrics.ChatMessages)

	srv := servernet.New(cfg.Server.TCPAddr, cfg.Server.TickRate, log, srvMetrics)
	if err := srv.Start(); err != nil {
		return fmt.Errorf("start tcp server: %w", err)
	}

	adminMux := http.NewServeMux()
	adminMux.HandleFunc("/healthz", func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte("ok"))
	})
	adminMux.HandleFunc("/readyz", func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte("ok"))
	})
	adminMux.Handle("/metrics", reg.Handler())
	adminMux.HandleFunc("/version", func(w http.ResponseWriter, _ *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		info := version.Get()
		fmt.Fprintf(w, `{"version":%q,"commit":%q,"date":%q,"go":%q}`,
			info.Version, info.Commit, info.Date, info.GoVersion)
	})

	adminSrv := &http.Server{
		Addr:              cfg.Server.AdminAddr,
		Handler:           adminMux,
		ReadHeaderTimeout: 5 * time.Second,
	}

	adminErrCh := make(chan error, 1)
	go func() {
		log.Info("admin http listening", "addr", cfg.Server.AdminAddr)
		if err := adminSrv.ListenAndServe(); err != nil && !errors.Is(err, http.ErrServerClosed) {
			adminErrCh <- err
		}
	}()

	ctx, stop := signal.NotifyContext(context.Background(), syscall.SIGINT, syscall.SIGTERM)
	defer stop()

	select {
	case <-ctx.Done():
		log.Info("shutdown signal received")
	case err := <-adminErrCh:
		return fmt.Errorf("admin http: %w", err)
	}

	shutdownCtx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	_ = adminSrv.Shutdown(shutdownCtx)
	if err := srv.Stop(shutdownCtx); err != nil {
		log.Warn("tcp server stop", "err", err)
	}
	log.Info("shutdown complete")
	return nil
}
