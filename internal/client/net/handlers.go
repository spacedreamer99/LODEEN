package net

import (
	"time"

	"github.com/spacedreamer99/lodeen/internal/shared/protocol"
)

// handle — диспетчер входящих сообщений от сервера.
// Каждый тип обрабатывается в отдельной функции.
func (c *Client) handle(env protocol.Envelope) {
	switch env.Type {
	case protocol.TypeSnapshot:
		c.handleSnapshot(env)
	case protocol.TypeChat:
		c.handleChat(env)
	case protocol.TypeInventoryUpdate:
		c.handleInventoryUpdate(env)
	case protocol.TypeTeleport:
		c.handleTeleport(env)
	case protocol.TypePong:
		c.handlePong(env)
	}
}

// handleSnapshot обновляет состояние мира по снапшоту 20 Гц:
// игроки (с историей для интерполяции), ресурсы, мобы, постройки, планеты.
func (c *Client) handleSnapshot(env protocol.Envelope) {
	var s protocol.Snapshot
	_ = env.Decode(&s)

	c.updatePlayers(s.Players)
	c.updateResources(s.Resources)
	c.updateMammoths(s.Mammoths)
	c.updateWells(s.Wells)
	c.updateHouses(s.Houses)
	c.updateBoats(s.Boats)
	c.updateMobs(s.Mobs)
	c.updateProjectiles(s.Projectiles)
	c.updateSolar(s.Solar)
	c.updateBatteries(s.Batteries)
	c.updateFactories(s.Factories)
	c.updateEarthAndPlanets(s)
	c.updateRockets(s.Rockets)
}

// --- Игроки (с историей для интерполяции) ---

func (c *Client) updatePlayers(players []protocol.PlayerState) {
	now := time.Now()

	c.mu.RLock()
	selfID := c.playerID
	c.mu.RUnlock()

	c.playersMu.Lock()
	defer c.playersMu.Unlock()

	seen := make(map[string]struct{}, len(players))
	for _, p := range players {
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
}

// --- Ресурсы и постройки ---

func (c *Client) updateResources(res []protocol.Resource) {
	c.resourcesMu.Lock()
	c.resources = make([]protocol.Resource, len(res))
	copy(c.resources, res)
	c.resourcesMu.Unlock()
}

func (c *Client) updateMammoths(ms []protocol.Mammoth) {
	c.mammothsMu.Lock()
	c.mammoths = make([]protocol.Mammoth, len(ms))
	copy(c.mammoths, ms)
	c.mammothsMu.Unlock()
}

func (c *Client) updateWells(wells []protocol.Well) {
	c.wellsMu.Lock()
	c.wells = make([]protocol.Well, len(wells))
	copy(c.wells, wells)
	c.wellsMu.Unlock()
}

func (c *Client) updateHouses(houses []protocol.House) {
	c.housesMu.Lock()
	c.houses = make([]protocol.House, len(houses))
	copy(c.houses, houses)
	c.housesMu.Unlock()
}

func (c *Client) updateBoats(boats []protocol.Boat) {
	c.boatsMu.Lock()
	c.boats = make([]protocol.Boat, len(boats))
	copy(c.boats, boats)
	c.boatsMu.Unlock()
}

func (c *Client) updateMobs(mobs []protocol.Mob) {
	c.mobsMu.Lock()
	c.mobs = make([]protocol.Mob, len(mobs))
	copy(c.mobs, mobs)
	c.mobsMu.Unlock()
}

func (c *Client) updateProjectiles(projs []protocol.MobProjectile) {
	c.projectilesMu.Lock()
	c.projectiles = make([]protocol.MobProjectile, len(projs))
	copy(c.projectiles, projs)
	c.projectilesMu.Unlock()
}

// --- Энергия ---

func (c *Client) updateSolar(panels []protocol.Solar) {
	c.solarMu.Lock()
	c.solar = make([]protocol.Solar, len(panels))
	copy(c.solar, panels)
	c.solarMu.Unlock()
}

func (c *Client) updateBatteries(bats []protocol.Battery) {
	c.batteriesMu.Lock()
	c.batteries = make([]protocol.Battery, len(bats))
	copy(c.batteries, bats)
	c.batteriesMu.Unlock()
}

func (c *Client) updateFactories(facts []protocol.Factory) {
	c.factoriesMu.Lock()
	c.factories = make([]protocol.Factory, len(facts))
	copy(c.factories, facts)
	c.factoriesMu.Unlock()
}

// --- Небесные тела ---

func (c *Client) updateEarthAndPlanets(s protocol.Snapshot) {
	c.earthMu.Lock()
	c.earthPos = s.EarthPos
	c.earthVel = s.EarthVel
	c.planet2Pos = s.Planet2Pos
	c.planet2Vel = s.Planet2Vel
	c.lastTick = s.Tick
	c.earthMu.Unlock()
}

// updateRockets — ракеты в helio, без конверта в geo.
func (c *Client) updateRockets(rockets []protocol.Rocket) {
	c.rocketsMu.Lock()
	c.rockets = make([]protocol.Rocket, len(rockets))
	copy(c.rockets, rockets)
	c.rocketsMu.Unlock()
}

// --- Остальные типы сообщений ---

func (c *Client) handleChat(env protocol.Envelope) {
	var cm protocol.ChatMessage
	_ = env.Decode(&cm)
	select {
	case c.chatCh <- cm:
	default:
	}
}

func (c *Client) handleInventoryUpdate(env protocol.Envelope) {
	var inv protocol.InventoryUpdate
	_ = env.Decode(&inv)
	c.inventoryMu.Lock()
	c.inventory = inv.Items
	c.inventoryMu.Unlock()
}

func (c *Client) handleTeleport(env protocol.Envelope) {
	var tp protocol.Teleport
	_ = env.Decode(&tp)
	select {
	case c.teleportCh <- tp:
	default:
	}
}

func (c *Client) handlePong(env protocol.Envelope) {
	var p protocol.Pong
	_ = env.Decode(&p)
	c.mu.Lock()
	c.rtt = time.Duration(time.Now().UnixMilli()-p.Sent) * time.Millisecond
	c.mu.Unlock()
}
