package net

import (
	"time"

	"github.com/spacedreamer99/lodeen/internal/shared/protocol"
)

// tickLoop — фиксированный dt-цикл сервера. Вызывает все доменные тики
// в фиксированном порядке и рассылает snapshot.
func (s *Server) tickLoop() {
	defer s.wg.Done()

	tickDur := time.Duration(float64(time.Second) / float64(s.tickRate))
	next := time.Now().Add(tickDur)
	lastTick := time.Now()

	for {
		select {
		case <-s.done:
			return
		default:
		}

		now := time.Now()
		if now.Before(next) {
			time.Sleep(next.Sub(now))
			continue
		}

		// Физика — ВСЕГДА fixed dt. Это критично: иначе снапшоты
		// идут неравномерно, и клиент интерполирует с дрожанием.
		realDt := now.Sub(lastTick).Seconds()
		lastTick = now
		_ = realDt

		dtF := float32(1.0) / float32(s.tickRate)
		dt := float64(dtF)

		s.tick++
		s.metrics.TicksTotal.Inc()

		tickStart := time.Now()

		s.tickHunger(dt)
		s.tickMammoths(dtF)
		s.tickBoats()
		s.tickMobs(dtF)
		s.tickProjectiles(dtF)
		s.tickBreeding()
		s.tickResources()
		s.tickEnergy(dtF)
		s.world.Tick(dtF)
		s.tickRockets(dtF)
		s.broadcastSnapshot()

		s.metrics.TickDuration.Observe(time.Since(tickStart).Seconds())

		next = next.Add(tickDur)
		if time.Since(next) > 200*time.Millisecond {
			next = time.Now().Add(tickDur)
		}
	}
}

// tickHunger уменьшает голод всех клиентов и автосъедает еду из инвентаря.
func (s *Server) tickHunger(dt float64) {
	const hungerRate = 0.333
	const autoEatThreshold = 50.0

	dec := float32(hungerRate * dt)

	s.mu.RLock()
	clients := make([]*Client, 0, len(s.clients))
	for _, c := range s.clients {
		clients = append(clients, c)
	}
	s.mu.RUnlock()

	for _, c := range clients {
		c.addHunger(-dec)

		for c.Hunger() < autoEatThreshold {
			eaten := ""
			if c.consumeItem("fruit") {
				c.addHunger(20)
				eaten = "fruit"
			} else if c.consumeItem("meat") {
				c.addHunger(30)
				eaten = "meat"
			}
			if eaten == "" {
				break
			}
			inv := c.inventorySnapshot()
			c.sendEnvelope(protocol.TypeInventoryUpdate, protocol.InventoryUpdate{Items: inv})
			c.log.Info("auto-ate", "item", eaten, "hunger", c.Hunger())
		}
	}
}
