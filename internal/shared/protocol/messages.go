package protocol

import "encoding/json"

// Type — тег типа сообщения в Envelope.
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
	TypePlantSeed       Type = "plant_seed"
	TypeWaterPlant      Type = "water_plant"
	TypeTakeWater       Type = "take_water"
	TypeTameMammoth     Type = "tame_mammoth"
	TypeSelectItem      Type = "select_item"
	TypeLeashMammoth    Type = "leash_mammoth"
	TypePlaceHouse      Type = "place_house"
	TypePlaceSolar      Type = "place_solar"
	TypePlaceBattery    Type = "place_battery"
	TypePlaceFactory    Type = "place_factory"
	TypeToggleDoor      Type = "toggle_door"
	TypeSaddleMammoth   Type = "saddle_mammoth"
	TypeRideMammoth     Type = "ride_mammoth"
	TypePlaceBoat       Type = "place_boat"
	TypeEnterBoat       Type = "enter_boat"
	TypeHitMob          Type = "hit_mob"
	TypeAcceptContract  Type = "accept_contract"
	TypeOpenFactory     Type = "open_factory"
	TypeCraftFactory    Type = "craft_factory"
	TypePlaceRocket     Type = "place_rocket"
	TypeBoardRocket     Type = "board_rocket"
	TypeExitRocket      Type = "exit_rocket"
	TypeRocketInput     Type = "rocket_input"
	TypeTeleport        Type = "teleport"
)

// Envelope — обёртка любого сообщения: type + произвольный JSON-payload.
type Envelope struct {
	Type Type            `json:"type"`
	Data json.RawMessage `json:"data,omitempty"`
}

// NewEnvelope собирает Envelope, сериализуя payload.
func NewEnvelope(t Type, data any) (*Envelope, error) {
	raw, err := json.Marshal(data)
	if err != nil {
		return nil, err
	}
	return &Envelope{Type: t, Data: raw}, nil
}

// Decode десериализует payload в переданный указатель.
func (e *Envelope) Decode(v any) error {
	if len(e.Data) == 0 {
		return nil
	}
	return json.Unmarshal(e.Data, v)
}
