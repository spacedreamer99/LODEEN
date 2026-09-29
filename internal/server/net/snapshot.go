package net

import (
	"encoding/json"

	"github.com/spacedreamer99/lodeen/internal/shared/protocol"
)

// Этот файл содержит broadcastSnapshot — формирование и рассылку
// состояния мира всем клиентам 20 раз в секунду.

func (s *Server) broadcastSnapshot() {
	s.mu.RLock()
	players := make([]protocol.PlayerState, 0, len(s.clients))
	for _, c := range s.clients {
		ps := c.State()
		ps.Hunger = c.Hunger()
		players = append(players, ps)
	}
	s.mu.RUnlock()

	s.resourcesMu.RLock()
	resources := make([]protocol.Resource, 0, len(s.resources))
	for _, r := range s.resources {
		resources = append(resources, r)
	}
	s.resourcesMu.RUnlock()

	s.wellsMu.RLock()
	wells := make([]protocol.Well, 0, len(s.wells))
	for _, w := range s.wells {
		wells = append(wells, *w)
	}
	s.wellsMu.RUnlock()

	s.projMu.RLock()
	projs := make([]protocol.MobProjectile, 0, len(s.projectiles))
	for _, p := range s.projectiles {
		projs = append(projs, protocol.MobProjectile{
			ID: p.ID,
			X:  p.Pos.X,
			Y:  p.Pos.Y,
			Z:  p.Pos.Z,
			DX: p.Dir.X,
			DY: p.Dir.Y,
			DZ: p.Dir.Z,
		})
	}
	s.projMu.RUnlock()

	s.mobsMu.RLock()
	mobs := make([]protocol.Mob, 0, len(s.mobs))
	for _, m := range s.mobs {
		mobs = append(mobs, protocol.Mob{
			ID:       m.ID,
			X:        m.Pos.X,
			Y:        m.Pos.Y,
			Z:        m.Pos.Z,
			HP:       m.HP,
			Kind:     m.Kind,
			Contract: m.Contract,
			OwnerID:  m.OwnerID,
		})
	}
	s.mobsMu.RUnlock()

	s.boatsMu.RLock()
	boats := make([]protocol.Boat, 0, len(s.boats))
	for _, b := range s.boats {
		boats = append(boats, protocol.Boat{
			ID:      b.ID,
			X:       b.Pos.X,
			Y:       b.Pos.Y,
			Z:       b.Pos.Z,
			Yaw:     b.Yaw,
			RiderID: b.RiderID,
		})
	}
	s.boatsMu.RUnlock()

	s.housesMu.RLock()
	houses := make([]protocol.House, 0, len(s.houses))
	for _, h := range s.houses {
		houses = append(houses, protocol.House{
			ID:       h.ID,
			X:        h.Pos.X,
			Y:        h.Pos.Y,
			Z:        h.Pos.Z,
			Yaw:      h.Yaw,
			DoorOpen: h.DoorOpen,
		})
	}
	s.housesMu.RUnlock()

	s.solarMu.RLock()
	solars := make([]protocol.Solar, 0, len(s.solar))
	for _, sl := range s.solar {
		solars = append(solars, protocol.Solar{
			ID: sl.ID, X: sl.Pos.X, Y: sl.Pos.Y, Z: sl.Pos.Z, Yaw: sl.Yaw,
		})
	}
	s.solarMu.RUnlock()

	s.batteriesMu.RLock()
	batteries := make([]protocol.Battery, 0, len(s.batteries))
	for _, b := range s.batteries {
		batteries = append(batteries, protocol.Battery{
			ID: b.ID, X: b.Pos.X, Y: b.Pos.Y, Z: b.Pos.Z, Yaw: b.Yaw,
			Energy: b.Energy, MaxEnergy: b.MaxEnergy,
		})
	}
	s.batteriesMu.RUnlock()

	s.factoriesMu.RLock()
	factories := make([]protocol.Factory, 0, len(s.factories))
	for _, f := range s.factories {
		factories = append(factories, protocol.Factory{
			ID: f.ID, X: f.Pos.X, Y: f.Pos.Y, Z: f.Pos.Z, Yaw: f.Yaw,
			Crafting: f.Crafting, Progress: f.Progress,
		})
	}
	s.factoriesMu.RUnlock()

	s.rocketsMu.RLock()
	rockets := make([]protocol.Rocket, 0, len(s.rockets))
	for _, r := range s.rockets {
		rockets = append(rockets, protocol.Rocket{
			ID:      r.ID,
			X:       r.Pos.X,
			Y:       r.Pos.Y,
			Z:       r.Pos.Z,
			DX:      r.Up.X,
			DY:      r.Up.Y,
			DZ:      r.Up.Z,
			Fuel:    r.Fuel,
			MaxFuel: r.MaxFuel,
			Piloted: r.Piloted,
			OwnerID: r.OwnerID,
			InOrbit: r.InOrbit,
		})
	}
	s.rocketsMu.RUnlock()

	s.mammothsMu.RLock()
	mammoths := make([]protocol.Mammoth, 0, len(s.mammoths))
	for _, m := range s.mammoths {
		mammoths = append(mammoths, protocol.Mammoth{
			ID:        m.ID,
			X:         m.Pos.X,
			Y:         m.Pos.Y,
			Z:         m.Pos.Z,
			HP:        m.HP,
			Tamed:     m.Tamed,
			Sex:       m.Sex,
			FedCount:  m.FedCount,
			Baby:      m.Baby,
			LeashedTo: m.LeashedTo,
			Saddle:    m.Saddle,
			RiderID:   m.RiderID,
		})
	}
	s.mammothsMu.RUnlock()

	env, err := protocol.NewEnvelope(protocol.TypeSnapshot, protocol.Snapshot{
		Tick:        s.tick,
		Players:     players,
		Resources:   resources,
		Mammoths:    mammoths,
		Wells:       wells,
		Houses:      houses,
		Boats:       boats,
		Mobs:        mobs,
		Projectiles: projs,
		Solar:       solars,
		Batteries:   batteries,
		Factories:   factories,
		Rockets:     rockets,
	})
	if err != nil {
		return
	}
	raw, err := json.Marshal(env)
	if err != nil {
		return
	}

	s.mu.RLock()
	defer s.mu.RUnlock()
	for _, c := range s.clients {
		c.enqueue(raw)
	}
}

