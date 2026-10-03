package protocol

// Vector3 — общий тип координат в протоколе.
type Vector3 struct {
	X float32 `json:"x"`
	Y float32 `json:"y"`
	Z float32 `json:"z"`
}

// ThrowSpear — бросок копья: направление в мировых координатах.
type ThrowSpear struct {
	Dir Vector3 `json:"dir"`
}

// CraftItem — крафт базового рецепта.
type CraftItem struct {
	Recipe string `json:"recipe"`
}

// HitMammoth — удар по мамонту.
type HitMammoth struct {
	MammothID string `json:"mammoth_id"`
}

// MobProjectile — летящий снаряд моба в снапшоте.
type MobProjectile struct {
	ID string  `json:"id"`
	X  float32 `json:"x"`
	Y  float32 `json:"y"`
	Z  float32 `json:"z"`
	DX float32 `json:"dx"`
	DY float32 `json:"dy"`
	DZ float32 `json:"dz"`
}

// Snapshot — полное состояние мира, рассылается клиентам ~20 раз в секунду.
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

	EarthPos Vector3 `json:"earth_pos,omitempty"`
	EarthVel Vector3 `json:"earth_vel,omitempty"`

	// Вторая звёздная система
	Planet2Pos Vector3 `json:"planet2_pos,omitempty"`
	Planet2Vel Vector3 `json:"planet2_vel,omitempty"`
}
