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
	s.resources.RLock()
	defer s.resources.RUnlock()
	out := make([]protocol.Resource, 0, len(s.resources.Map()))
	for _, r := range s.resources.Map() {
		out = append(out, r)
	}
	return out
}

// collectWells — колодцы (вода).
func (s *Server) collectWells() []protocol.Well {
	s.wells.RLock()
	defer s.wells.RUnlock()
	out := make([]protocol.Well, 0, len(s.wells.Map()))
	for _, w := range s.wells.Map() {
		out = append(out, *w)
	}
	return out
}

// collectProjectiles — летящие снаряды мобов.
func (s *Server) collectProjectiles() []protocol.MobProjectile {
	s.projectiles.RLock()
	defer s.projectiles.RUnlock()
	out := make([]protocol.MobProjectile, 0, len(s.projectiles.Map()))
	for _, p := range s.projectiles.Map() {
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
	s.mobs.RLock()
	defer s.mobs.RUnlock()
	out := make([]protocol.Mob, 0, len(s.mobs.Map()))
	for _, m := range s.mobs.Map() {
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
	s.boats.RLock()
	defer s.boats.RUnlock()
	out := make([]protocol.Boat, 0, len(s.boats.Map()))
	for _, b := range s.boats.Map() {
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
	s.houses.RLock()
	defer s.houses.RUnlock()
	out := make([]protocol.House, 0, len(s.houses.Map()))
	for _, h := range s.houses.Map() {
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
	s.solar.RLock()
	defer s.solar.RUnlock()
	out := make([]protocol.Solar, 0, len(s.solar.Map()))
	for _, sl := range s.solar.Map() {
		out = append(out, protocol.Solar{
			ID: sl.ID, X: sl.Pos.X, Y: sl.Pos.Y, Z: sl.Pos.Z, Yaw: sl.Yaw,
		})
	}
	return out
}

// collectBatteries — батареи с уровнем энергии.
func (s *Server) collectBatteries() []protocol.Battery {
	s.batteries.RLock()
	defer s.batteries.RUnlock()
	out := make([]protocol.Battery, 0, len(s.batteries.Map()))
	for _, b := range s.batteries.Map() {
		out = append(out, protocol.Battery{
			ID: b.ID, X: b.Pos.X, Y: b.Pos.Y, Z: b.Pos.Z, Yaw: b.Yaw,
			Energy: b.Energy, MaxEnergy: b.MaxEnergy,
		})
	}
	return out
}

// collectFactories — фабрики с прогрессом крафта.
func (s *Server) collectFactories() []protocol.Factory {
	s.factories.RLock()
	defer s.factories.RUnlock()
	out := make([]protocol.Factory, 0, len(s.factories.Map()))
	for _, f := range s.factories.Map() {
		out = append(out, protocol.Factory{
			ID: f.ID, X: f.Pos.X, Y: f.Pos.Y, Z: f.Pos.Z, Yaw: f.Yaw,
			Crafting: f.Crafting, Progress: f.Progress,
		})
	}
	return out
}

// collectRockets — ракеты со всей орбитальной телеметрией.
func (s *Server) collectRockets() []protocol.Rocket {
	s.rockets.RLock()
	defer s.rockets.RUnlock()
	out := make([]protocol.Rocket, 0, len(s.rockets.Map()))
	for _, r := range s.rockets.Map() {
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
	s.mammoths.RLock()
	defer s.mammoths.RUnlock()
	out := make([]protocol.Mammoth, 0, len(s.mammoths.Map()))
	for _, m := range s.mammoths.Map() {
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
