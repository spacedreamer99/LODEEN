package net

import (
	"github.com/spacedreamer99/lodeen/internal/shared/protocol"
)

// Этот файл содержит энергетику: солнечные панели, батареи, завод.
// Логика: панели заряжают ближайшие батареи, завод тратит энергию на крафт.

type Solar struct {
	ID  string
	Pos protocol.Vector3
	Yaw float32
}

type Battery struct {
	ID        string
	Pos       protocol.Vector3
	Yaw       float32
	Energy    int
	MaxEnergy int
}

type Factory struct {
	ID       string
	Pos      protocol.Vector3
	Yaw      float32
	Crafting string
	Progress int
}

const energyLinkRadiusD2 = 10.0 * 10.0
const solarEnergyPerSec = 5
const batteryMaxEnergy = 1000

// --- Place handlers ---

func (s *Server) handlePlaceSolar(c *Client, p protocol.PlaceSolar) {
	if !c.consumeItem("solar") {
		return
	}
	id := newID()
	s.solar.Lock()
	s.solar.Map()[id] = &Solar{ID: id, Pos: protocol.Vector3{X: p.X, Y: p.Y, Z: p.Z}, Yaw: p.Yaw}
	s.solar.Unlock()

	c.mu.Lock()
	inv := make(map[string]int, len(c.inventory))
	for k, v := range c.inventory {
		inv[k] = v
	}
	c.mu.Unlock()
	c.sendEnvelope(protocol.TypeInventoryUpdate, protocol.InventoryUpdate{Items: inv})
	c.log.Info("solar placed", "id", id)
}

func (s *Server) handlePlaceBattery(c *Client, p protocol.PlaceBattery) {
	if !c.consumeItem("battery") {
		return
	}
	id := newID()
	s.batteries.Lock()
	s.batteries.Map()[id] = &Battery{
		ID: id, Pos: protocol.Vector3{X: p.X, Y: p.Y, Z: p.Z},
		Yaw: p.Yaw, Energy: 0, MaxEnergy: batteryMaxEnergy,
	}
	s.batteries.Unlock()

	c.mu.Lock()
	inv := make(map[string]int, len(c.inventory))
	for k, v := range c.inventory {
		inv[k] = v
	}
	c.mu.Unlock()
	c.sendEnvelope(protocol.TypeInventoryUpdate, protocol.InventoryUpdate{Items: inv})
	c.log.Info("battery placed", "id", id)
}

func (s *Server) handlePlaceFactory(c *Client, p protocol.PlaceFactory) {
	if !c.consumeItem("factory") {
		return
	}
	id := newID()
	s.factories.Lock()
	s.factories.Map()[id] = &Factory{ID: id, Pos: protocol.Vector3{X: p.X, Y: p.Y, Z: p.Z}, Yaw: p.Yaw}
	s.factories.Unlock()

	c.mu.Lock()
	inv := make(map[string]int, len(c.inventory))
	for k, v := range c.inventory {
		inv[k] = v
	}
	c.mu.Unlock()
	c.sendEnvelope(protocol.TypeInventoryUpdate, protocol.InventoryUpdate{Items: inv})
	c.log.Info("factory placed", "id", id)
}

// --- Factory recipes ---

type factoryRecipe struct {
	need   map[string]int
	energy int
	out    string
	outQty int
}

var factoryRecipes = map[string]factoryRecipe{
	"steel":   {need: map[string]int{"ore": 5}, energy: 10, out: "steel", outQty: 1},
	"gear":    {need: map[string]int{"stone": 2, "wood": 2}, energy: 20, out: "gear", outQty: 1},
	"circuit": {need: map[string]int{"ore": 3, "liana": 1}, energy: 50, out: "circuit", outQty: 1},
	"drone":   {need: map[string]int{"gear": 1, "circuit": 1}, energy: 100, out: "drone", outQty: 1},
	"rocket":  {need: map[string]int{"circuit": 3, "steel": 5, "gear": 2}, energy: 500, out: "rocket", outQty: 1},
}

// nearestBatteryLocked ищет ближайшую батарею (без блокировки — вызывать под s.batteries.Lock).
func (s *Server) nearestBatteryLocked(pos protocol.Vector3) *Battery {
	var best *Battery
	bestD2 := float32(energyLinkRadiusD2)
	for _, b := range s.batteries.Map() {
		dx := b.Pos.X - pos.X
		dy := b.Pos.Y - pos.Y
		dz := b.Pos.Z - pos.Z
		d2 := dx*dx + dy*dy + dz*dz
		if d2 < bestD2 {
			best = b
			bestD2 = d2
		}
	}
	return best
}

