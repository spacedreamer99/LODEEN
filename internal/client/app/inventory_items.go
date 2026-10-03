package app

import (
	rl "github.com/gen2brain/raylib-go/raylib"
)

func itemColor(typ string) rl.Color {
	switch typ {
	case "stone":
		return rl.NewColor(140, 140, 150, 255)
	case "water":
		return rl.NewColor(60, 120, 220, 255)
	case "liana":
		return rl.NewColor(80, 160, 60, 255)
	case "leash":
		return rl.NewColor(160, 120, 70, 255)
	case "house":
		return rl.NewColor(140, 95, 55, 255)
	case "saddle":
		return rl.NewColor(90, 60, 40, 255)
	case "boat":
		return rl.NewColor(120, 80, 40, 255)
	case "solar":
		return rl.NewColor(40, 80, 140, 255)
	case "battery":
		return rl.NewColor(60, 60, 80, 255)
	case "factory":
		return rl.NewColor(110, 100, 90, 255)
	case "steel":
		return rl.NewColor(180, 180, 190, 255)
	case "gear":
		return rl.NewColor(160, 160, 130, 255)
	case "circuit":
		return rl.NewColor(80, 200, 120, 255)
	case "drone":
		return rl.NewColor(100, 150, 220, 255)
	case "rocket":
		return rl.NewColor(220, 80, 80, 255)
	case "wood":
		return rl.NewColor(120, 80, 40, 255)
	case "ore":
		return rl.NewColor(200, 170, 60, 255)
	case "fruit":
		return rl.NewColor(230, 70, 70, 255)
	case "meat":
		return rl.NewColor(200, 100, 100, 255)
	case "spear":
		return rl.NewColor(180, 160, 120, 255)
	case "torch":
		return rl.NewColor(240, 180, 80, 255)
	}
	return rl.White
}

func itemName(typ string) string {
	switch typ {
	case "stone":
		return "Stone"
	case "wood":
		return "Wood"
	case "ore":
		return "Ore"
	case "fruit":
		return "Fruit"
	case "meat":
		return "Meat"
	case "spear":
		return "Spear"
	case "torch":
		return "Torch"
	case "water":
		return "Water"
	case "liana":
		return "Liana"
	case "leash":
		return "Leash"
	case "house":
		return "House"
	case "saddle":
		return "Saddle"
	case "boat":
		return "Boat"
	case "solar":
		return "Solar Panel"
	case "battery":
		return "Battery"
	case "factory":
		return "Factory"
	case "steel":
		return "Steel"
	case "gear":
		return "Gear"
	case "circuit":
		return "Circuit"
	case "drone":
		return "Drone"
	case "rocket":
		return "Rocket"
	}
	return typ
}
