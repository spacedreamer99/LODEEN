package net

import (
	"math"
	"time"

	"github.com/spacedreamer99/lodeen/internal/shared/protocol"
)

// Этот файл содержит именованные handler'ы для типов сообщений,
// которым нужна нестандартная логика: state, ping, eat, select, exit.
// Всё остальное — в dispatch.go (реестр) и handlers_chat.go (чат).

func handleStateMsg(s *Server, c *Client, env *protocol.Envelope) error {
	var st protocol.PlayerState
	if err := env.Decode(&st); err != nil {
		return err
	}
	st.ID = c.ID
	st.Nick = c.Nick
	pos := protocol.ClampToSurface(protocol.Vector3{X: st.X, Y: st.Y, Z: st.Z})
	l := float32(math.Sqrt(float64(pos.X*pos.X + pos.Y*pos.Y + pos.Z*pos.Z)))
	if l > 0.01 {
		h := (l + protocol.PlayerHeight) / l
		pos.X *= h
		pos.Y *= h
		pos.Z *= h
	}
	st.X = pos.X
	st.Y = pos.Y
	st.Z = pos.Z
	c.setState(st)
	if st.RTTms > 0 {
		s.metrics.RTTSeconds.Observe(float64(st.RTTms) / 1000.0)
	}
	return nil
}

func handlePingMsg(_ *Server, c *Client, env *protocol.Envelope) error {
	var p protocol.Ping
	_ = env.Decode(&p)
	c.sendEnvelope(protocol.TypePong, protocol.Pong{
		Sent:     p.Sent,
		ServerTS: time.Now().UnixMilli(),
	})
	return nil
}

func handleEatFruitMsg(_ *Server, c *Client, _ *protocol.Envelope) error {
	if c.consumeItem("fruit") {
		c.addHunger(20)
		inv := c.inventorySnapshot()
		c.sendEnvelope(protocol.TypeInventoryUpdate, protocol.InventoryUpdate{Items: inv})
		c.log.Info("ate fruit", "hunger", c.Hunger())
	}
	return nil
}

func handleSelectItemMsg(_ *Server, c *Client, env *protocol.Envelope) error {
	var p protocol.SelectItem
	if err := env.Decode(&p); err != nil {
		return err
	}
	c.mu.Lock()
	c.heldItem = p.Item
	c.mu.Unlock()
	c.log.Info("select_item received", "item", p.Item)
	return nil
}

func handleExitRocketMsg(s *Server, c *Client, _ *protocol.Envelope) error {
	s.handleExitRocket(c)
	return nil
}

// handleCraft — крафт базовых рецептов с валидацией ресурсов.
func (s *Server) handleCraft(c *Client, recipe string) {
	c.mu.Lock()
	defer c.mu.Unlock()

	// Требования рецептов — на сервере, для валидации.
	type req map[string]int
	recipes := map[string]struct {
		need req
		have req
	}{
		"spear":  {need: req{"stone": 2, "wood": 1}, have: req{"spear": 1}},
		"leash":  {need: req{"liana": 2}, have: req{"leash": 1}},
		"house":  {need: req{"wood": 50}, have: req{"house": 1}},
		"saddle": {need: req{"liana": 4}, have: req{"saddle": 1}},
		"boat":   {need: req{"wood": 20}, have: req{"boat": 1}},
	}
	r, ok := recipes[recipe]
	if !ok {
		c.log.Warn("unknown recipe", "recipe", recipe)
		return
	}
	for k, n := range r.need {
		if c.inventory[k] < n {
			c.log.Info("craft failed: not enough materials", "recipe", recipe, "missing", k)
			return
		}
	}
	for k, n := range r.need {
		c.inventory[k] -= n
		if c.inventory[k] == 0 {
			delete(c.inventory, k)
		}
	}
	for k, n := range r.have {
		c.inventory[k] += n
	}

	out := make(map[string]int, len(c.inventory))
	for k, v := range c.inventory {
		out[k] = v
	}
	c.sendEnvelope(protocol.TypeInventoryUpdate, protocol.InventoryUpdate{Items: out})
	c.log.Info("crafted", "recipe", recipe)
}

// handlePickup — подбор ресурса с проверкой дистанции.
func (s *Server) handlePickup(c *Client, resourceID string) {
	ps := c.State()

	s.resources.Lock()
	r, ok := s.resources.Map()[resourceID]
	if !ok || r.Type == "seed" {
		s.resources.Unlock()
		return
	}
	dx := float64(r.X - ps.X)
	dy := float64(r.Y - ps.Y)
	dz := float64(r.Z - ps.Z)
	if dx*dx+dy*dy+dz*dz > pickupRadius*pickupRadius {
		s.resources.Unlock()
		return
	}
	delete(s.resources.Map(), resourceID)
	s.resources.Unlock()

	inv := c.addItem(r.Type)
	c.sendEnvelope(protocol.TypeInventoryUpdate, protocol.InventoryUpdate{Items: inv})
	c.log.Info("item picked up", "item", r.Type)
}
