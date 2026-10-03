package net

import (
	"strings"
	"time"

	"github.com/spacedreamer99/lodeen/internal/shared/protocol"
)

// handleChatMsg — обычный чат + чит-команды (game-dev shortcuts).
func handleChatMsg(s *Server, c *Client, env *protocol.Envelope) error {
	var cm protocol.ChatMessage
	if err := env.Decode(&cm); err != nil {
		return err
	}
	if strings.HasPrefix(cm.Text, "/") && s.handleCheatCommand(c, cm.Text) {
		return nil
	}
	cm.From = c.Nick
	cm.TS = time.Now().UnixMilli()
	s.broadcast(protocol.TypeChat, cm)
	s.metrics.ChatMessages.Inc()
	return nil
}

// handleCheatCommand — пытается обработать чит-команду. true если обработал.
func (s *Server) handleCheatCommand(c *Client, text string) bool {
	switch text {
	case "/allinv":
		s.cheatAllInv(c)
	case "/infuel":
		s.cheatInfiniteFuel(c)
	case "/clearinv":
		s.cheatClearInv(c)
	case "/tp sun":
		s.cheatTeleport(c, protocol.SunPos, protocol.SunRadius, 200)
	case "/tp earth":
		s.cheatTeleport(c, s.world.EarthPos, protocol.PlanetRadius, 5)
	case "/tp star2":
		s.cheatTeleport(c, protocol.Star2Pos, protocol.Star2Radius, 200)
	case "/tp planet2":
		s.cheatTeleport(c, s.world.Planet2Pos, protocol.Planet2Radius, 5)
	default:
		if !strings.HasPrefix(text, "/get") {
			return false
		}
		s.cheatGet(c, text[len("/get"):])
	}
	return true
}

// cheatAllInv — выдать все предметы по 100.
func (s *Server) cheatAllInv(c *Client) {
	allItems := []string{
		"stone", "wood", "ore", "fruit", "meat", "spear", "torch",
		"water", "liana", "leash", "house", "saddle", "boat",
		"solar", "battery", "factory",
		"steel", "gear", "circuit", "drone", "rocket",
	}
	c.mu.Lock()
	for _, it := range allItems {
		c.inventory[it] = 100
	}
	inv := make(map[string]int, len(c.inventory))
	for k, v := range c.inventory {
		inv[k] = v
	}
	c.mu.Unlock()
	c.sendEnvelope(protocol.TypeInventoryUpdate, protocol.InventoryUpdate{Items: inv})
	c.log.Info("cheat: allinv", "count", len(allItems))
}

// cheatInfiniteFuel — toggle бесконечного топлива.
func (s *Server) cheatInfiniteFuel(c *Client) {
	c.mu.Lock()
	c.infiniteFuel = !c.infiniteFuel
	state := c.infiniteFuel
	c.mu.Unlock()

	msg := "infinite fuel: OFF"
	if state {
		msg = "infinite fuel: ON"
	}
	c.sendEnvelope(protocol.TypeChat, protocol.ChatMessage{
		From: "server", Text: msg, TS: time.Now().UnixMilli(),
	})
	c.log.Info("cheat: infuel", "state", state)
}

// cheatClearInv — очистить инвентарь.
func (s *Server) cheatClearInv(c *Client) {
	c.mu.Lock()
	c.inventory = make(map[string]int)
	inv := make(map[string]int)
	c.mu.Unlock()
	c.sendEnvelope(protocol.TypeInventoryUpdate, protocol.InventoryUpdate{Items: inv})
	c.log.Info("cheat: clearinv")
}

// parseGetArg парсит аргумент команды /get: "<item><qty>".
// Отделяет хвостовые цифры как количество, ограничивает 100000.
// Возвращает ok=false, если имя или количество пустые.
//
// Примеры:
//
//	"stone10"  -> ("stone", 10, true)
//	"a0"       -> ("a", 0, true)
//	"x9999999" -> ("x", 100000, true)  // clamp
//	"stone"    -> ("", 0, false)        // нет количества
//	"123"      -> ("", 0, false)        // нет имени
//	""         -> ("", 0, false)
func parseGetArg(rest string) (name string, qty int, ok bool) {
	i := len(rest)
	for i > 0 && rest[i-1] >= '0' && rest[i-1] <= '9' {
		i--
	}
	name = rest[:i]
	qtyStr := rest[i:]
	if name == "" || qtyStr == "" {
		return "", 0, false
	}
	for _, ch := range qtyStr {
		qty = qty*10 + int(ch-'0')
	}
	if qty > 100000 {
		qty = 100000
	}
	return name, qty, true
}

// cheatGet — /get<item><qty>, например /getfruit100.
func (s *Server) cheatGet(c *Client, rest string) {
	name, qty, ok := parseGetArg(rest)
	if !ok {
		c.sendEnvelope(protocol.TypeChat, protocol.ChatMessage{
			From: "server", Text: "usage: /get<item><qty>", TS: time.Now().UnixMilli(),
		})
		return
	}
	c.mu.Lock()
	c.inventory[name] += qty
	inv := make(map[string]int, len(c.inventory))
	for k, v := range c.inventory {
		inv[k] = v
	}
	c.mu.Unlock()
	c.sendEnvelope(protocol.TypeInventoryUpdate, protocol.InventoryUpdate{Items: inv})
	c.log.Info("cheat: get", "item", name, "qty", qty)
}

// cheatTeleport — общий helper для /tp: ставит игрока над телом на высоту bodyRadius+offset.
// Если offset > 100 — используется как «далеко от тела» (Sun/Star2),
// иначе — «на поверхность» (Earth/Planet2, +PlayerHeight).
func (s *Server) cheatTeleport(c *Client, bodyPos protocol.Vector3, bodyRadius, offset float32) {
	y := bodyPos.Y + bodyRadius + offset
	if offset < 100 {
		y += protocol.PlayerHeight
	}
	x := bodyPos.X
	z := bodyPos.Z
	c.setState(protocol.PlayerState{ID: c.ID, Nick: c.Nick, X: x, Y: y, Z: z})
	c.sendEnvelope(protocol.TypeTeleport, protocol.Teleport{X: x, Y: y, Z: z})
}
