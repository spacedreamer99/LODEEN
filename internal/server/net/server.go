// Package net implements the LODEEN TCP game server.
package net

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io"
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
}

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

	rocketsMu sync.RWMutex
	rockets   map[string]*Rocket

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
	s.rockets = make(map[string]*Rocket)
	s.boats = make(map[string]*Boat)
	s.mobs = make(map[string]*Mob)
	s.spawnMobs(5)
	s.projectiles = make(map[string]*Projectile)
	s.world = NewWorldState()
	return s
}

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

func (s *Server) Addr() string {
	if s.ln == nil {
		return ""
	}
	return s.ln.Addr().String()
}

func (s *Server) acceptLoop() {
	defer s.wg.Done()
	for {
		conn, err := s.ln.Accept()
		if err != nil {
			select {
			case <-s.done:
				return
			default:
				s.log.Warn("accept error", "err", err)
				continue
			}
		}
		s.wg.Add(1)
		go s.handleConn(conn)
	}
}

func (s *Server) handleConn(conn net.Conn) {
	defer s.wg.Done()

	id := newID()
	log := s.log.With("id", id, "remote", conn.RemoteAddr().String())

	client := &Client{
		ID:        id,
		conn:      conn,
		send:      make(chan []byte, sendBufSize),
		done:      make(chan struct{}),
		log:       log,
		inventory: make(map[string]int),
		hunger:    100,
		hp:        100,
	}

	_ = conn.SetReadDeadline(time.Now().Add(10 * time.Second))
	env, err := readEnvelope(conn)
	if err != nil {
		log.Warn("hello read failed", "err", err)
		_ = conn.Close()
		return
	}
	if env.Type != protocol.TypeHello {
		log.Warn("expected hello", "got", string(env.Type))
		_ = conn.Close()
		return
	}
	var hello protocol.Hello
	if err := env.Decode(&hello); err != nil {
		log.Warn("hello decode failed", "err", err)
		_ = conn.Close()
		return
	}
	if hello.Nick == "" {
		hello.Nick = "anon-" + id[:6]
	}
	client.Nick = hello.Nick
	client.setState(protocol.PlayerState{ID: id, Nick: hello.Nick})
	_ = conn.SetReadDeadline(time.Time{})

	client.sendEnvelope(protocol.TypeWelcome, protocol.Welcome{
		PlayerID:      id,
		WorldName:     "lodeen-flat",
		TickRate:      s.tickRate,
		ServerVersion: "dev",
	})
	client.sendEnvelope(protocol.TypeInventoryUpdate, protocol.InventoryUpdate{
		Items: map[string]int{},
	})

	s.mu.Lock()
	s.clients[id] = client
	s.mu.Unlock()
	s.metrics.PlayersConnected.Inc()
	log.Info("player joined", "nick", client.Nick)

	writerDone := make(chan struct{})
	go func() { defer close(writerDone); s.writeLoop(client) }()

	s.readLoop(client)

	client.closeOnce.Do(func() { close(client.done) })
	s.mu.Lock()
	delete(s.clients, id)
	s.mu.Unlock()
	s.metrics.PlayersConnected.Dec()

	// Освободить ракеты, лодки, мамонтов этого игрока.
	s.rocketsMu.Lock()
	for _, r := range s.rockets {
		if r.OwnerID == id {
			r.Piloted = false
			r.OwnerID = ""
			r.Thrust = 0
		}
	}
	s.rocketsMu.Unlock()
	_ = conn.Close()
	<-writerDone
	log.Info("player left", "nick", client.Nick)
}

func (s *Server) readLoop(c *Client) {
	for {
		_ = c.conn.SetReadDeadline(time.Now().Add(readTimeout))
		env, err := readEnvelope(c.conn)
		if err != nil {
			return
		}
		s.handleMessage(c, env)
	}
}

func (s *Server) writeLoop(c *Client) {
	for {
		select {
		case payload := <-c.send:
			_ = c.conn.SetWriteDeadline(time.Now().Add(writeTimeout))
			if err := protocol.WriteFrame(c.conn, payload); err != nil {
				return
			}
		case <-c.done:
			return
		case <-s.done:
			return
		}
	}
}

func (s *Server) broadcast(t protocol.Type, data any) {
	env, err := protocol.NewEnvelope(t, data)
	if err != nil {
		return
	}
	raw, err := json.Marshal(env)
	if err != nil {
		return
	}
	s.mu.RLock()
	defer s.mu.RUnlock()
	for _, c := range s.clients {
		c.enqueue(raw)
	}
}

func (s *Server) tickLoop() {
	defer s.wg.Done()
	interval := time.Second / time.Duration(s.tickRate)
	t := time.NewTicker(interval)
	defer t.Stop()
	for {
		select {
		case <-t.C:
			s.tick++
			s.metrics.TicksTotal.Inc()
			s.tickHunger(1.0 / float64(s.tickRate))
			s.tickMammoths(float32(1.0 / float64(s.tickRate)))
			s.tickBoats()
			s.tickMobs(float32(1.0 / float64(s.tickRate)))
			s.tickProjectiles(float32(1.0 / float64(s.tickRate)))
			s.tickBreeding()
			s.tickResources()
			s.tickEnergy(float32(1.0 / float64(s.tickRate)))
			s.world.Tick(float32(1.0 / float64(s.tickRate)))
			s.tickRockets(float32(1.0 / float64(s.tickRate)))
			s.broadcastSnapshot()
		case <-s.done:
			return
		}
	}
}

func (s *Server) tickHunger(dt float64) {
	const hungerRate = 0.333
	const autoEatThreshold = 50.0
	const fruitValue = 20.0

	dec := float32(hungerRate * dt)

	s.mu.RLock()
	clients := make([]*Client, 0, len(s.clients))
	for _, c := range s.clients {
		clients = append(clients, c)
	}
	s.mu.RUnlock()

	for _, c := range clients {
		c.addHunger(-dec)

		// Автосъедание: fruit (20), затем meat (30)
		for c.Hunger() < autoEatThreshold {
			eaten := ""
			if c.consumeItem("fruit") {
				c.addHunger(20)
				eaten = "fruit"
			} else if c.consumeItem("meat") {
				c.addHunger(30)
				eaten = "meat"
			}
			if eaten == "" {
				break
			}
			inv := c.inventorySnapshot()
			c.sendEnvelope(protocol.TypeInventoryUpdate, protocol.InventoryUpdate{Items: inv})
			c.log.Info("auto-ate", "item", eaten, "hunger", c.Hunger())
		}
	}
}

func readEnvelope(r io.Reader) (*protocol.Envelope, error) {
	payload, err := protocol.ReadFrame(r)
	if err != nil {
		return nil, err
	}
	var env protocol.Envelope
	if err := json.Unmarshal(payload, &env); err != nil {
		return nil, err
	}
	return &env, nil
}

func newID() string {
	var b [8]byte
	_, _ = rand.Read(b[:])
	return hex.EncodeToString(b[:])
}
