package net

import (
	"time"

	"github.com/spacedreamer99/lodeen/internal/shared/protocol"
)

func (c *Client) ThrowSpear(dir protocol.Vector3) error {
	return c.send(protocol.TypeThrowSpear, protocol.ThrowSpear{Dir: dir})
}

func (c *Client) HitMammoth(id string) error {
	return c.send(protocol.TypeHitMammoth, protocol.HitMammoth{MammothID: id})
}

func (c *Client) CraftItem(recipe string) error {
	return c.send(protocol.TypeCraftItem, protocol.CraftItem{Recipe: recipe})
}

func (c *Client) EatFruit() error {
	return c.send(protocol.TypeEatFruit, protocol.EatFruit{})
}

func (c *Client) SetState(st protocol.PlayerState) {
	c.stateMu.Lock()
	c.lastState = st
	c.stateMu.Unlock()
}

func (c *Client) StartStateLoop() {
	c.wg.Add(1)
	go func() {
		defer c.wg.Done()
		t := time.NewTicker(time.Second / 20)
		defer t.Stop()
		for {
			select {
			case <-t.C:
				c.stateMu.Lock()
				st := c.lastState
				c.stateMu.Unlock()
				st.RTTms = int32(c.RTT().Milliseconds())
				_ = c.send(protocol.TypeState, st)
			case <-c.done:
				return
			}
		}
	}()
}

func (c *Client) PickupItem(resourceID string) error {
	return c.send(protocol.TypePickupItem, protocol.PickupItem{ResourceID: resourceID})
}

func (c *Client) PlantSeed(x, y, z float32) error {
	return c.send(protocol.TypePlantSeed, protocol.PlantSeed{X: x, Y: y, Z: z})
}

func (c *Client) WaterPlant(resourceID string) error {
	return c.send(protocol.TypeWaterPlant, protocol.WaterPlant{ResourceID: resourceID})
}

func (c *Client) TakeWater(wellID string) error {
	return c.send(protocol.TypeTakeWater, protocol.TakeWater{WellID: wellID})
}

func (c *Client) HitMob(id string) error {
	return c.send(protocol.TypeHitMob, protocol.HitMob{MobID: id})
}

func (c *Client) AcceptContract(mobID, contractID string) error {
	return c.send(protocol.TypeAcceptContract, protocol.AcceptContract{
		MobID: mobID, ContractID: contractID,
	})
}

func (c *Client) TameMammoth(id string) error {
	return c.send(protocol.TypeTameMammoth, protocol.TameMammoth{MammothID: id})
}

func (c *Client) SelectItem(item string) error {
	return c.send(protocol.TypeSelectItem, protocol.SelectItem{Item: item})
}

func (c *Client) LeashMammoth(id string) error {
	return c.send(protocol.TypeLeashMammoth, protocol.LeashMammoth{MammothID: id})
}

func (c *Client) PlaceHouse(x, y, z, yaw float32) error {
	return c.send(protocol.TypePlaceHouse, protocol.PlaceHouse{X: x, Y: y, Z: z, Yaw: yaw})
}

func (c *Client) ToggleDoor(houseID string) error {
	return c.send(protocol.TypeToggleDoor, protocol.ToggleDoor{HouseID: houseID})
}

func (c *Client) SaddleMammoth(id string) error {
	return c.send(protocol.TypeSaddleMammoth, protocol.SaddleMammoth{MammothID: id})
}

func (c *Client) PlaceBoat(x, y, z, yaw float32) error {
	return c.send(protocol.TypePlaceBoat, protocol.PlaceBoat{X: x, Y: y, Z: z, Yaw: yaw})
}

func (c *Client) EnterBoat(boatID string) error {
	return c.send(protocol.TypeEnterBoat, protocol.EnterBoat{BoatID: boatID})
}

func (c *Client) PlaceRocket(x, y, z float32) error {
	return c.send(protocol.TypePlaceRocket, protocol.PlaceRocket{X: x, Y: y, Z: z})
}

func (c *Client) BoardRocket(id string) error {
	return c.send(protocol.TypeBoardRocket, protocol.BoardRocket{RocketID: id})
}

func (c *Client) ExitRocket() error {
	return c.send(protocol.TypeExitRocket, protocol.ExitRocket{})
}

func (c *Client) RocketInput(thrust, upX, upY, upZ float32, autoPilot string) error {
	return c.send(protocol.TypeRocketInput, protocol.RocketInput{
		Thrust:    thrust,
		TargetUpX: upX, TargetUpY: upY, TargetUpZ: upZ,
		AutoPilot: autoPilot,
	})
}

func (c *Client) PlaceSolar(x, y, z, yaw float32) error {
	return c.send(protocol.TypePlaceSolar, protocol.PlaceSolar{X: x, Y: y, Z: z, Yaw: yaw})
}

func (c *Client) PlaceBattery(x, y, z, yaw float32) error {
	return c.send(protocol.TypePlaceBattery, protocol.PlaceBattery{X: x, Y: y, Z: z, Yaw: yaw})
}

func (c *Client) PlaceFactory(x, y, z, yaw float32) error {
	return c.send(protocol.TypePlaceFactory, protocol.PlaceFactory{X: x, Y: y, Z: z, Yaw: yaw})
}

func (c *Client) OpenFactory(id string) error {
	return c.send(protocol.TypeOpenFactory, protocol.OpenFactory{FactoryID: id})
}

func (c *Client) CraftFactory(factoryID, recipeID string) error {
	return c.send(protocol.TypeCraftFactory, protocol.CraftFactory{FactoryID: factoryID, Recipe: recipeID})
}

func (c *Client) RideMammoth(id string) error {
	return c.send(protocol.TypeRideMammoth, protocol.RideMammoth{MammothID: id})
}

func (c *Client) SendChat(text string) error {
	return c.send(protocol.TypeChat, protocol.ChatMessage{Text: text})
}