func (s *Server) handleOpenFactory(c *Client, factoryID string) {
	s.factories.RLock()
	_, ok := s.factories.Map()[factoryID]
	s.factories.RUnlock()
	if !ok {
		c.log.Warn("factory: not found")
		return
	}
	c.log.Info("factory opened", "id", factoryID)
}

// handleCraftFactory — оркестратор крафта на фабрике:
// валидация → списание энергии → списание ресурсов + выдача → inventory update.
func (s *Server) handleCraftFactory(c *Client, factoryID, recipeID string) {
	f, r, ok := s.lookupFactoryAndRecipe(c, factoryID, recipeID)
	if !ok {
		return
	}
	if !s.consumeBatteryEnergy(c, f.Pos, r.energy) {
		return
	}
	inv, ok := s.consumeAndGrant(c, r)
	if !ok {
		return
	}
	c.sendEnvelope(protocol.TypeInventoryUpdate, protocol.InventoryUpdate{Items: inv})
	c.log.Info("factory crafted", "id", factoryID, "recipe", recipeID, "out", r.out)
}

// lookupFactoryAndRecipe достаёт фабрику и рецепт.
// Возвращает ok=false, если что-то не найдено или ресурсов не хватает.
func (s *Server) lookupFactoryAndRecipe(c *Client, factoryID, recipeID string) (*Factory, factoryRecipe, bool) {
	s.factories.RLock()
	f, ok := s.factories.Map()[factoryID]
	s.factories.RUnlock()
	if !ok {
		c.log.Warn("factory craft: not found")
		return nil, factoryRecipe{}, false
	}

	r, ok := factoryRecipes[recipeID]
	if !ok {
		c.log.Warn("factory craft: unknown recipe", "id", recipeID)
		return nil, factoryRecipe{}, false
	}

	if !clientHasResources(c, r.need) {
		c.log.Warn("factory craft: not enough resources")
		return nil, factoryRecipe{}, false
	}
	return f, r, true
}

// clientHasResources проверяет, что у клиента есть все ресурсы для рецепта.
func clientHasResources(c *Client, need map[string]int) bool {
	c.mu.Lock()
	defer c.mu.Unlock()
	for item, q := range need {
		if c.inventory[item] < q {
			return false
		}
	}
	return true
}

// consumeBatteryEnergy списывает энергию с ближайшей к фабрике батареи.
// Возвращает false, если батареи нет или энергии не хватает.
func (s *Server) consumeBatteryEnergy(c *Client, factoryPos protocol.Vector3, energy int) bool {
	s.batteries.Lock()
	defer s.batteries.Unlock()

	b := s.nearestBatteryLocked(factoryPos)
	if b == nil {
		c.log.Warn("factory craft: no battery in range")
		return false
	}
	if b.Energy < energy {
		c.log.Warn("factory craft: not enough energy",
			"have", b.Energy, "need", energy)
		return false
	}
	b.Energy -= energy
	return true
}

// consumeAndGrant списывает ресурсы рецепта и кладёт выход в инвентарь.
// Возвращает актуальный inventory snapshot для отправки клиенту.
// ok=false, если ресурсов не хватило (race с другим действием).
func (s *Server) consumeAndGrant(c *Client, r factoryRecipe) (map[string]int, bool) {
	c.mu.Lock()
	defer c.mu.Unlock()

	for item, q := range r.need {
		if c.inventory[item] < q {
			return nil, false
		}
	}
	for item, q := range r.need {
		c.inventory[item] -= q
		if c.inventory[item] <= 0 {
			delete(c.inventory, item)
		}
	}
	c.inventory[r.out] += r.outQty

	inv := make(map[string]int, len(c.inventory))
	for k, v := range c.inventory {
		inv[k] = v
	}
	return inv, true
}

// tickEnergy — панели заряжают ближайшие батареи.
func (s *Server) tickEnergy(dt float32) {
	// Снимок панелей.
	s.solar.RLock()
	panels := make([]*Solar, 0, len(s.solar.Map()))
	for _, p := range s.solar.Map() {
		panels = append(panels, p)
	}
	s.solar.RUnlock()

	if len(panels) == 0 {
		return
	}

	// Сколько энергии даёт одна панель за этот тик.
	gain := float32(solarEnergyPerSec) * dt
	energyPerPanel := int(gain * 100) // фиксированная точка: gain*100, чтобы копить копейки

	s.batteries.Lock()
	defer s.batteries.Unlock()
	for _, p := range panels {
		b := s.nearestBatteryLocked(p.Pos)
		if b == nil {
			continue
		}
		// Накапливаем «копейки» в отдельном поле — упрощаю: прибавляем целое.
		b.Energy += energyPerPanel / 100
		if b.Energy > b.MaxEnergy {
			b.Energy = b.MaxEnergy
		}
	}
}
