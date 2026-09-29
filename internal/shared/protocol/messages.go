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
	HP     int     `json:"hp"`
}

type Resource struct {
	ID      string  `json:"id"`
	Type    string  `json:"type"`
	X       float32 `json:"x"`
	Y       float32 `json:"y"`
	Z       float32 `json:"z"`
	Watered bool    `json:"watered,omitempty"`
	GrowAt  int64   `json:"grow_at,omitempty"`
}

type Mammoth struct {
	ID        string  `json:"id"`
	X         float32 `json:"x"`
	Y         float32 `json:"y"`
	Z         float32 `json:"z"`
	HP        int     `json:"hp"`
	Tamed     bool    `json:"tamed,omitempty"`
	Sex       string  `json:"sex,omitempty"`
	FedCount  int     `json:"fed_count,omitempty"`
	Baby      bool    `json:"baby,omitempty"`
	LeashedTo string  `json:"leashed_to,omitempty"`
	Saddle    bool    `json:"saddle,omitempty"`
	RiderID   string  `json:"rider_id,omitempty"`
}

type Snapshot struct {
	Tick        uint64          `json:"tick"`
	Players     []PlayerState   `json:"players"`
	Resources   []Resource      `json:"resources"`
	Mammoths    []Mammoth       `json:"mammoths"`
	Wells       []Well          `json:"wells"`
	Houses      []House         `json:"houses"`
	Boats       []Boat          `json:"boats"`
	Mobs        []Mob           `json:"mobs"`
	Projectiles []MobProjectile `json:"projectiles"`
	Solar       []Solar         `json:"solar"`
	Batteries   []Battery       `json:"batteries"`
	Factories   []Factory       `json:"factories"`
	Rockets     []Rocket        `json:"rockets"`
}

type Rocket struct {
	ID        string  `json:"id"`
	X         float32 `json:"x"`
	Y         float32 `json:"y"`
	Z         float32 `json:"z"`
	DX        float32 `json:"dx"`
	DY        float32 `json:"dy"`
	DZ        float32 `json:"dz"`
	Fuel      int     `json:"fuel"`
	MaxFuel   int     `json:"max_fuel"`
	Piloted   bool    `json:"piloted"`
	OwnerID   string  `json:"owner_id,omitempty"`
	InOrbit   bool    `json:"in_orbit,omitempty"`
	Apoapsis  float32 `json:"apoapsis,omitempty"`  // высота апогея, м
	Periapsis float32 `json:"periapsis,omitempty"` // высота перигея, м
	Speed     float32 `json:"speed,omitempty"`     // |v| в м/с
	Altitude  float32 `json:"altitude,omitempty"`  // высота над поверхностью
	TargetVelocity float32 `json:"target_velocity,omitempty"`
	VX        float32 `json:"vx,omitempty"`
	VY        float32 `json:"vy,omitempty"`
	VZ        float32 `json:"vz,omitempty"`
}

type Solar struct {
	ID  string  `json:"id"`
	X   float32 `json:"x"`
	Y   float32 `json:"y"`
	Z   float32 `json:"z"`
	Yaw float32 `json:"yaw"`
}

type Battery struct {
	ID       string  `json:"id"`
	X        float32 `json:"x"`
	Y        float32 `json:"y"`
	Z        float32 `json:"z"`
	Yaw      float32 `json:"yaw"`
	Energy   int     `json:"energy"`
	MaxEnergy int    `json:"max_energy"`
}

type Factory struct {
	ID       string  `json:"id"`
	X        float32 `json:"x"`
	Y        float32 `json:"y"`
	Z        float32 `json:"z"`
	Yaw      float32 `json:"yaw"`
	Crafting string  `json:"crafting,omitempty"` // какой рецепт в работе
	Progress int     `json:"progress,omitempty"` // 0..100
}

type MobProjectile struct {
	ID string  `json:"id"`
	X  float32 `json:"x"`
	Y  float32 `json:"y"`
	Z  float32 `json:"z"`
	DX float32 `json:"dx"`
	DY float32 `json:"dy"`
	DZ float32 `json:"dz"`
}

type HitMob struct {
	MobID string `json:"mob_id"`
}

type AcceptContract struct {
	MobID      string `json:"mob_id"`
	ContractID string `json:"contract_id"` // "gather4" | "guard"
}

type OpenFactory struct {
	FactoryID string `json:"factory_id"`
}

type CraftFactory struct {
	FactoryID string `json:"factory_id"`
	Recipe    string `json:"recipe"`
}

type PlaceRocket struct {
	X   float32 `json:"x"`
	Y   float32 `json:"y"`
	Z   float32 `json:"z"`
}

type BoardRocket struct {
	RocketID string `json:"rocket_id"`
}

type ExitRocket struct{}

type RocketInput struct {
	Thrust    float32 `json:"thrust"`
	TargetUpX float32 `json:"target_up_x"`
	TargetUpY float32 `json:"target_up_y"`
	TargetUpZ float32 `json:"target_up_z"`
	AutoPilot string  `json:"auto_pilot,omitempty"` // "" | prograde | retrograde | radial_out | radial_in | normal | antinormal
}

type Mob struct {
	ID       string  `json:"id"`
	X        float32 `json:"x"`
	Y        float32 `json:"y"`
	Z        float32 `json:"z"`
	HP       int     `json:"hp"`
	Kind     string  `json:"kind"`
	Contract string  `json:"contract,omitempty"`
	OwnerID  string  `json:"owner_id,omitempty"`
}

type Boat struct {
	ID      string  `json:"id"`
	X       float32 `json:"x"`
	Y       float32 `json:"y"`
	Z       float32 `json:"z"`
	Yaw     float32 `json:"yaw"`
	RiderID string  `json:"rider_id,omitempty"`
}

type PlaceBoat struct {
	X   float32 `json:"x"`
	Y   float32 `json:"y"`
	Z   float32 `json:"z"`
	Yaw float32 `json:"yaw"`
}

type EnterBoat struct {
	BoatID string `json:"boat_id"`
}

type House struct {
	ID       string  `json:"id"`
	X        float32 `json:"x"`
	Y        float32 `json:"y"`
	Z        float32 `json:"z"`
	Yaw      float32 `json:"yaw"`
	DoorOpen bool    `json:"door_open,omitempty"`
}

type PlaceHouse struct {
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

type PlaceBattery struct {
	X   float32 `json:"x"`
	Y   float32 `json:"y"`
	Z   float32 `json:"z"`
	Yaw float32 `json:"yaw"`
}

type PlaceFactory struct {
	X   float32 `json:"x"`
	Y   float32 `json:"y"`
	Z   float32 `json:"z"`
	Yaw float32 `json:"yaw"`
}

type ToggleDoor struct {
	HouseID string `json:"house_id"`
}

type SaddleMammoth struct {
	MammothID string `json:"mammoth_id"`
}

type RideMammoth struct {
	MammothID string `json:"mammoth_id"`
}

type Well struct {
	ID string  `json:"id"`
	X  float32 `json:"x"`
	Y  float32 `json:"y"`
	Z  float32 `json:"z"`
}

type TakeWater struct {
	WellID string `json:"well_id"`
}

type TameMammoth struct {
	MammothID string `json:"mammoth_id"`
}

type SelectItem struct {
	Item string `json:"item"`
}

type LeashMammoth struct {
	MammothID string `json:"mammoth_id"`
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

type PlantSeed struct {
	X float32 `json:"x"`
	Y float32 `json:"y"`
	Z float32 `json:"z"`
}

type WaterPlant struct {
	ResourceID string `json:"resource_id"`
}
