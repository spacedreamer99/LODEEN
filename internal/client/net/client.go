package net

import (
	"encoding/json"
	"errors"
	"log/slog"
	"net"
	"sync"
	"time"

	"github.com/spacedreamer99/lodeen/internal/shared/protocol"
)

const (
	StatusDisconnected Status = iota
	StatusConnecting
	StatusConnected
)

const (
	interpolationDelay = 75 * time.Millisecond
	maxHistoryStates   = 20
)

type timedState struct {
	state protocol.PlayerState
	at    time.Time
}

type Client struct {
	log *slog.Logger

	mu       sync.RWMutex
	conn     net.Conn
	status   Status
	nick     string
	playerID string
	rtt      time.Duration

	playersMu sync.RWMutex
	players   map[string][]timedState

	ownHungerMu sync.RWMutex
	ownHunger   float32
	ownHP       int

	chatCh     chan protocol.ChatMessage
	teleportCh chan protocol.Teleport

	earthMu    sync.RWMutex
	earthPos   protocol.Vector3
	earthVel   protocol.Vector3
	planet2Pos protocol.Vector3
	planet2Vel protocol.Vector3
	lastTick   uint64

	resourcesMu sync.RWMutex
	resources   []protocol.Resource

	mammothsMu sync.RWMutex
	mammoths   []protocol.Mammoth

	wellsMu sync.RWMutex
	wells   []protocol.Well

	housesMu sync.RWMutex
	houses   []protocol.House

	boatsMu sync.RWMutex
	boats   []protocol.Boat

	mobsMu sync.RWMutex
	mobs   []protocol.Mob

	projectilesMu sync.RWMutex
	projectiles   []protocol.MobProjectile

	inventoryMu sync.RWMutex
	inventory   map[string]int

	stateMu   sync.Mutex
	lastState protocol.PlayerState

	writeMu sync.Mutex

	done chan struct{}
	wg   sync.WaitGroup

	solarMu     sync.RWMutex
	solar       []protocol.Solar
	batteriesMu sync.RWMutex
	batteries   []protocol.Battery
	factoriesMu sync.RWMutex
	factories   []protocol.Factory

	rocketsMu sync.RWMutex
	rockets   []protocol.Rocket
}

func New(log *slog.Logger) *Client {
	return &Client{
		log:        log,
		players:    make(map[string][]timedState),
		chatCh:     make(chan protocol.ChatMessage, 128),
		teleportCh: make(chan protocol.Teleport, 4),
		inventory:  make(map[string]int),
	}
}

// Close завершает соединение с сервером.
func (c *Client) Close() {
	if c.conn != nil {
		_ = c.conn.Close()
	}
}

func (c *Client) Connect(addr, nick, colorHex string) (*protocol.Welcome, error) {
	c.mu.Lock()
	if c.status == StatusConnected || c.status == StatusConnecting {
		c.mu.Unlock()
		return nil, errors.New("already connected")
	}
	c.status = StatusConnecting
	c.nick = nick
	c.mu.Unlock()

	conn, err := net.DialTimeout("tcp", addr, 5*time.Second)
	if err != nil {
		c.mu.Lock()
		c.status = StatusDisconnected
		c.mu.Unlock()
		return nil, err
	}

	env, err := protocol.NewEnvelope(protocol.TypeHello, protocol.Hello{Nick: nick, Version: "dev", ColorHex: colorHex})
	if err != nil {
		_ = conn.Close()
		return nil, err
	}
	raw, err := json.Marshal(env)
	if err != nil {
		_ = conn.Close()
		return nil, err
	}
	c.writeMu.Lock()
	_ = conn.SetWriteDeadline(time.Now().Add(5 * time.Second))
	err = protocol.WriteFrame(conn, raw)
	c.writeMu.Unlock()
	if err != nil {
		_ = conn.Close()
		return nil, err
	}

	_ = conn.SetReadDeadline(time.Now().Add(5 * time.Second))
	payload, err := protocol.ReadFrame(conn)
	if err != nil {
		_ = conn.Close()
		return nil, err
	}
	_ = conn.SetReadDeadline(time.Time{})
	var welcomeEnv protocol.Envelope
	if err := json.Unmarshal(payload, &welcomeEnv); err != nil {
		_ = conn.Close()
		return nil, err
	}
	if welcomeEnv.Type != protocol.TypeWelcome {
		_ = conn.Close()
		return nil, errors.New("expected welcome, got " + string(welcomeEnv.Type))
	}
	var welcome protocol.Welcome
	if err := welcomeEnv.Decode(&welcome); err != nil {
		_ = conn.Close()
		return nil, err
	}

	c.mu.Lock()
	c.conn = conn
	c.status = StatusConnected
	c.playerID = welcome.PlayerID
	c.done = make(chan struct{})
	c.mu.Unlock()

	c.wg.Add(2)
	go c.readLoop()
	go c.pingLoop()

	c.log.Info("connected", "addr", addr, "player_id", welcome.PlayerID)
	return &welcome, nil
}

