package net

import (
	mrand "math/rand"
	"time"

	"github.com/spacedreamer99/lodeen/internal/shared/protocol"
)

// breeding — параметры размножения.
const (
	breedingCloseD2 = 5.0 * 5.0 // радиус "рядом"
	breedingMinFed  = 4         // сколько раз надо накормить
)

// tickBreeding — оркестратор: подрастить мамонтят, найти пару, родить.
// Рожает максимум одного мамонтёнка за тик.
func (s *Server) tickBreeding() {
	now := time.Now()

	s.mammoths.Lock()
	defer s.mammoths.Unlock()

	s.growUpMammoths(now)

	males, females := collectBreedingCandidates(s.mammoths.Map())
	for _, male := range males {
		for _, female := range females {
			if !mammothsAreClose(male, female) {
				continue
			}
			s.birthMammoth(male, female, now)
			return // один детёныш за тик
		}
	}
}

// growUpMammoths превращает мамонтят во взрослых, когда они выросли.
func (s *Server) growUpMammoths(now time.Time) {
	for _, m := range s.mammoths.Map() {
		if !m.Baby || m.BornAt.IsZero() {
			continue
		}
		if now.Sub(m.BornAt) >= babyGrowTime {
			m.Baby = false
			s.log.Info("mammoth grew up", "id", m.ID)
		}
	}
}

// collectBreedingCandidates собирает приручённых взрослых, накормленных
// breedingMinFed+ раз, разбитых по полу.
func collectBreedingCandidates(all map[string]*Mammoth) (males, females []*Mammoth) {
	for _, m := range all {
		if !m.Tamed || m.Baby || m.FedCount < breedingMinFed {
			continue
		}
		switch m.Sex {
		case "m":
			males = append(males, m)
		case "f":
			females = append(females, m)
		}
	}
	return males, females
}

// mammothsAreClose — пара находится в радиусе breedingCloseD2.
func mammothsAreClose(a, b *Mammoth) bool {
	dx := a.Pos.X - b.Pos.X
	dy := a.Pos.Y - b.Pos.Y
	dz := a.Pos.Z - b.Pos.Z
	return dx*dx+dy*dy+dz*dz <= breedingCloseD2
}

// birthMammoth рождает мамонтёнка посередине между родителями.
// Сбрасывает FedCount обоих родителей.
func (s *Server) birthMammoth(male, female *Mammoth, now time.Time) {
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
}
