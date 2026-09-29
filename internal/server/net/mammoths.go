package net

import (
	"crypto/rand"
	"encoding/binary"
	"math"
	mrand "math/rand"
	"time"

	"github.com/spacedreamer99/lodeen/internal/shared/protocol"
)

// Этот файл содержит Mammoth struct и всю логику мамонтов:
// spawn, tick, taming, breeding, leash, saddle, riding, hit.

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


func (s *Server) handleTameMammoth(c *Client, mammothID string) {
	ps := c.State()

	s.mammothsMu.Lock()
	m, ok := s.mammoths[mammothID]
	if !ok {
		s.mammothsMu.Unlock()
		c.log.Warn("tame: mammoth not found")
		return
	}
	dx := float64(m.Pos.X - ps.X)
	dy := float64(m.Pos.Y - ps.Y)
	dz := float64(m.Pos.Z - ps.Z)
	if dx*dx+dy*dy+dz*dz > 5.0*5.0 {
		s.mammothsMu.Unlock()
		c.log.Warn("tame: too far")
		return
	}

	if !m.Tamed {
		m.Tamed = true
		s.mammothsMu.Unlock()
		if !c.consumeItem("fruit") {
			// откатываем
			s.mammothsMu.Lock()
			m.Tamed = false
			s.mammothsMu.Unlock()
			return
		}
		c.log.Info("mammoth tamed", "id", mammothID)
	} else {
		if !c.consumeItem("fruit") {
			s.mammothsMu.Unlock()
			return
		}
		m.FedCount++
		fed := m.FedCount
		s.mammothsMu.Unlock()
		c.log.Info("mammoth fed", "id", mammothID, "count", fed)
	}

	c.mu.Lock()
	inv := make(map[string]int, len(c.inventory))
	for k, v := range c.inventory {
		inv[k] = v
	}
	c.mu.Unlock()
	c.sendEnvelope(protocol.TypeInventoryUpdate, protocol.InventoryUpdate{Items: inv})
}


func (s *Server) tickBreeding() {
	const closeD2 = 5.0 * 5.0
	now := time.Now()

	s.mammothsMu.Lock()
	defer s.mammothsMu.Unlock()

	// 1. Рост мамонтят
	for _, m := range s.mammoths {
		if m.Baby && !m.BornAt.IsZero() && now.Sub(m.BornAt) >= babyGrowTime {
			m.Baby = false
			s.log.Info("mammoth grew up", "id", m.ID)
		}
	}

	// 2. Кандидаты — приручённые взрослые, накормленные 4+
	var males, females []*Mammoth
	for _, m := range s.mammoths {
		if !m.Tamed || m.Baby || m.FedCount < 4 {
			continue
		}
		if m.Sex == "m" {
			males = append(males, m)
		} else if m.Sex == "f" {
			females = append(females, m)
		}
	}

	for _, male := range males {
		for _, female := range females {
			dx := male.Pos.X - female.Pos.X
			dy := male.Pos.Y - female.Pos.Y
			dz := male.Pos.Z - female.Pos.Z
			if dx*dx+dy*dy+dz*dz > closeD2 {
				continue
			}
			// рожаем сразу
			babyID := newID()
			babySex := "m"
			if mrand.Intn(2) == 0 {
				babySex = "f"
			}
			mid := protocol.Vector3{
				X: (male.Pos.X + female.Pos.X) / 2,
				Y: (male.Pos.Y + female.Pos.Y) / 2,
				Z: (male.Pos.Z + female.Pos.Z) / 2,
			}
			s.mammoths[babyID] = &Mammoth{
				ID:     babyID,
				Pos:    mid,
				HP:     1,
				Tamed:  true,
				Sex:    babySex,
				Baby:   true,
				BornAt: now,
			}
			male.FedCount = 0
			female.FedCount = 0
			s.log.Info("mammoth baby born",
				"baby", babyID, "sex", babySex,
				"parent_m", male.ID, "parent_f", female.ID)
			return
		}
	}
}


func (s *Server) handleLeashMammoth(c *Client, mammothID string) {
	ps := c.State()

	s.mammothsMu.Lock()
	m, ok := s.mammoths[mammothID]
	if !ok {
		s.mammothsMu.Unlock()
		c.log.Warn("leash: not found")
		return
	}
	dx := float64(m.Pos.X - ps.X)
	dy := float64(m.Pos.Y - ps.Y)
	dz := float64(m.Pos.Z - ps.Z)
	if dx*dx+dy*dy+dz*dz > 5.0*5.0 {
		s.mammothsMu.Unlock()
		c.log.Warn("leash: too far")
		return
	}
	if !m.Tamed {
		s.mammothsMu.Unlock()
		c.log.Warn("leash: not tamed")
		return
	}
	myID := c.ID
	if m.LeashedTo == myID {
		m.LeashedTo = ""
		s.mammothsMu.Unlock()
		c.log.Info("mammoth unleashed", "id", mammothID)
		return
	}
	m.LeashedTo = myID
	s.mammothsMu.Unlock()

	c.log.Info("mammoth leashed", "id", mammothID)
}


