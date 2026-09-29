package net

import (
	"encoding/json"
	"log/slog"
	"net"
	"sync"
	"time"

	"github.com/spacedreamer99/lodeen/internal/shared/protocol"
)

// Этот файл содержит Client (одно подключение игрока) и его методы.

type Client struct {
	ID   string
	Nick string
	conn net.Conn
	send chan []byte
	done chan struct{}
	log  *slog.Logger

	mu          sync.RWMutex
	state       protocol.PlayerState
	inventory   map[string]int
	hunger      float32
	hp          int
	lastThrowAt time.Time
	lastHitAt   time.Time
	heldItem    string

	closeOnce sync.Once
}

func (c *Client) State() protocol.PlayerState {
	c.mu.RLock()
	defer c.mu.RUnlock()
	st := c.state
	st.HP = c.hp
	return st
}

func (c *Client) damage(n int) {
	c.mu.Lock()
	c.hp -= n
	if c.hp < 0 {
		c.hp = 0
	}
	dead := c.hp == 0
	c.mu.Unlock()
	if dead {
		c.log.Info("player died")
		c.sendEnvelope(protocol.TypeChat, protocol.ChatMessage{
			From: "server", Text: "you died", TS: time.Now().UnixMilli(),
		})
	}
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

func (c *Client) addHunger(delta float32) {
	c.mu.Lock()
	c.hunger += delta
	if c.hunger > 100 {
		c.hunger = 100
	}
	if c.hunger < 0 {
		c.hunger = 0
	}
	c.mu.Unlock()
}

func (c *Client) Hunger() float32 {
	c.mu.RLock()
	defer c.mu.RUnlock()
	return c.hunger
}

func (c *Client) consumeItem(item string) bool {
	c.mu.Lock()
	defer c.mu.Unlock()
	if c.inventory == nil {
		return false
	}
	if c.inventory[item] <= 0 {
		return false
	}
	c.inventory[item]--
	if c.inventory[item] == 0 {
		delete(c.inventory, item)
	}
	return true
}

func (c *Client) inventorySnapshot() map[string]int {
	c.mu.RLock()
	defer c.mu.RUnlock()
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
