package net

import (
	"math"
	mrand "math/rand"
	"time"

	"github.com/spacedreamer99/lodeen/internal/shared/protocol"
)

func (s *Server) tickProjectiles(dt float32) {
	now := time.Now()
	const projTTL = 3 * time.Second
	const projHitD2 = 1.5 * 1.5

	// Снимок мобов ДО projMu — иначе deadlock с tickMobs.
	s.mobsMu.RLock()
	mobPos := make(map[string]protocol.Vector3, len(s.mobs))
	for id, mo := range s.mobs {
		mobPos[id] = mo.Pos
	}
	s.mobsMu.RUnlock()

	type mobHit struct {
		id string
	}
	var mobHits []mobHit

	s.projMu.Lock()
	for id, p := range s.projectiles {
		if now.Sub(p.SpawnAt) > projTTL {
			delete(s.projectiles, id)
			continue
		}
		p.Pos.X += p.Dir.X * p.Speed * dt
		p.Pos.Y += p.Dir.Y * p.Speed * dt
		p.Pos.Z += p.Dir.Z * p.Speed * dt

		// 1. Проверка попадания в целевого игрока.
		s.mu.RLock()
		var target *Client
		for _, c := range s.clients {
			if c.ID == p.TargetID {
				target = c
				break
			}
		}
		s.mu.RUnlock()
		if target != nil {
			ps := target.State()
			dx := ps.X - p.Pos.X
			dy := ps.Y - p.Pos.Y
			dz := ps.Z - p.Pos.Z
			if dx*dx+dy*dy+dz*dz < projHitD2 {
				target.damage(mobAttackDamage)
				s.log.Info("mob spear hit", "target", p.TargetID, "dmg", mobAttackDamage)
				delete(s.projectiles, id)
			}
			continue
		}

		// 2. Проверка попадания в моба-цель.
		if mp, ok := mobPos[p.TargetID]; ok {
			dx := mp.X - p.Pos.X
			dy := mp.Y - p.Pos.Y
			dz := mp.Z - p.Pos.Z
			if dx*dx+dy*dy+dz*dz < projHitD2 {
				mobHits = append(mobHits, mobHit{id: p.TargetID})
				delete(s.projectiles, id)
			}
		}
		// Если цель исчезла — снаряд летит дальше до TTL.
	}
	s.projMu.Unlock()

	// Применяем попадания в мобов.
	if len(mobHits) > 0 {
		type killDrop struct {
			pos    protocol.Vector3
			kind   string
			mobInv map[string]int
		}
		var kills []killDrop

		s.mobsMu.Lock()
		for _, h := range mobHits {
			mo, ok := s.mobs[h.id]
			if !ok {
				continue
			}
			mo.HP -= mobAttackDamage
			if mo.HP <= 0 {
				kills = append(kills, killDrop{
					pos:    mo.Pos,
					kind:   mo.Kind,
					mobInv: mo.Inventory,
				})
				delete(s.mobs, h.id)
				s.log.Info("mob killed by projectile", "id", h.id, "kind", mo.Kind)
			} else {
				s.log.Info("mob hit by projectile", "id", h.id, "hp", mo.HP)
			}
		}
		s.mobsMu.Unlock()

		// Спавним дроп после unlock.
		if len(kills) > 0 {
			s.resourcesMu.Lock()
			for _, k := range kills {
				switch k.kind {
				case "hostile":
					for i := 0; i < 10; i++ {
						rid := newID()
						theta := mrand.Float32() * 2 * math.Pi
						rr := mrand.Float32() * 2.0
						dx := float32(math.Cos(float64(theta))) * rr
						dz := float32(math.Sin(float64(theta))) * rr
						pp := protocol.ClampToSurface(protocol.Vector3{
							X: k.pos.X + dx, Y: k.pos.Y, Z: k.pos.Z + dz,
						})
						s.resources[rid] = protocol.Resource{
							ID: rid, Type: "spear",
							X: pp.X, Y: pp.Y, Z: pp.Z,
						}
					}
				case "collector":
					for itemType, qty := range k.mobInv {
						for i := 0; i < qty; i++ {
							rid := newID()
							theta := mrand.Float32() * 2 * math.Pi
							rr := mrand.Float32() * 2.5
							dx := float32(math.Cos(float64(theta))) * rr
							dz := float32(math.Sin(float64(theta))) * rr
							pp := protocol.ClampToSurface(protocol.Vector3{
								X: k.pos.X + dx, Y: k.pos.Y, Z: k.pos.Z + dz,
							})
							s.resources[rid] = protocol.Resource{
								ID: rid, Type: itemType,
								X: pp.X, Y: pp.Y, Z: pp.Z,
							}
						}
					}
				}
			}
			s.resourcesMu.Unlock()
		}
	}
}
