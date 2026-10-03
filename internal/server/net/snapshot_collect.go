package net

import (
	"github.com/spacedreamer99/lodeen/internal/shared/protocol"
)

// collectPlayers — срез состояний всех подключённых игроков + голод.
func (s *Server) collectPlayers() []protocol.PlayerState {
	s.mu.RLock()
	defer s.mu.RUnlock()
	out := make([]protocol.PlayerState, 0, len(s.clients))
	for _, c := range s.clients {
		ps := c.State()
		ps.Hunger = c.Hunger()
		out = append(out, ps)
	}
	return out
}

// collectResources — подобранные/разбросанные ресурсы на поверхности.
func (s *Server) collectResources() []protocol.Resource {
	s.resourcesMu.RLock()
	defer s.resourcesMu.RUnlock()
	out := make([]protocol.Resource, 0, len(s.resources))
	for _, r := range s.resources {
		out = append(out, r)
	}
	return out
}

// collectWells — колодцы (вода).
func (s *Server) collectWells() []protocol.Well {
	s.wellsMu.RLock()
	defer s.wellsMu.RUnlock()
	out := make([]protocol.Well, 0, len(s.wells))
	for _, w := range s.wells {
		out = append(out, *w)
	}
	return out
}

// collectProjectiles — летящие снаряды мобов.
func (s *Server) collectProjectiles() []protocol.MobProjectile {
	s.projMu.RLock()
	defer s.projMu.RUnlock()
	out := make([]protocol.MobProjectile, 0, len(s.projectiles))
	for _, p := range s.projectiles {
		out = append(out, protocol.MobProjectile{
			ID: p.ID,
			X:  p.Pos.X, Y: p.Pos.Y, Z: p.Pos.Z,
			DX: p.Dir.X, DY: p.Dir.Y, DZ: p.Dir.Z,
		})
	}
	return out
}

// collectMobs — хосты/мобы с их контрактами.
func (s *Server) collectMobs() []protocol.Mob {
	s.mobsMu.RLock()
	defer s.mobsMu.RUnlock()
	out := make([]protocol.Mob, 0, len(s.mobs))
	for _, m := range s.mobs {
		out = append(out, protocol.Mob{
			ID: m.ID,
			X:  m.Pos.X, Y: m.Pos.Y, Z: m.Pos.Z,
			HP:       m.HP,
			Kind:     m.Kind,
			Contract: m.Contract,
			OwnerID:  m.OwnerID,
		})
	}
	return out
}

// collectBoats — лодки с наездниками.
func (s *Server) collectBoats() []protocol.Boat {
	s.boatsMu.RLock()
	defer s.boatsMu.RUnlock()
	out := make([]protocol.Boat, 0, len(s.boats))
	for _, b := range s.boats {
		out = append(out, protocol.Boat{
			ID: b.ID,
			X:  b.Pos.X, Y: b.Pos.Y, Z: b.Pos.Z,
			Yaw:     b.Yaw,
			RiderID: b.RiderID,
		})
	}
	return out
}

// collectHouses — дома (state двери).
func (s *Server) collectHouses() []protocol.House {
	s.housesMu.RLock()
	defer s.housesMu.RUnlock()
	out := make([]protocol.House, 0, len(s.houses))
	for _, h := range s.houses {
		out = append(out, protocol.House{
			ID: h.ID,
			X:  h.Pos.X, Y: h.Pos.Y, Z: h.Pos.Z,
			Yaw:      h.Yaw,
			DoorOpen: h.DoorOpen,
		})
	}
	return out
}

// collectSolar — солнечные панели.
func (s *Server) collectSolar() []protocol.Solar {
	s.solarMu.RLock()
	defer s.solarMu.RUnlock()
	out := make([]protocol.Solar, 0, len(s.solar))
	for _, sl := range s.solar {
		out = append(out, protocol.Solar{
			ID: sl.ID, X: sl.Pos.X, Y: sl.Pos.Y, Z: sl.Pos.Z, Yaw: sl.Yaw,
		})
	}
	return out
}

// collectBatteries — батареи с уровнем энергии.
func (s *Server) collectBatteries() []protocol.Battery {
	s.batteriesMu.RLock()
	defer s.batteriesMu.RUnlock()
	out := make([]protocol.Battery, 0, len(s.batteries))
	for _, b := range s.batteries {
		out = append(out, protocol.Battery{
			ID: b.ID, X: b.Pos.X, Y: b.Pos.Y, Z: b.Pos.Z, Yaw: b.Yaw,
			Energy: b.Energy, MaxEnergy: b.MaxEnergy,
		})
	}
	return out
}

// collectFactories — фабрики с прогрессом крафта.
func (s *Server) collectFactories() []protocol.Factory {
	s.factoriesMu.RLock()
	defer s.factoriesMu.RUnlock()
	out := make([]protocol.Factory, 0, len(s.factories))
	for _, f := range s.factories {
		out = append(out, protocol.Factory{
			ID: f.ID, X: f.Pos.X, Y: f.Pos.Y, Z: f.Pos.Z, Yaw: f.Yaw,
			Crafting: f.Crafting, Progress: f.Progress,
		})
	}
	return out
}

// collectRockets — ракеты со всей орбитальной телеметрией.
func (s *Server) collectRockets() []protocol.Rocket {
	s.rocketsMu.RLock()
	defer s.rocketsMu.RUnlock()
	out := make([]protocol.Rocket, 0, len(s.rockets))
	for _, r := range s.rockets {
		out = append(out, protocol.Rocket{
			ID: r.ID,
			X:  r.Pos.X, Y: r.Pos.Y, Z: r.Pos.Z,
			DX: r.Up.X, DY: r.Up.Y, DZ: r.Up.Z,
			Fuel: r.Fuel, MaxFuel: r.MaxFuel,
			Piloted: r.Piloted, OwnerID: r.OwnerID,
			InOrbit:  r.InOrbit,
			Apoapsis: r.Apoapsis, Periapsis: r.Periapsis,
			Speed: r.Speed, Altitude: r.Altitude,
			TargetVelocity: r.TargetVelocity,
			VX:             r.Vel.X, VY: r.Vel.Y, VZ: r.Vel.Z,
			PrimaryBody: r.PrimaryBody,
			DistSun:     r.DistSun,
			SOIRadius:   r.SOIRadius,
		})
	}
	return out
}

// collectMammoths — мамонты (приручение, разведение, сёдла).
func (s *Server) collectMammoths() []protocol.Mammoth {
	s.mammothsMu.RLock()
	defer s.mammothsMu.RUnlock()
	out := make([]protocol.Mammoth, 0, len(s.mammoths))
	for _, m := range s.mammoths {
		out = append(out, protocol.Mammoth{
			ID: m.ID,
			X:  m.Pos.X, Y: m.Pos.Y, Z: m.Pos.Z,
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
	return out
}
