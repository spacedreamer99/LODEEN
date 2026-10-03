package net

import (
	"math"
	mrand "math/rand"
	"time"

	"github.com/spacedreamer99/lodeen/internal/shared/protocol"
)

const (
	projTTL   = 3 * time.Second
	projHitD2 = 1.5 * 1.5
)

// mobHit — отложенное попадание снаряда в моба. Копится под s.projectiles.Lock(),
// применяется после unlock — чтобы не брать s.mobs.Lock() внутри.
type mobHit struct {
	id string
}

// killDrop — моб, убитый снарядом, с данными для дропа.
type killDrop struct {
	pos    protocol.Vector3
	kind   string
	mobInv map[string]int
}

// tickProjectiles — оркестратор тика снарядов: движение → попадания → дропы.
func (s *Server) tickProjectiles(dt float32) {
	now := time.Now()

	// Снимок мобов ДО s.projectiles.Lock() — иначе deadlock с tickMobs.
	mobPos := s.snapshotMobPositions()

	// Движение + первичная проверка попаданий под одним lock'ом.
	mobHits := s.advanceProjectiles(now, dt, mobPos)
	if len(mobHits) == 0 {
		return
	}

	// Применяем попадания в мобов, собираем убитых.
	kills := s.applyMobHits(mobHits)
	if len(kills) == 0 {
		return
	}

	// Спавним дроп после unlock всех мобов.
	s.spawnDrops(kills)
}

// snapshotMobPositions копирует позиции мобов в map. Нужен, чтобы не держать
// s.mobs.Lock() одновременно с s.projectiles.Lock().
func (s *Server) snapshotMobPositions() map[string]protocol.Vector3 {
	s.mobs.RLock()
	defer s.mobs.RUnlock()
	out := make(map[string]protocol.Vector3, len(s.mobs.Map()))
	for id, mo := range s.mobs.Map() {
		out[id] = mo.Pos
	}
	return out
}

// advanceProjectiles двигает все снаряды, удаляет по TTL и собирает
// попадания: в игрока (мгновенно) и в моба (отложенно, вернуть mobHits).
func (s *Server) advanceProjectiles(now time.Time, dt float32, mobPos map[string]protocol.Vector3) []mobHit {
	var mobHits []mobHit

	s.projectiles.Lock()
	defer s.projectiles.Unlock()

	for id, p := range s.projectiles.Map() {
		if now.Sub(p.SpawnAt) > projTTL {
			delete(s.projectiles.Map(), id)
			continue
		}

		// Движение.
		p.Pos.X += p.Dir.X * p.Speed * dt
		p.Pos.Y += p.Dir.Y * p.Speed * dt
		p.Pos.Z += p.Dir.Z * p.Speed * dt

		// Попадание в игрока — мгновенно.
		if s.tryProjectileHitClient(p) {
			delete(s.projectiles.Map(), id)
			continue
		}

		// Попадание в моба — отложенно.
		if s.tryProjectileHitMob(p, mobPos) {
			mobHits = append(mobHits, mobHit{id: p.TargetID})
			delete(s.projectiles.Map(), id)
		}
		// Если цель исчезла — снаряд летит дальше до TTL.
	}
	return mobHits
}

// tryProjectileHitClient проверяет попадание в целевого игрока и наносит урон.
// Возвращает true, если снаряд попал (и должен быть удалён).
func (s *Server) tryProjectileHitClient(p *Projectile) bool {
	target := s.findClient(p.TargetID)
	if target == nil {
		return false
	}
	ps := target.State()
	dx := ps.X - p.Pos.X
	dy := ps.Y - p.Pos.Y
	dz := ps.Z - p.Pos.Z
	if dx*dx+dy*dy+dz*dz >= projHitD2 {
		return false
	}
	target.damage(mobAttackDamage)
	s.log.Info("mob spear hit", "target", p.TargetID, "dmg", mobAttackDamage)
	return true
}

// tryProjectileHitMob проверяет попадание в моба по снимку позиций.
func (s *Server) tryProjectileHitMob(p *Projectile, mobPos map[string]protocol.Vector3) bool {
	mp, ok := mobPos[p.TargetID]
	if !ok {
		return false
	}
	dx := mp.X - p.Pos.X
	dy := mp.Y - p.Pos.Y
	dz := mp.Z - p.Pos.Z
	return dx*dx+dy*dy+dz*dz < projHitD2
}

// applyMobHits наносит урон всем попаданиям и возвращает убитых мобов.
func (s *Server) applyMobHits(hits []mobHit) []killDrop {
	var kills []killDrop

	s.mobs.Lock()
	defer s.mobs.Unlock()

	for _, h := range hits {
		mo, ok := s.mobs.Map()[h.id]
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
			delete(s.mobs.Map(), h.id)
			s.log.Info("mob killed by projectile", "id", h.id, "kind", mo.Kind)
		} else {
			s.log.Info("mob hit by projectile", "id", h.id, "hp", mo.HP)
		}
	}
	return kills
}

// spawnDrops разбрасывает дроп вокруг позиций убитых мобов.
func (s *Server) spawnDrops(kills []killDrop) {
	s.resources.Lock()
	defer s.resources.Unlock()

	for _, k := range kills {
		switch k.kind {
		case "hostile":
			s.spawnHostileDrop(k.pos)
		case "collector":
			s.spawnCollectorDrop(k.pos, k.mobInv)
		}
	}
}

// spawnHostileDrop — hostile-моб дропает 10 копий копья в радиусе 2м.
func (s *Server) spawnHostileDrop(pos protocol.Vector3) {
	for i := 0; i < 10; i++ {
		rid := newID()
		pp := scatteredAround(pos, 2.0)
		s.resources.Map()[rid] = protocol.Resource{
			ID: rid, Type: "spear",
			X: pp.X, Y: pp.Y, Z: pp.Z,
		}
	}
}

// spawnCollectorDrop — collector-моб дропает содержимое инвентаря в радиусе 2.5м.
func (s *Server) spawnCollectorDrop(pos protocol.Vector3, mobInv map[string]int) {
	for itemType, qty := range mobInv {
		for i := 0; i < qty; i++ {
			rid := newID()
			pp := scatteredAround(pos, 2.5)
			s.resources.Map()[rid] = protocol.Resource{
				ID: rid, Type: itemType,
				X: pp.X, Y: pp.Y, Z: pp.Z,
			}
		}
	}
}

// scatteredAround возвращает точку на поверхности вокруг base в радиусе <= r.
func scatteredAround(base protocol.Vector3, r float32) protocol.Vector3 {
	theta := mrand.Float32() * 2 * math.Pi
	rr := mrand.Float32() * r
	dx := float32(math.Cos(float64(theta))) * rr
	dz := float32(math.Sin(float64(theta))) * rr
	return protocol.ClampToSurface(protocol.Vector3{
		X: base.X + dx, Y: base.Y, Z: base.Z + dz,
	})
}
