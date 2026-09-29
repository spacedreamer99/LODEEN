package net

import (
	"time"

	"github.com/spacedreamer99/lodeen/internal/shared/protocol"
)

// Этот файл содержит Projectile и логику броска копья игроком.

type Projectile struct {
	ID         string
	Pos        protocol.Vector3
	Dir        protocol.Vector3
	Speed      float32
	OwnerMobID string
	TargetID   string
	SpawnAt    time.Time
}


func (s *Server) handleThrowSpear(c *Client, dir protocol.Vector3) {
	// Валидация: есть ли копьё.
	if !c.consumeItem("spear") {
		return
	}
	// Cooldown: не чаще раза в 0.8 сек.
	now := time.Now()
	if now.Sub(c.lastThrowAt) < 800*time.Millisecond {
		// отдаём копьё обратно
		c.mu.Lock()
		c.inventory["spear"]++
		inv := make(map[string]int, len(c.inventory))
		for k, v := range c.inventory {
			inv[k] = v
		}
		c.mu.Unlock()
		c.sendEnvelope(protocol.TypeInventoryUpdate, protocol.InventoryUpdate{Items: inv})
		return
	}
	c.lastThrowAt = now
	c.log.Info("spear thrown (client will detect hit)")

	c.mu.Lock()
	inv := make(map[string]int, len(c.inventory))
	for k, v := range c.inventory {
		inv[k] = v
	}
	c.mu.Unlock()
	c.sendEnvelope(protocol.TypeInventoryUpdate, protocol.InventoryUpdate{Items: inv})
}