func (s *Server) handleSaddleMammoth(c *Client, mammothID string) {
	ps := c.State()

	s.mammothsMu.Lock()
	m, ok := s.mammoths[mammothID]
	if !ok {
		s.mammothsMu.Unlock()
		c.log.Warn("saddle: not found")
		return
	}
	dx := float64(m.Pos.X - ps.X)
	dy := float64(m.Pos.Y - ps.Y)
	dz := float64(m.Pos.Z - ps.Z)
	if dx*dx+dy*dy+dz*dz > 5.0*5.0 {
		s.mammothsMu.Unlock()
		c.log.Warn("saddle: too far")
		return
	}
	if !m.Tamed {
		s.mammothsMu.Unlock()
		c.log.Warn("saddle: not tamed")
		return
	}
	if m.Baby {
		s.mammothsMu.Unlock()
		c.log.Warn("saddle: baby")
		return
	}
	if !m.Saddle {
		// ставим: нужен предмет
		if !c.consumeItem("saddle") {
			s.mammothsMu.Unlock()
			return
		}
		m.Saddle = true
		s.mammothsMu.Unlock()
		c.log.Info("saddle installed", "id", mammothID)
	} else {
		// снимаем
		if m.RiderID != "" {
			s.mammothsMu.Unlock()
			c.log.Warn("saddle: rider on it")
			return
		}
		m.Saddle = false
		s.mammothsMu.Unlock()
		c.addItem("saddle")
		c.log.Info("saddle removed", "id", mammothID)
	}

	c.mu.Lock()
	inv := make(map[string]int, len(c.inventory))
	for k, v := range c.inventory {
		inv[k] = v
	}
	c.mu.Unlock()
	c.sendEnvelope(protocol.TypeInventoryUpdate, protocol.InventoryUpdate{Items: inv})
}


func (s *Server) handleRideMammoth(c *Client, mammothID string) {
	ps := c.State()

	s.mammothsMu.Lock()
	for _, m := range s.mammoths {
		if m.RiderID == c.ID {
			m.RiderID = ""
			s.mammothsMu.Unlock()
			c.log.Info("dismounted")
			return
		}
	}
	if mammothID == "" {
		s.mammothsMu.Unlock()
		return
	}
	m, ok := s.mammoths[mammothID]
	if !ok {
		s.mammothsMu.Unlock()
		c.log.Warn("ride: not found")
		return
	}
	dx := float64(m.Pos.X - ps.X)
	dy := float64(m.Pos.Y - ps.Y)
	dz := float64(m.Pos.Z - ps.Z)
	if dx*dx+dy*dy+dz*dz > 6.0*6.0 {
		s.mammothsMu.Unlock()
		c.log.Warn("ride: too far")
		return
	}
	if !m.Tamed || !m.Saddle || m.RiderID != "" {
		s.mammothsMu.Unlock()
		c.log.Warn("ride: not ready")
		return
	}
	m.RiderID = c.ID
	s.mammothsMu.Unlock()
	c.log.Info("mounted", "id", mammothID)
}


