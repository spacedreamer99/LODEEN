package net

import (
	"math"

	"github.com/spacedreamer99/lodeen/internal/shared/protocol"
)

// nearPlayer is a player snapshot for the mammoth tick.
type nearPlayer struct {
	id    string
	state protocol.PlayerState
	held  string
	fruit bool
}

// collectNearPlayers snapshots all connected clients for the mammoth tick.
func (s *Server) collectNearPlayers() []nearPlayer {
	s.mu.RLock()
	defer s.mu.RUnlock()
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
	return players
}

// findNearPlayer finds a snapshot by player ID.
func findNearPlayer(players []nearPlayer, id string) *nearPlayer {
	for i := range players {
		if players[i].id == id {
			return &players[i]
		}
	}
	return nil
}

// tickRiddenMammoth: rider controls the mammoth position.
func tickRiddenMammoth(m *Mammoth, players []nearPlayer) {
	rider := findNearPlayer(players, m.RiderID)
	if rider == nil {
		m.RiderID = ""
		return
	}
	m.Pos.X = rider.state.X
	m.Pos.Y = rider.state.Y - 1.0
	m.Pos.Z = rider.state.Z
}

// tickLeashedMammoth: leashed mammoth follows owner if > 3m away.
func tickLeashedMammoth(m *Mammoth, players []nearPlayer, dt float32) {
	owner := findNearPlayer(players, m.LeashedTo)
	if owner == nil {
		m.LeashedTo = ""
		return
	}
	dx := owner.state.X - m.Pos.X
	dy := owner.state.Y - m.Pos.Y
	dz := owner.state.Z - m.Pos.Z
	d2 := dx*dx + dy*dy + dz*dz
	if d2 <= 9.0 {
		return
	}
	d := float32(math.Sqrt(float64(d2)))
	const followSpeed = 6.0
	m.Pos.X += (dx / d) * followSpeed * dt
	m.Pos.Y += (dy / d) * followSpeed * dt
	m.Pos.Z += (dz / d) * followSpeed * dt
	m.Pos = protocol.ClampToSurface(m.Pos)
}

// tickWildMammoth: wild mammoth flees nearest player in tangent plane.
// Does not flee if the player holds a fruit (waits to be fed).
func tickWildMammoth(m *Mammoth, players []nearPlayer, dt float32) {
	const fleeRadius = 30.0
	const speed = 8.0

	var nearest *nearPlayer
	minD2 := float32(fleeRadius * fleeRadius)
	for i := range players {
		dx := players[i].state.X - m.Pos.X
		dy := players[i].state.Y - m.Pos.Y
		dz := players[i].state.Z - m.Pos.Z
		d2 := dx*dx + dy*dy + dz*dz
		if d2 < minD2 {
			minD2 = d2
			nearest = &players[i]
		}
	}
	if nearest == nil {
		return
	}
	if nearest.held == "fruit" {
		return
	}

	// direction "away from player" in the tangent plane
	upX, upY, upZ := m.Pos.X, m.Pos.Y, m.Pos.Z
	l := float32(math.Sqrt(float64(upX*upX + upY*upY + upZ*upZ)))
	if l < 0.01 {
		return
	}
	upX /= l
	upY /= l
	upZ /= l

	toX := nearest.state.X - m.Pos.X
	toY := nearest.state.Y - m.Pos.Y
	toZ := nearest.state.Z - m.Pos.Z
	dot := toX*upX + toY*upY + toZ*upZ
	tanX := toX - upX*dot
	tanY := toY - upY*dot
	tanZ := toZ - upZ*dot
	lt := float32(math.Sqrt(float64(tanX*tanX + tanY*tanY + tanZ*tanZ)))
	if lt < 0.1 {
		return
	}
	tanX = -tanX / lt
	tanY = -tanY / lt
	tanZ = -tanZ / lt

	m.Pos.X += tanX * speed * dt
	m.Pos.Y += tanY * speed * dt
	m.Pos.Z += tanZ * speed * dt

	// clamp to surface
	rl := float32(math.Sqrt(float64(m.Pos.X*m.Pos.X + m.Pos.Y*m.Pos.Y + m.Pos.Z*m.Pos.Z)))
	if rl < 0.01 {
		return
	}
	m.Pos = protocol.ClampToSurface(m.Pos)
}
