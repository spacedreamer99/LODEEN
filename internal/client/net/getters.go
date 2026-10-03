package net

import (
	"time"

	"github.com/spacedreamer99/lodeen/internal/shared/protocol"
)

type Status int

func (c *Client) EarthPos() protocol.Vector3 {
	c.earthMu.RLock()
	defer c.earthMu.RUnlock()
	return c.earthPos
}

func (c *Client) LastSnapshotTick() uint64 {
	c.earthMu.RLock()
	defer c.earthMu.RUnlock()
	return c.lastTick
}

func (c *Client) Planet2Pos() protocol.Vector3 {
	c.earthMu.RLock()
	defer c.earthMu.RUnlock()
	return c.planet2Pos
}

func (c *Client) Planet2Vel() protocol.Vector3 {
	c.earthMu.RLock()
	defer c.earthMu.RUnlock()
	return c.planet2Vel
}

func (c *Client) EarthVel() protocol.Vector3 {
	c.earthMu.RLock()
	defer c.earthMu.RUnlock()
	return c.earthVel
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

func (c *Client) Teleport() <-chan protocol.Teleport { return c.teleportCh }

func (c *Client) Hunger() float32 {
	c.ownHungerMu.RLock()
	defer c.ownHungerMu.RUnlock()
	return c.ownHunger
}

func (c *Client) OwnHP() int {
	c.ownHungerMu.RLock()
	defer c.ownHungerMu.RUnlock()
	return c.ownHP
}

func (c *Client) Houses() []protocol.House {
	c.housesMu.RLock()
	defer c.housesMu.RUnlock()
	out := make([]protocol.House, len(c.houses))
	copy(out, c.houses)
	return out
}

func (c *Client) Wells() []protocol.Well {
	c.wellsMu.RLock()
	defer c.wellsMu.RUnlock()
	out := make([]protocol.Well, len(c.wells))
	copy(out, c.wells)
	return out
}

func (c *Client) Mammoths() []protocol.Mammoth {
	c.mammothsMu.RLock()
	defer c.mammothsMu.RUnlock()
	out := make([]protocol.Mammoth, len(c.mammoths))
	copy(out, c.mammoths)
	return out
}

func (c *Client) Resources() []protocol.Resource {
	c.resourcesMu.RLock()
	defer c.resourcesMu.RUnlock()
	out := make([]protocol.Resource, len(c.resources))
	copy(out, c.resources)
	return out
}

func (c *Client) Inventory() map[string]int {
	c.inventoryMu.RLock()
	defer c.inventoryMu.RUnlock()
	out := make(map[string]int, len(c.inventory))
	for k, v := range c.inventory {
		out[k] = v
	}
	return out
}

func (c *Client) HasInventory() bool {
	c.inventoryMu.RLock()
	defer c.inventoryMu.RUnlock()
	return c.inventory != nil
}

func (c *Client) Solar() []protocol.Solar {
	c.solarMu.RLock()
	defer c.solarMu.RUnlock()
	out := make([]protocol.Solar, len(c.solar))
	copy(out, c.solar)
	return out
}

func (c *Client) Batteries() []protocol.Battery {
	c.batteriesMu.RLock()
	defer c.batteriesMu.RUnlock()
	out := make([]protocol.Battery, len(c.batteries))
	copy(out, c.batteries)
	return out
}

func (c *Client) Rockets() []protocol.Rocket {
	c.rocketsMu.RLock()
	defer c.rocketsMu.RUnlock()
	out := make([]protocol.Rocket, len(c.rockets))
	copy(out, c.rockets)
	return out
}

func (c *Client) Factories() []protocol.Factory {
	c.factoriesMu.RLock()
	defer c.factoriesMu.RUnlock()
	out := make([]protocol.Factory, len(c.factories))
	copy(out, c.factories)
	return out
}

func (c *Client) Projectiles() []protocol.MobProjectile {
	c.projectilesMu.RLock()
	defer c.projectilesMu.RUnlock()
	out := make([]protocol.MobProjectile, len(c.projectiles))
	copy(out, c.projectiles)
	return out
}

func (c *Client) Mobs() []protocol.Mob {
	c.mobsMu.RLock()
	defer c.mobsMu.RUnlock()
	out := make([]protocol.Mob, len(c.mobs))
	copy(out, c.mobs)
	return out
}

func (c *Client) Boats() []protocol.Boat {
	c.boatsMu.RLock()
	defer c.boatsMu.RUnlock()
	out := make([]protocol.Boat, len(c.boats))
	copy(out, c.boats)
	return out
}
