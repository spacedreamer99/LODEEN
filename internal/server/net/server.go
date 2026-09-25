// Package net implements the LODEEN TCP game server.
package net

import (
	"context"
	"crypto/rand"
	"encoding/binary"
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

var resourceTypes = []string{"stone", "wood", "ore"}

type Metrics struct {
	PlayersConnected prometheus.Gauge
	TicksTotal       prometheus.Counter
	ChatMessages     prometheus.Counter
}

type Client struct {
	ID   string
	Nick string
	conn net.Conn
	send chan []byte
	done chan struct{}
	log  *slog.Logger

	mu        sync.RWMutex
	state     protocol.PlayerState
	inventory map[string]int
}

func (c *Client) State() protocol.PlayerState {
	c.mu.RLock()
	defer c.mu.RUnlock()
	return c.state
}

func (c *Client) setState(s protocol.PlayerState) {
	c.mu.Lock()
	c.state = s
	c.mu.Unlock()
}

func (c *Client) addItem(item string) map[string]int {
	c.mu.Lock()
	defer c.mu.Unlock()
	if c.inventory == nil {
		c.inventory = make(map[string]int)
	}
	c.inventory[item]++
	out := make(map[string]int, len(c.inventory))
	for k, v := range c.inventory {
		out[k] = v
	}
	return out
}

func (c *Client) enqueue(payload []byte) {
	select {
	case c.send <- payload:
	case <-c.done:
	default:
	}
}

func (c *Client) sendEnvelope(t protocol.Type, data any) {
	env, err := protocol.NewEnvelope(t, data)
	if err != nil {
		return
	}
	raw, err := json.Marshal(env)
	if err != nil {
		return
	}
	c.enqueue(raw)
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

	tick uint64

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
	s.spawnResources(30)
	return s
}

func (s *Server) spawnResources(n int) {
	for i := 0; i < n; i++ {
		r := protocol.Resource{
			ID:   newID(),
			Type: resourceTypes[i%len(resourceTypes)],
			X:    randomRange(-50, 50),
			Y:    30,
			Z:    randomRange(10, 110),
		}
		s.resources[r.ID] = r
	}
	s.log.Info("spawned resources", "count", n)
}

func randomRange(min, max float32) float32 {
	var b [4]byte
	_, _ = rand.Read(b[:])
	n := binary.BigEndian.Uint32(b[:])
	return min + (float32(n%100000)/100000.0)*(max-min)
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

	close(client.done)
	s.mu.Lock()
	delete(s.clients, id)
	s.mu.Unlock()
	s.metrics.PlayersConnected.Dec()
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

func (s *Server) handleMessage(c *Client, env *protocol.Envelope) {
	switch env.Type {
	case protocol.TypeState:
		var st protocol.PlayerState
		if err := env.Decode(&st); err != nil {
			return
		}
		st.ID = c.ID
		st.Nick = c.Nick
		c.setState(st)
	case protocol.TypeChat:
		var cm protocol.ChatMessage
		if err := env.Decode(&cm); err != nil {
			return
		}
		cm.From = c.Nick
		cm.TS = time.Now().UnixMilli()
		s.broadcast(protocol.TypeChat, cm)
		s.metrics.ChatMessages.Inc()
	case protocol.TypePing:
		var p protocol.Ping
		_ = env.Decode(&p)
		c.sendEnvelope(protocol.TypePong, protocol.Pong{
			Sent:     p.Sent,
			ServerTS: time.Now().UnixMilli(),
		})
	case protocol.TypePickupItem:
		var p protocol.PickupItem
		if err := env.Decode(&p); err != nil {
			return
		}
		s.handlePickup(c, p.ResourceID)
	}
}

func (s *Server) handlePickup(c *Client, resourceID string) {
	ps := c.State()

	s.resourcesMu.Lock()
	r, ok := s.resources[resourceID]
	if !ok {
		s.resourcesMu.Unlock()
		return
	}
	dx := float64(r.X - ps.X)
	dy := float64(r.Y - ps.Y)
	dz := float64(r.Z - ps.Z)
	if dx*dx+dy*dy+dz*dz > pickupRadius*pickupRadius {
		s.resourcesMu.Unlock()
		return
	}
	delete(s.resources, resourceID)
	s.resourcesMu.Unlock()

	inv := c.addItem(r.Type)
	c.sendEnvelope(protocol.TypeInventoryUpdate, protocol.InventoryUpdate{Items: inv})
	c.log.Info("item picked up", "item", r.Type)
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
			s.broadcastSnapshot()
		case <-s.done:
			return
		}
	}
}

func (s *Server) broadcastSnapshot() {
	s.mu.RLock()
	players := make([]protocol.PlayerState, 0, len(s.clients))
	for _, c := range s.clients {
		players = append(players, c.State())
	}
	s.mu.RUnlock()

	s.resourcesMu.RLock()
	resources := make([]protocol.Resource, 0, len(s.resources))
	for _, r := range s.resources {
		resources = append(resources, r)
	}
	s.resourcesMu.RUnlock()

	env, err := protocol.NewEnvelope(protocol.TypeSnapshot, protocol.Snapshot{
		Tick:      s.tick,
		Players:   players,
		Resources: resources,
	})
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