func (c *Client) Disconnect() {
	c.mu.Lock()
	done := c.done
	if c.conn != nil {
		_ = c.conn.Close()
	}
	c.status = StatusDisconnected
	c.conn = nil
	c.mu.Unlock()

	if done != nil {
		select {
		case <-done:
		default:
			close(done)
		}
	}
	c.wg.Wait()

	c.playersMu.Lock()
	c.players = make(map[string][]timedState)
	c.playersMu.Unlock()
}

func (c *Client) InterpolatedSnapshot() []protocol.PlayerState {
	renderTime := time.Now().Add(-interpolationDelay)

	c.playersMu.RLock()
	defer c.playersMu.RUnlock()

	out := make([]protocol.PlayerState, 0, len(c.players))
	for _, hist := range c.players {
		if len(hist) == 0 {
			continue
		}
		if len(hist) == 1 {
			out = append(out, hist[0].state)
			continue
		}

		var a, b *timedState
		for i := len(hist) - 1; i >= 0; i-- {
			if !hist[i].at.After(renderTime) {
				a = &hist[i]
				if i+1 < len(hist) {
					b = &hist[i+1]
				}
				break
			}
		}
		if a == nil {
			out = append(out, hist[0].state)
			continue
		}
		if b == nil {
			out = append(out, a.state)
			continue
		}
		span := b.at.Sub(a.at).Seconds()
		if span <= 0 {
			out = append(out, a.state)
			continue
		}
		t := renderTime.Sub(a.at).Seconds() / span
		if t < 0 {
			t = 0
		} else if t > 1 {
			t = 1
		}
		// Копируем ВСЁ из последнего состояния (ColorHex, HP, Hunger,
		// Nick, RTTms и любые будущие поля), затем интерполируем позицию.
		interp := a.state
		interp.X = a.state.X + (b.state.X-a.state.X)*float32(t)
		interp.Y = a.state.Y + (b.state.Y-a.state.Y)*float32(t)
		interp.Z = a.state.Z + (b.state.Z-a.state.Z)*float32(t)
		out = append(out, interp)
	}
	return out
}

func (c *Client) send(t protocol.Type, data any) error {
	c.mu.RLock()
	conn := c.conn
	c.mu.RUnlock()
	if conn == nil {
		return errors.New("not connected")
	}
	env, err := protocol.NewEnvelope(t, data)
	if err != nil {
		return err
	}
	raw, err := json.Marshal(env)
	if err != nil {
		return err
	}
	c.writeMu.Lock()
	defer c.writeMu.Unlock()
	_ = conn.SetWriteDeadline(time.Now().Add(2 * time.Second))
	return protocol.WriteFrame(conn, raw)
}

func (c *Client) readLoop() {
	defer c.wg.Done()
	for {
		payload, err := protocol.ReadFrame(c.conn)
		if err != nil {
			c.log.Warn("read loop ended", "err", err)
			c.mu.Lock()
			c.status = StatusDisconnected
			c.mu.Unlock()
			select {
			case <-c.done:
			default:
				close(c.done)
			}
			return
		}
		var env protocol.Envelope
		if err := json.Unmarshal(payload, &env); err != nil {
			continue
		}
		c.handle(env)
	}
}

func (c *Client) pingLoop() {
	defer c.wg.Done()
	t := time.NewTicker(time.Second)
	defer t.Stop()
	for {
		select {
		case <-t.C:
			_ = c.send(protocol.TypePing, protocol.Ping{Sent: time.Now().UnixMilli()})
		case <-c.done:
			return
		}
	}
}
