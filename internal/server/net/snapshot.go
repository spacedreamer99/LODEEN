package net

import (
	"encoding/json"

	"github.com/spacedreamer99/lodeen/internal/shared/protocol"
)

// broadcastSnapshot — формирует снапшот мира и рассылает его всем клиентам.
// Вызывается ~20 раз в секунду из tick-цикла сервера.
func (s *Server) broadcastSnapshot() {
	env, err := protocol.NewEnvelope(protocol.TypeSnapshot, s.collectSnapshot())
	if err != nil {
		return
	}
	raw, err := json.Marshal(env)
	if err != nil {
		return
	}

	s.metrics.SnapshotBytes.Observe(float64(len(raw)))

	s.mu.RLock()
	defer s.mu.RUnlock()
	for _, c := range s.clients {
		c.enqueue(raw)
	}
}

// collectSnapshot — собирает снапшот из всех доменов мира.
// Каждый collectXxx делает собственный RLock и возвращает уже готовый срез.
func (s *Server) collectSnapshot() protocol.Snapshot {
	return protocol.Snapshot{
		Tick:        s.tick,
		Players:     s.collectPlayers(),
		Resources:   s.collectResources(),
		Mammoths:    s.collectMammoths(),
		Wells:       s.collectWells(),
		Houses:      s.collectHouses(),
		Boats:       s.collectBoats(),
		Mobs:        s.collectMobs(),
		Projectiles: s.collectProjectiles(),
		Solar:       s.collectSolar(),
		Batteries:   s.collectBatteries(),
		Factories:   s.collectFactories(),
		Rockets:     s.collectRockets(),
		EarthPos:    s.world.EarthPos,
		EarthVel:    s.world.EarthVel,
		Planet2Pos:  s.world.Planet2Pos,
		Planet2Vel:  s.world.Planet2Vel,
	}
}
