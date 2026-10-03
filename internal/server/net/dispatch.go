package net

import (
	"github.com/spacedreamer99/lodeen/internal/shared/protocol"
)

// msgHandler — обработчик одного типа входящего сообщения.
type msgHandler func(s *Server, c *Client, env *protocol.Envelope) error

// dispatch — оборачивает типизированный handler в msgHandler,
// декодируя payload в T перед вызовом.
func dispatch[T any](h func(s *Server, c *Client, p T)) msgHandler {
	return func(s *Server, c *Client, env *protocol.Envelope) error {
		var p T
		if err := env.Decode(&p); err != nil {
			return err
		}
		h(s, c, p)
		return nil
	}
}

// messageHandlers — таблица маршрутизации по типу входящего сообщения.
// Регистрируем здесь ВСЕ поддерживаемые типы: name→handler.
var messageHandlers = map[protocol.Type]msgHandler{
	// Специальные (нестандартная логика).
	protocol.TypeState:      handleStateMsg,
	protocol.TypeChat:       handleChatMsg,
	protocol.TypePing:       handlePingMsg,
	protocol.TypeEatFruit:   handleEatFruitMsg,
	protocol.TypeSelectItem: handleSelectItemMsg,
	protocol.TypeExitRocket: handleExitRocketMsg,

	// Единообразные: decode + проброс в domain-метод.
	protocol.TypePickupItem: dispatch(func(s *Server, c *Client, p protocol.PickupItem) {
		s.handlePickup(c, p.ResourceID)
	}),
	protocol.TypeThrowSpear: dispatch(func(s *Server, c *Client, p protocol.ThrowSpear) {
		s.handleThrowSpear(c, p.Dir)
	}),
	protocol.TypeCraftItem: dispatch(func(s *Server, c *Client, p protocol.CraftItem) {
		s.handleCraft(c, p.Recipe)
	}),
	protocol.TypeHitMammoth: dispatch(func(s *Server, c *Client, p protocol.HitMammoth) {
		s.handleHitMammoth(c, p.MammothID)
	}),
	protocol.TypePlantSeed: dispatch(func(s *Server, c *Client, p protocol.PlantSeed) {
		s.handlePlantSeed(c, p)
	}),
	protocol.TypeWaterPlant: dispatch(func(s *Server, c *Client, p protocol.WaterPlant) {
		s.handleWaterPlant(c, p.ResourceID)
	}),
	protocol.TypeTakeWater: dispatch(func(s *Server, c *Client, p protocol.TakeWater) {
		s.handleTakeWater(c, p.WellID)
	}),
	protocol.TypeTameMammoth: dispatch(func(s *Server, c *Client, p protocol.TameMammoth) {
		s.handleTameMammoth(c, p.MammothID)
	}),
	protocol.TypeLeashMammoth: dispatch(func(s *Server, c *Client, p protocol.LeashMammoth) {
		c.log.Info("leash packet received", "id", p.MammothID)
		s.handleLeashMammoth(c, p.MammothID)
	}),
	protocol.TypePlaceHouse: dispatch(func(s *Server, c *Client, p protocol.PlaceHouse) {
		s.handlePlaceHouse(c, p)
	}),
	protocol.TypeToggleDoor: dispatch(func(s *Server, c *Client, p protocol.ToggleDoor) {
		s.handleToggleDoor(c, p.HouseID)
	}),
	protocol.TypeSaddleMammoth: dispatch(func(s *Server, c *Client, p protocol.SaddleMammoth) {
		s.handleSaddleMammoth(c, p.MammothID)
	}),
	protocol.TypeRideMammoth: dispatch(func(s *Server, c *Client, p protocol.RideMammoth) {
		s.handleRideMammoth(c, p.MammothID)
	}),
	protocol.TypePlaceBoat: dispatch(func(s *Server, c *Client, p protocol.PlaceBoat) {
		s.handlePlaceBoat(c, p)
	}),
	protocol.TypeEnterBoat: dispatch(func(s *Server, c *Client, p protocol.EnterBoat) {
		s.handleEnterBoat(c, p.BoatID)
	}),
	protocol.TypeHitMob: dispatch(func(s *Server, c *Client, p protocol.HitMob) {
		s.handleHitMob(c, p.MobID)
	}),
	protocol.TypeAcceptContract: dispatch(func(s *Server, c *Client, p protocol.AcceptContract) {
		s.handleAcceptContract(c, p.MobID, p.ContractID)
	}),
	protocol.TypePlaceSolar: dispatch(func(s *Server, c *Client, p protocol.PlaceSolar) {
		s.handlePlaceSolar(c, p)
	}),
	protocol.TypePlaceBattery: dispatch(func(s *Server, c *Client, p protocol.PlaceBattery) {
		s.handlePlaceBattery(c, p)
	}),
	protocol.TypePlaceFactory: dispatch(func(s *Server, c *Client, p protocol.PlaceFactory) {
		s.handlePlaceFactory(c, p)
	}),
	protocol.TypeOpenFactory: dispatch(func(s *Server, c *Client, p protocol.OpenFactory) {
		s.handleOpenFactory(c, p.FactoryID)
	}),
	protocol.TypeCraftFactory: dispatch(func(s *Server, c *Client, p protocol.CraftFactory) {
		s.handleCraftFactory(c, p.FactoryID, p.Recipe)
	}),
	protocol.TypePlaceRocket: dispatch(func(s *Server, c *Client, p protocol.PlaceRocket) {
		s.handlePlaceRocket(c, p)
	}),
	protocol.TypeBoardRocket: dispatch(func(s *Server, c *Client, p protocol.BoardRocket) {
		s.handleBoardRocket(c, p.RocketID)
	}),
	protocol.TypeRocketInput: dispatch(func(s *Server, c *Client, p protocol.RocketInput) {
		s.handleRocketInput(c, p)
	}),
}

// handleMessage — маршрутизирует входящее сообщение по таблице.
// Неизвестные типы тихо игнорируются (как и раньше).
func (s *Server) handleMessage(c *Client, env *protocol.Envelope) {
	if h, ok := messageHandlers[env.Type]; ok {
		_ = h(s, c, env)
	}
}
