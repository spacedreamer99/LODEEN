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

type Status int

const (
	StatusDisconnected Status = iota
	StatusConnecting
	StatusConnected
)

type Client struct {
	log *slog.Logger

	mu       sync.RWMutex
	conn     net.Conn
	status   Status
	nick     string
	playerID string
	rtt      time.Duration

	playersMu sync.RWMutex
	players   map[string]protocol.PlayerState

	chatCh chan protocol.ChatMessage

	stateMu   sync.Mutex
	lastState protocol.PlayerState

	done chan struct{}
	wg   sync.WaitGroup
}

func New(log *slog.Logger) *Client {
	return &Client{
		log:     log,
		players: make(map[string]protocol.PlayerState),
		chatCh:  make(chan protocol.ChatMessage, 128),
	}
}

func (c *Client) Status() Status {
	c.mu.RLock()
	defer c.mu.RUnlock()
	return c.status
}

func (c *Client) PlayerID() string {
	c.mu.RLock()
	defer c.mu.RUnlock()
	return c.playerID
}

func (c *Client) RTT() time.Duration {
	c.mu.RLock()
	defer c.mu.RUnlock()
	return c.rtt
}

func (c *Client) Chat() <-chan protocol.ChatMessage { return c.chatCh }

func (c *Client) Connect(addr, nick string) (*protocol.Welcome, error) {
	c.mu.Lock()
	if c.status == StatusConnected {
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

	env, err := protocol.NewEnvelope(protocol.TypeHello, protocol.Hello{Nick: nick, Version: "dev"})
	if err != nil {
		_ = conn.Close()
		return nil, err
	}
	raw, err := json.Marshal(env)
	if err != nil {
		_ = conn.Close()
		return nil, err
	}
	_ = conn.SetWriteDeadline(time.Now().Add(5 * time.Second))
	if err := protocol.WriteFrame(conn, raw); err != nil {
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
}

func (c *Client) SetState(st protocol.PlayerState) {
	c.stateMu.Lock()
	c.lastState = st
	c.stateMu.Unlock()
}

func (c *Client) StartStateLoop() {
	c.wg.Add(1)
	go func() {
		defer c.wg.Done()
		t := time.NewTicker(time.Second / 20)
		defer t.Stop()
		for {
			select {
			case <-t.C:
				c.stateMu.Lock()
				st := c.lastState
				c.stateMu.Unlock()
				_ = c.send(protocol.TypeState, st)
			case <-c.done:
				return
			}
		}
	}()
}

func (c *Client) SendChat(text string) error {
	return c.send(protocol.TypeChat, protocol.ChatMessage{Text: text})
}

func (c *Client) Snapshot() []protocol.PlayerState {
	c.playersMu.RLock()
	defer c.playersMu.RUnlock()
	out := make([]protocol.PlayerState, 0, len(c.players))
	for _, p := range c.players {
		out = append(out, p)
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
			return
		}
		var env protocol.Envelope
		if err := json.Unmarshal(payload, &env); err != nil {
			continue
		}
		c.handle(env)
	}
}

func (c *Client) handle(env protocol.Envelope) {
	switch env.Type {
	case protocol.TypeSnapshot:
		var s protocol.Snapshot
		_ = env.Decode(&s)
		c.playersMu.Lock()
		c.players = make(map[string]protocol.PlayerState, len(s.Players))
		for _, p := range s.Players {
			c.players[p.ID] = p
		}
		c.playersMu.Unlock()
	case protocol.TypeChat:
		var cm protocol.ChatMessage
		_ = env.Decode(&cm)
		select {
		case c.chatCh <- cm:
		default:
		}
	case protocol.TypePong:
		var p protocol.Pong
		_ = env.Decode(&p)
		c.mu.Lock()
		c.rtt = time.Duration(time.Now().UnixMilli()-p.Sent) * time.Millisecond
		c.mu.Unlock()
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
