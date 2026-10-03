package protocol

// --- Ресурсы и колодцы ---

type Resource struct {
	ID      string  `json:"id"`
	Type    string  `json:"type"`
	X       float32 `json:"x"`
	Y       float32 `json:"y"`
	Z       float32 `json:"z"`
	Watered bool    `json:"watered,omitempty"`
	GrowAt  int64   `json:"grow_at,omitempty"`
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

// --- Растения ---

type PlantSeed struct {
	X float32 `json:"x"`
	Y float32 `json:"y"`
	Z float32 `json:"z"`
}

type WaterPlant struct {
	ResourceID string `json:"resource_id"`
}

// --- Мамонты ---

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

type TameMammoth struct {
	MammothID string `json:"mammoth_id"`
}

type LeashMammoth struct {
	MammothID string `json:"mammoth_id"`
}

type SaddleMammoth struct {
	MammothID string `json:"mammoth_id"`
}

type RideMammoth struct {
	MammothID string `json:"mammoth_id"`
}

// --- Мобы и контракты ---

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

type HitMob struct {
	MobID string `json:"mob_id"`
}

type AcceptContract struct {
	MobID      string `json:"mob_id"`
	ContractID string `json:"contract_id"` // "gather4" | "guard"
}

// --- Дома ---

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

type ToggleDoor struct {
	HouseID string `json:"house_id"`
}

// --- Лодки ---

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