func (s *Server) tickMammoths(dt float32) {
	const fleeRadius = 30.0
	const speed = 8.0
	const mammothR = float64(protocol.PlanetRadius) + 2.0

	// Собираем игроков + что у каждого в руке
	type nearPlayer struct {
		id    string
		state protocol.PlayerState
		held  string
		fruit bool
	}
	s.mu.RLock()
	players := make([]nearPlayer, 0, len(s.clients))
	for _, c := range s.clients {
		c.mu.Lock()
		held := c.heldItem
		hasFruit := c.inventory["fruit"] > 0
		c.mu.Unlock()
		players = append(players, nearPlayer{
			id:    c.ID,
			state: c.State(),
			held:  held,
			fruit: hasFruit,
		})
	}
	s.mu.RUnlock()

	s.mammothsMu.Lock()
	defer s.mammothsMu.Unlock()

	for _, m := range s.mammoths {
		// Всадник управляет позицией мамонта.
		if m.RiderID != "" {
			var rider *nearPlayer
			for i := range players {
				if players[i].id == m.RiderID {
					rider = &players[i]
					break
				}
			}
			if rider == nil {
				m.RiderID = ""
			} else {
				m.Pos.X = rider.state.X
				m.Pos.Y = rider.state.Y - 1.0
				m.Pos.Z = rider.state.Z
			}
			continue
		}
		if m.LeashedTo != "" {
			var owner *protocol.PlayerState
			for i := range players {
				if players[i].id == m.LeashedTo {
					owner = &players[i].state
					break
				}
			}
			if owner == nil {
				m.LeashedTo = ""
			} else {
				dx := owner.X - m.Pos.X
				dy := owner.Y - m.Pos.Y
				dz := owner.Z - m.Pos.Z
				d2 := dx*dx + dy*dy + dz*dz
				if d2 > 9.0 {
					d := float32(math.Sqrt(float64(d2)))
					const followSpeed = 6.0
					m.Pos.X += (dx / d) * followSpeed * dt
					m.Pos.Y += (dy / d) * followSpeed * dt
					m.Pos.Z += (dz / d) * followSpeed * dt
					m.Pos = protocol.ClampToSurface(m.Pos)
				}
			}
			continue
		}
		if m.Tamed {
			continue // приручённый мамонт не убегает
		}
		// ближайший игрок
		var nearest *protocol.PlayerState
		nearestHeld := ""
		minD2 := float32(fleeRadius * fleeRadius)
		for i := range players {
			dx := players[i].state.X - m.Pos.X
			dy := players[i].state.Y - m.Pos.Y
			dz := players[i].state.Z - m.Pos.Z
			d2 := dx*dx + dy*dy + dz*dz
			if d2 < minD2 {
				minD2 = d2
				nearest = &players[i].state
				nearestHeld = players[i].held
			}
		}
		if nearest == nil {
			continue
		}
		// Если игрок держит фрукт — мамонт не убегает, ждёт кормления.
		if nearestHeld == "fruit" {
			continue
		}
		s.log.Debug("mammoth flees",
			"id", m.ID, "nearest_held", nearestHeld, "d2", minD2)
		s.log.Debug("mammoth flee",
			"mammoth", m.ID,
			"nearest_held", nearestHeld,
			"d2", minD2)

		// направление "от игрока" в касательной плоскости
		upX := m.Pos.X
		upY := m.Pos.Y
		upZ := m.Pos.Z
		l := float32(math.Sqrt(float64(upX*upX + upY*upY + upZ*upZ)))
		if l < 0.01 {
			continue
		}
		upX /= l
		upY /= l
		upZ /= l

		toX := nearest.X - m.Pos.X
		toY := nearest.Y - m.Pos.Y
		toZ := nearest.Z - m.Pos.Z
		dot := toX*upX + toY*upY + toZ*upZ
		tanX := toX - upX*dot
		tanY := toY - upY*dot
		tanZ := toZ - upZ*dot
		lt := float32(math.Sqrt(float64(tanX*tanX + tanY*tanY + tanZ*tanZ)))
		if lt < 0.1 {
			continue
		}
		// нормализуем и инвертируем (бежать ОТ игрока)
		tanX = -tanX / lt
		tanY = -tanY / lt
		tanZ = -tanZ / lt

		m.Pos.X += tanX * speed * dt
		m.Pos.Y += tanY * speed * dt
		m.Pos.Z += tanZ * speed * dt

		// прижать к поверхности
		rl := float32(math.Sqrt(float64(m.Pos.X*m.Pos.X + m.Pos.Y*m.Pos.Y + m.Pos.Z*m.Pos.Z)))
		if rl < 0.01 {
			continue
		}
		_ = mammothR
		m.Pos = protocol.ClampToSurface(m.Pos)
	}
}


func (s *Server) handleHitMammoth(c *Client, mammothID string) {
	now := time.Now()

	// Тихо гасим дубли HitMammoth в пределах 200 мс — норма для клиента.
	if now.Sub(c.lastHitAt) < 200*time.Millisecond {
		return
	}

	// Валидация: игрок недавно бросил копьё.
	if now.Sub(c.lastThrowAt) > 2*time.Second {
		c.log.Warn("hit rejected: no recent throw")
		return
	}

	c.lastHitAt = now

	// Валидация: мамонт существует.
	s.mammothsMu.Lock()
	m, ok := s.mammoths[mammothID]
	if !ok {
		s.mammothsMu.Unlock()
		c.log.Warn("hit rejected: mammoth not found")
		return
	}

	// Валидация: мамонт в разумной близости от игрока.
	ps := c.State()
	dx := m.Pos.X - ps.X
	dy := m.Pos.Y - ps.Y
	dz := m.Pos.Z - ps.Z
	dist2 := dx*dx + dy*dy + dz*dz
	const maxD2 float32 = 80.0 * 80.0
	if dist2 > maxD2 {
		s.mammothsMu.Unlock()
		c.log.Warn("hit rejected: too far", "d2", dist2)
		return
	}

	m.HP--
	var meatPos protocol.Vector3
	killed := m.HP <= 0
	if killed {
		meatPos = m.Pos
		delete(s.mammoths, mammothID)
	}
	s.mammothsMu.Unlock()

	if killed {
		s.resourcesMu.Lock()
		meatID := newID()
		s.resources[meatID] = protocol.Resource{
			ID:   meatID,
			Type: "meat",
			X:    meatPos.X,
			Y:    meatPos.Y,
			Z:    meatPos.Z,
		}
		// Небольшой сдвиг, чтобы копьё не совпадало с мясом и его можно было подобрать отдельно.
		spearID := newID()
		s.resources[spearID] = protocol.Resource{
			ID:   spearID,
			Type: "spear",
			X:    meatPos.X + 1.5,
			Y:    meatPos.Y,
			Z:    meatPos.Z + 1.5,
		}
		s.resourcesMu.Unlock()
		c.log.Info("mammoth KILLED", "id", mammothID)
	} else {
		c.log.Info("mammoth hit", "id", mammothID, "hp", m.HP)
	}
}

