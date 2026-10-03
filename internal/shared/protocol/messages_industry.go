package protocol

// --- Солнечные панели и батареи ---

type Solar struct {
	ID  string  `json:"id"`
	X   float32 `json:"x"`
	Y   float32 `json:"y"`
	Z   float32 `json:"z"`
	Yaw float32 `json:"yaw"`
}

type PlaceSolar struct {
	X   float32 `json:"x"`
	Y   float32 `json:"y"`
	Z   float32 `json:"z"`
	Yaw float32 `json:"yaw"`
}

type Battery struct {
	ID        string  `json:"id"`
	X         float32 `json:"x"`
	Y         float32 `json:"y"`
	Z         float32 `json:"z"`
	Yaw       float32 `json:"yaw"`
	Energy    int     `json:"energy"`
	MaxEnergy int     `json:"max_energy"`
}

type PlaceBattery struct {
	X   float32 `json:"x"`
	Y   float32 `json:"y"`
	Z   float32 `json:"z"`
	Yaw float32 `json:"yaw"`
}

// --- Фабрики ---

type Factory struct {
	ID       string  `json:"id"`
	X        float32 `json:"x"`
	Y        float32 `json:"y"`
	Z        float32 `json:"z"`
	Yaw      float32 `json:"yaw"`
	Crafting string  `json:"crafting,omitempty"` // какой рецепт в работе
	Progress int     `json:"progress,omitempty"` // 0..100
}

type PlaceFactory struct {
	X   float32 `json:"x"`
	Y   float32 `json:"y"`
	Z   float32 `json:"z"`
	Yaw float32 `json:"yaw"`
}

type OpenFactory struct {
	FactoryID string `json:"factory_id"`
}

type CraftFactory struct {
	FactoryID string `json:"factory_id"`
	Recipe    string `json:"recipe"`
}
