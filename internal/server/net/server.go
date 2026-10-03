// Package net implements the LODEEN TCP game server.
//
// Layout:
//   - server.go      — Server struct, lifecycle (New/Start/Stop)
//   - server_conn.go — accept loop, per-connection handshake + IO
//   - server_tick.go — fixed-dt tick loop + per-tick phases
//   - server_util.go — broadcast, frame decode, ID generation
package net

import (
	"context"
	"fmt"
	"log/slog"
	"net"
	"sync"
	"time"

	"github.com/prometheus/client_golang/prometheus"

	"github.com/spacedreamer99/lodeen/internal/shared/protocol"
)

const (
	readTimeout  = 60 * time.Second
	writeTimeout = 5 * time.Second
	sendBufSize  = 256
	pickupRadius = 5.0
)

type Metrics struct {
	PlayersConnected prometheus.Gauge
	TicksTotal       prometheus.Counter
	ChatMessages     prometheus.Counter
	TickDuration     prometheus.Histogram
	SnapshotBytes    prometheus.Histogram
	RTTSeconds       prometheus.Histogram
}

// Server — корневой объект игрового сервера.
// Хранит: реестр клиентов, состояние мира, доменные map'ы сущностей.
type Server struct {
	addr     string
	tickRate int
	log      *slog.Logger
	metrics  *Metrics

	ln net.Listener

	mu      sync.RWMutex
	clients map[string]*Client

	resourcesMu sync.RWMutex
	resources   map[string]protocol.Resource

	mammothsMu sync.RWMutex
	mammoths   map[string]*Mammoth

	wellsMu sync.RWMutex
	wells   map[string]*protocol.Well

	housesMu sync.RWMutex
	houses   map[string]*House

	solarMu sync.RWMutex
	solar   map[string]*Solar

	batteriesMu sync.RWMutex
	batteries   map[string]*Battery

	factoriesMu sync.RWMutex
	factories   map[string]*Factory

	rockets *store[*Rocket]

	boatsMu sync.RWMutex
	boats   map[string]*Boat

	mobsMu sync.RWMutex
	mobs   map[string]*Mob

	projMu      sync.RWMutex
	projectiles map[string]*Projectile

	tick  uint64
	world *WorldState

	wg        sync.WaitGroup
	closeOnce sync.Once
	done      chan struct{}
}

func New(addr string, tickRate int, log *slog.Logger, m *Metrics) *Server {
	s := &Server{
		addr:      addr,
		tickRate:  tickRate,
		log:       log,
		metrics:   m,
		clients:   make(map[string]*Client),
		resources: make(map[string]protocol.Resource),
		done:      make(chan struct{}),
	}
	s.mammoths = make(map[string]*Mammoth)
	s.spawnResources(40)
	s.spawnMammoths(5)

	s.wells = make(map[string]*protocol.Well)
	s.spawnWells(6)

	s.houses = make(map[string]*House)
	s.solar = make(map[string]*Solar)
	s.batteries = make(map[string]*Battery)
	s.factories = make(map[string]*Factory)
	s.rockets = newStore[*Rocket]()
	s.boats = make(map[string]*Boat)
	s.mobs = make(map[string]*Mob)
	s.spawnMobs(5)
	s.projectiles = make(map[string]*Projectile)
	s.world = NewWorldState()
	return s
}

// Start запускает listener и два фоновых loop'а: accept + tick.
func (s *Server) Start() error {
	ln, err := net.Listen("tcp", s.addr)
	if err != nil {
		return fmt.Errorf("listen: %w", err)
	}
	s.ln = ln
	s.log.Info("tcp server listening", "addr", ln.Addr().String())
	s.wg.Add(2)
	go s.acceptLoop()
	go s.tickLoop()
	return nil
}

// Stop закрывает listener и все клиентские соединения, ждёт завершения loop'ов.
func (s *Server) Stop(ctx context.Context) error {
	s.closeOnce.Do(func() {
		close(s.done)
		if s.ln != nil {
			_ = s.ln.Close()
		}
		s.mu.Lock()
		for _, c := range s.clients {
			_ = c.conn.Close()
		}
		s.mu.Unlock()
	})
	waitCh := make(chan struct{})
	go func() { s.wg.Wait(); close(waitCh) }()
	select {
	case <-waitCh:
		return nil
	case <-ctx.Done():
		return ctx.Err()
	}
}

// Addr возвращает реальный адрес listener'а (полезно при addr=":0").
func (s *Server) Addr() string {
	if s.ln == nil {
		return ""
	}
	return s.ln.Addr().String()
}
