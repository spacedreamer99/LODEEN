package protocol

import "encoding/json"

const (
	// PlanetRadius — радиус планеты в игровых единицах.
	PlanetRadius float32 = 50.0
	// PlayerHeight — расстояние от поверхности до «глаз» игрока.
	PlayerHeight float32 = 1.5
)

type Type string

const (
	TypeHello           Type = "hello"
	TypeWelcome         Type = "welcome"
	TypeState           Type = "state"
	TypeSnapshot        Type = "snapshot"
	TypeChat            Type = "chat"
	TypePing            Type = "ping"
	TypePong            Type = "pong"
	TypePickupItem      Type = "pickup_item"
	TypeInventoryUpdate Type = "inventory_update"
	TypeEatFruit        Type = "eat_fruit"
	TypeThrowSpear      Type = "throw_spear"
	TypeCraftItem       Type = "craft_item"
	TypeHitMammoth      Type = "hit_mammoth"
)

type Envelope struct {
	Type Type            `json:"type"`
	Data json.RawMessage `json:"data,omitempty"`
}

func NewEnvelope(t Type, data any) (*Envelope, error) {
	raw, err := json.Marshal(data)
	if err != nil {
		return nil, err
	}
	return &Envelope{Type: t, Data: raw}, nil
}

func (e *Envelope) Decode(v any) error {
	if len(e.Data) == 0 {
		return nil
	}
	return json.Unmarshal(e.Data, v)
}

type Hello struct {
	Nick    string `json:"nick"`
	Version string `json:"version"`
}

type Welcome struct {
	PlayerID      string `json:"player_id"`
	WorldName     string `json:"world_name"`
	TickRate      int    `json:"tick_rate"`
	ServerVersion string `json:"server_version"`
}

type PlayerState struct {
	ID     string  `json:"id"`
	Nick   string  `json:"nick"`
	X      float32 `json:"x"`
	Y      float32 `json:"y"`
	Z      float32 `json:"z"`
	Yaw    float32 `json:"yaw"`
	Pitch  float32 `json:"pitch"`
	Hunger float32 `json:"hunger"`
}

type Resource struct {
	ID   string  `json:"id"`
	Type string  `json:"type"`
	X    float32 `json:"x"`
	Y    float32 `json:"y"`
	Z    float32 `json:"z"`
}

type Mammoth struct {
	ID string  `json:"id"`
	X  float32 `json:"x"`
	Y  float32 `json:"y"`
	Z  float32 `json:"z"`
	HP int     `json:"hp"`
}

type Snapshot struct {
	Tick      uint64        `json:"tick"`
	Players   []PlayerState `json:"players"`
	Resources []Resource    `json:"resources"`
	Mammoths  []Mammoth     `json:"mammoths"`
}

type ChatMessage struct {
	From string `json:"from"`
	Text string `json:"text"`
	TS   int64  `json:"ts"`
}

type Ping struct {
	Sent int64 `json:"sent"`
}

type Pong struct {
	Sent     int64 `json:"sent"`
	ServerTS int64 `json:"server_ts"`
}

type PickupItem struct {
	ResourceID string `json:"resource_id"`
}

type InventoryUpdate struct {
	Items map[string]int `json:"items"`
}

type EatFruit struct{}

type Vector3 struct {
	X float32 `json:"x"`
	Y float32 `json:"y"`
	Z float32 `json:"z"`
}

type ThrowSpear struct {
	Dir Vector3 `json:"dir"`
}

type CraftItem struct {
	Recipe string `json:"recipe"`
}

type HitMammoth struct {
	MammothID string `json:"mammoth_id"`
}
