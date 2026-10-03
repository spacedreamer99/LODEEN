package net

import (
	"crypto/rand"
	"encoding/binary"
	"math"
	"time"

	"github.com/spacedreamer99/lodeen/internal/shared/protocol"
)

// Mammoth struct + spawn + tick orchestrator.
// Handlers are in mammoths_handlers.go, breeding in mammoths_breeding.go,
// tick phases in mammoths_tick.go.

type Mammoth struct {
	ID            string
	Pos           protocol.Vector3
	HP            int
	Tamed         bool
	Sex           string
	FedCount      int
	Baby          bool
	LeashedTo     string
	Saddle        bool
	RiderID       string
	MateID        string
	TogetherSince time.Time
	BornAt        time.Time
}

const babyGrowTime = 60 * time.Second

func (s *Server) spawnMammoths(n int) {
	for i := 0; i < n; i++ {
		var b [16]byte
		_, _ = rand.Read(b[:])
		u := float64(binary.BigEndian.Uint64(b[0:8])) / float64(^uint64(0))
		v := float64(binary.BigEndian.Uint64(b[8:16])) / float64(^uint64(0))
		theta := 2 * math.Pi * u
		phi := math.Acos(2*v - 1)
		r := float64(protocol.PlanetRadius) + 2.0
		x := float32(math.Sin(phi) * math.Cos(theta) * r)
		y := float32(math.Cos(phi) * r)
		z := float32(math.Sin(phi) * math.Sin(theta) * r)
		id := newID()
		sex := "m"
		if i%2 == 1 {
			sex = "f"
		}
		s.mammoths[id] = &Mammoth{
			ID:  id,
			Pos: protocol.Vector3{X: x, Y: y, Z: z},
			HP:  1,
			Sex: sex,
		}
	}
	s.log.Info("spawned mammoths", "count", n)
}

// tickMammoths orchestrates all mammoth tick phases.
func (s *Server) tickMammoths(dt float32) {
	players := s.collectNearPlayers()

	s.mammothsMu.Lock()
	defer s.mammothsMu.Unlock()

	for _, m := range s.mammoths {
		switch {
		case m.RiderID != "":
			tickRiddenMammoth(m, players)
		case m.LeashedTo != "":
			tickLeashedMammoth(m, players, dt)
		case m.Tamed:
			// tamed mammoths stand still
		default:
			tickWildMammoth(m, players, dt)
		}
	}
}
