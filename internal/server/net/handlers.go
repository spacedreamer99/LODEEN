package net

import (
	"math"
	"time"

	"github.com/spacedreamer99/lodeen/internal/shared/protocol"
)

// Этот файл содержит входящие handler'ы: диспетчер сообщений handleMessage,
// handleChat (чит-команды), handlePickup, handleCraft.

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


func (s *Server) handleMessage(c *Client, env *protocol.Envelope) {
	switch env.Type {
	case protocol.TypeState:
		var st protocol.PlayerState
		if err := env.Decode(&st); err != nil {
			return
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
	case protocol.TypeChat:
		var cm protocol.ChatMessage
		if err := env.Decode(&cm); err != nil {
			return
		}
		// Чит-команда /allinv — выдать все предметы по 100.
		if cm.Text == "/allinv" {
			allItems := []string{
				"stone", "wood", "ore", "fruit", "meat", "spear", "torch",
				"water", "liana", "leash", "house", "saddle", "boat",
				"solar", "battery", "factory",
				"steel", "gear", "circuit", "drone",
			}
			c.mu.Lock()
			for _, it := range allItems {
				c.inventory[it] = 100
			}
			inv := make(map[string]int, len(c.inventory))
			for k, v := range c.inventory {
				inv[k] = v
			}
			c.mu.Unlock()
			c.sendEnvelope(protocol.TypeInventoryUpdate, protocol.InventoryUpdate{Items: inv})
			c.log.Info("cheat: allinv", "count", len(allItems))
			break
		}
		// Чит-команда /clearinv — очистить инвентарь.
		if cm.Text == "/clearinv" {
			c.mu.Lock()
			c.inventory = make(map[string]int)
			inv := make(map[string]int)
			c.mu.Unlock()
			c.sendEnvelope(protocol.TypeInventoryUpdate, protocol.InventoryUpdate{Items: inv})
			c.log.Info("cheat: clearinv")
			break
		}
		// Чит-команды вида /get<item><qty>, например /getfruit100.
		if len(cm.Text) > 4 && cm.Text[:4] == "/get" {
			rest := cm.Text[4:]
			// Отделяем хвостовые цифры (количество).
			i := len(rest)
			for i > 0 && rest[i-1] >= '0' && rest[i-1] <= '9' {
				i--
			}
			name := rest[:i]
			qtyStr := rest[i:]
			if name == "" || qtyStr == "" {
				c.sendEnvelope(protocol.TypeChat, protocol.ChatMessage{
					From: "server", Text: "usage: /get<item><qty>", TS: time.Now().UnixMilli(),
				})
				break
			}
			qty := 0
			for _, ch := range qtyStr {
				qty = qty*10 + int(ch-'0')
			}
			if qty > 100000 {
				qty = 100000
			}
			c.mu.Lock()
			c.inventory[name] += qty
			inv := make(map[string]int, len(c.inventory))
			for k, v := range c.inventory {
				inv[k] = v
			}
			c.mu.Unlock()
			c.sendEnvelope(protocol.TypeInventoryUpdate, protocol.InventoryUpdate{Items: inv})
			c.log.Info("cheat: get", "item", name, "qty", qty)
			break
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
	case protocol.TypeEatFruit:
		if c.consumeItem("fruit") {
			c.addHunger(20)
			inv := c.inventorySnapshot()
			c.sendEnvelope(protocol.TypeInventoryUpdate, protocol.InventoryUpdate{Items: inv})
			c.log.Info("ate fruit", "hunger", c.Hunger())
		}
	case protocol.TypePickupItem:
		var p protocol.PickupItem
		if err := env.Decode(&p); err != nil {
			return
		}
		s.handlePickup(c, p.ResourceID)
	case protocol.TypeThrowSpear:
		var p protocol.ThrowSpear
		if err := env.Decode(&p); err != nil {
			return
		}
		s.handleThrowSpear(c, p.Dir)
	case protocol.TypeCraftItem:
		var p protocol.CraftItem
		if err := env.Decode(&p); err != nil {
			return
		}
		s.handleCraft(c, p.Recipe)
	case protocol.TypeHitMammoth:
		var p protocol.HitMammoth
		if err := env.Decode(&p); err != nil {
			return
		}
		s.handleHitMammoth(c, p.MammothID)
	case protocol.TypePlantSeed:
		var p protocol.PlantSeed
		if err := env.Decode(&p); err != nil {
			return
		}
		s.handlePlantSeed(c, p)
	case protocol.TypeWaterPlant:
		var p protocol.WaterPlant
		if err := env.Decode(&p); err != nil {
			return
		}
		s.handleWaterPlant(c, p.ResourceID)
	case protocol.TypeTakeWater:
		var p protocol.TakeWater
		if err := env.Decode(&p); err != nil {
			return
		}
		s.handleTakeWater(c, p.WellID)
	case protocol.TypeTameMammoth:
		var p protocol.TameMammoth
		if err := env.Decode(&p); err != nil {
			return
		}
		s.handleTameMammoth(c, p.MammothID)
	case protocol.TypeSelectItem:
		var p protocol.SelectItem
		if err := env.Decode(&p); err != nil {
			return
		}
		c.mu.Lock()
		c.heldItem = p.Item
		c.mu.Unlock()
		c.log.Info("select_item received", "item", p.Item)
	case protocol.TypeLeashMammoth:
		var p protocol.LeashMammoth
		if err := env.Decode(&p); err != nil {
			return
		}
		c.log.Info("leash packet received", "id", p.MammothID)
		s.handleLeashMammoth(c, p.MammothID)
	case protocol.TypePlaceHouse:
		var p protocol.PlaceHouse
		if err := env.Decode(&p); err != nil {
			return
		}
		s.handlePlaceHouse(c, p)
	case protocol.TypeToggleDoor:
		var p protocol.ToggleDoor
		if err := env.Decode(&p); err != nil {
			return
		}
		s.handleToggleDoor(c, p.HouseID)
	case protocol.TypeSaddleMammoth:
		var p protocol.SaddleMammoth
		if err := env.Decode(&p); err != nil {
			return
		}
		s.handleSaddleMammoth(c, p.MammothID)
	case protocol.TypeRideMammoth:
		var p protocol.RideMammoth
		if err := env.Decode(&p); err != nil {
			return
		}
		s.handleRideMammoth(c, p.MammothID)
	case protocol.TypePlaceBoat:
		var p protocol.PlaceBoat
		if err := env.Decode(&p); err != nil {
			return
		}
		s.handlePlaceBoat(c, p)
	case protocol.TypeEnterBoat:
		var p protocol.EnterBoat
		if err := env.Decode(&p); err != nil {
			return
		}
		s.handleEnterBoat(c, p.BoatID)
	case protocol.TypeHitMob:
		var p protocol.HitMob
		if err := env.Decode(&p); err != nil {
			return
		}
		s.handleHitMob(c, p.MobID)
	case protocol.TypeAcceptContract:
		var p protocol.AcceptContract
		if err := env.Decode(&p); err != nil {
			return
		}
		s.handleAcceptContract(c, p.MobID, p.ContractID)
	case protocol.TypePlaceSolar:
		var p protocol.PlaceSolar
		if err := env.Decode(&p); err != nil {
			return
		}
		s.handlePlaceSolar(c, p)
	case protocol.TypePlaceBattery:
		var p protocol.PlaceBattery
		if err := env.Decode(&p); err != nil {
			return
		}
		s.handlePlaceBattery(c, p)
	case protocol.TypePlaceFactory:
		var p protocol.PlaceFactory
		if err := env.Decode(&p); err != nil {
			return
		}
		s.handlePlaceFactory(c, p)
	case protocol.TypeOpenFactory:
		var p protocol.OpenFactory
		if err := env.Decode(&p); err != nil {
			return
		}
		s.handleOpenFactory(c, p.FactoryID)
	case protocol.TypeCraftFactory:
		var p protocol.CraftFactory
		if err := env.Decode(&p); err != nil {
			return
		}
		s.handleCraftFactory(c, p.FactoryID, p.Recipe)
	}
}


func (s *Server) handlePickup(c *Client, resourceID string) {
	ps := c.State()

	s.resourcesMu.Lock()
	r, ok := s.resources[resourceID]
	if !ok || r.Type == "seed" {
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

