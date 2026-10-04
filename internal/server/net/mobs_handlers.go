package net

import (
	"github.com/spacedreamer99/lodeen/internal/shared/protocol"
)

// hitMobResult — итог нанесения урона мобу.
type hitMobResult struct {
	killed  bool
	hpAfter int
	kind    string
	drop    killDrop // валиден только если killed
}

// handleHitMob — оркестратор удара по мобу:
// валидация+урон → дроп (если убит) → inventory update.
func (s *Server) handleHitMob(c *Client, mobID string) {
	ps := c.State()
	res, ok := s.applyHitMob(c, mobID, ps)
	if !ok {
		return
	}

	if res.killed {
		s.spawnDrops([]killDrop{res.drop})
		c.log.Info("mob killed", "id", mobID, "kind", res.kind)
	} else {
		c.log.Info("mob hit", "id", mobID, "hp", res.hpAfter, "kind", res.kind)
	}

	s.sendInventory(c)
}

// applyHitMob под одним lock'ом: находит моба, валидирует дистанцию,
// списывает spear, наносит урон, помечает collector как Angered,
// удаляет убитого. Возвращает ok=false, если удар не состоялся.
func (s *Server) applyHitMob(c *Client, mobID string, ps protocol.PlayerState) (hitMobResult, bool) {
	s.mobs.Lock()
	defer s.mobs.Unlock()

	m, ok := s.mobs.Map()[mobID]
	if !ok {
		c.log.Warn("hit mob: not found")
		return hitMobResult{}, false
	}
	dx := float64(m.Pos.X - ps.X)
	dy := float64(m.Pos.Y - ps.Y)
	dz := float64(m.Pos.Z - ps.Z)
	if dx*dx+dy*dy+dz*dz > 100.0*100.0 {
		c.log.Warn("hit mob: too far")
		return hitMobResult{}, false
	}
	if !c.consumeItem("spear") {
		return hitMobResult{}, false
	}
	m.HP--
	if m.Kind == "collector" {
		m.Angered = true
	}

	res := hitMobResult{
		hpAfter: m.HP,
		kind:    m.Kind,
		killed:  m.HP <= 0,
	}
	if res.killed {
		res.drop = killDrop{
			pos:    m.Pos,
			kind:   m.Kind,
			mobInv: m.Inventory,
		}
		delete(s.mobs.Map(), mobID)
	}
	return res, true
}

// sendInventory отправляет клиенту актуальный InventoryUpdate.
func (s *Server) sendInventory(c *Client) {
	c.mu.Lock()
	inv := make(map[string]int, len(c.inventory))
	for k, v := range c.inventory {
		inv[k] = v
	}
	c.mu.Unlock()
	c.sendEnvelope(protocol.TypeInventoryUpdate, protocol.InventoryUpdate{Items: inv})
}

// handleAcceptContract — оркестратор: принять контракт → залогировать → inventory update.
func (s *Server) handleAcceptContract(c *Client, mobID, contractID string) {
	if !s.tryAcceptContract(c, mobID, contractID) {
		return
	}
	c.log.Info("contract accepted", "mob", mobID, "contract", contractID)
	s.sendInventory(c)
}

// tryAcceptContract делает всю работу под одним s.mobs.Lock():
// валидация моба → оплата → установка контракта. ok=false, если что-то не так.
func (s *Server) tryAcceptContract(c *Client, mobID, contractID string) bool {
	s.mobs.Lock()
	defer s.mobs.Unlock()

	m, ok := s.mobs.Map()[mobID]
	if !ok {
		c.log.Warn("contract: mob not found")
		return false
	}
	if m.Kind != "pink" {
		c.log.Warn("contract: not a pink mob")
		return false
	}
	if m.Contract != "" {
		c.log.Warn("contract: mob already busy", "existing", m.Contract)
		return false
	}
	if contractID != "gather4" && contractID != "guard" {
		c.log.Warn("contract: unknown id", "id", contractID)
		return false
	}
	if !payContractPrice(c, contractID) {
		return false
	}
	applyContract(m, c.ID, contractID)
	return true
}

// payContractPrice списывает с клиента цену контракта.
//
//	gather4 → 1 fruit
//	guard   → 10 spear
func payContractPrice(c *Client, contractID string) bool {
	switch contractID {
	case "gather4":
		if !c.consumeItem("fruit") {
			c.log.Warn("contract: no fruit")
			return false
		}
	case "guard":
		c.mu.Lock()
		defer c.mu.Unlock()
		if c.inventory["spear"] < 10 {
			c.log.Warn("contract: not enough spears")
			return false
		}
		c.inventory["spear"] -= 10
	}
	return true
}

// applyContract устанавливает мобу контракт и владельца.
// Для gather4 сбрасывает инвентарь моба (начинает сбор заново).
func applyContract(m *Mob, ownerID, contractID string) {
	m.Contract = contractID
	m.OwnerID = ownerID
	if contractID == "gather4" {
		m.Inventory = make(map[string]int)
	}
}
