package net

import (
	"time"

	"github.com/spacedreamer99/lodeen/internal/shared/protocol"
)

func (c *Client) handle(env protocol.Envelope) {
	switch env.Type {
	case protocol.TypeSnapshot:
		var s protocol.Snapshot
		_ = env.Decode(&s)
		now := time.Now()

		c.mu.RLock()
		selfID := c.playerID
		c.mu.RUnlock()

		c.playersMu.Lock()
		seen := make(map[string]struct{}, len(s.Players))
		for _, p := range s.Players {
			seen[p.ID] = struct{}{}
			hist := c.players[p.ID]
			hist = append(hist, timedState{state: p, at: now})
			if len(hist) > maxHistoryStates {
				hist = hist[len(hist)-maxHistoryStates:]
			}
			c.players[p.ID] = hist

			if p.ID == selfID {
				c.ownHungerMu.Lock()
				c.ownHunger = p.Hunger
				c.ownHP = p.HP
				c.ownHungerMu.Unlock()
			}
		}
		for id := range c.players {
			if _, ok := seen[id]; !ok {
				delete(c.players, id)
			}
		}
		c.playersMu.Unlock()
		c.resourcesMu.Lock()
		c.resources = make([]protocol.Resource, len(s.Resources))
		copy(c.resources, s.Resources)
		c.resourcesMu.Unlock()

		c.mammothsMu.Lock()
		c.mammoths = make([]protocol.Mammoth, len(s.Mammoths))
		copy(c.mammoths, s.Mammoths)
		c.mammothsMu.Unlock()

		c.wellsMu.Lock()
		c.wells = make([]protocol.Well, len(s.Wells))
		copy(c.wells, s.Wells)
		c.wellsMu.Unlock()

		c.housesMu.Lock()
		c.houses = make([]protocol.House, len(s.Houses))
		copy(c.houses, s.Houses)
		c.housesMu.Unlock()

		c.boatsMu.Lock()
		c.boats = make([]protocol.Boat, len(s.Boats))
		copy(c.boats, s.Boats)
		c.boatsMu.Unlock()

		c.mobsMu.Lock()
		c.mobs = make([]protocol.Mob, len(s.Mobs))
		copy(c.mobs, s.Mobs)
		c.mobsMu.Unlock()

		c.projectilesMu.Lock()
		c.projectiles = make([]protocol.MobProjectile, len(s.Projectiles))
		copy(c.projectiles, s.Projectiles)
		c.projectilesMu.Unlock()

		c.solarMu.Lock()
		c.solar = make([]protocol.Solar, len(s.Solar))
		copy(c.solar, s.Solar)
		c.solarMu.Unlock()

		c.batteriesMu.Lock()
		c.batteries = make([]protocol.Battery, len(s.Batteries))
		copy(c.batteries, s.Batteries)
		c.batteriesMu.Unlock()

		c.factoriesMu.Lock()
		c.factories = make([]protocol.Factory, len(s.Factories))
		copy(c.factories, s.Factories)
		c.factoriesMu.Unlock()

		c.earthMu.Lock()
		c.earthPos = s.EarthPos
		c.earthVel = s.EarthVel
		c.planet2Pos = s.Planet2Pos
		c.planet2Vel = s.Planet2Vel
		c.lastTick = s.Tick
		c.earthMu.Unlock()

		// Ракеты теперь в helio — без конверта в geo.
		c.rocketsMu.Lock()
		c.rockets = make([]protocol.Rocket, len(s.Rockets))
		copy(c.rockets, s.Rockets)
		c.rocketsMu.Unlock()
	case protocol.TypeChat:
		var cm protocol.ChatMessage
		_ = env.Decode(&cm)
		select {
		case c.chatCh <- cm:
		default:
		}
	case protocol.TypeInventoryUpdate:
		var inv protocol.InventoryUpdate
		_ = env.Decode(&inv)
		c.inventoryMu.Lock()
		c.inventory = inv.Items
		c.inventoryMu.Unlock()
	case protocol.TypeTeleport:
		var tp protocol.Teleport
		_ = env.Decode(&tp)
		select {
		case c.teleportCh <- tp:
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
