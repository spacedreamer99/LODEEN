package net

import (
	mrand "math/rand"
	"time"

	"github.com/spacedreamer99/lodeen/internal/shared/protocol"
)

func (s *Server) tickBreeding() {
	const closeD2 = 5.0 * 5.0
	now := time.Now()

	s.mammoths.Lock()
	defer s.mammoths.Unlock()

	// 1. Рост мамонтят
	for _, m := range s.mammoths.Map() {
		if m.Baby && !m.BornAt.IsZero() && now.Sub(m.BornAt) >= babyGrowTime {
			m.Baby = false
			s.log.Info("mammoth grew up", "id", m.ID)
		}
	}

	// 2. Кандидаты — приручённые взрослые, накормленные 4+
	var males, females []*Mammoth
	for _, m := range s.mammoths.Map() {
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
			s.mammoths.Map()[babyID] = &Mammoth{
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
