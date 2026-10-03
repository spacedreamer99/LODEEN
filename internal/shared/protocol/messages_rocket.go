package protocol

// Rocket — сериализованное состояние ракеты в снапшоте.
type Rocket struct {
	ID             string  `json:"id"`
	X              float32 `json:"x"`
	Y              float32 `json:"y"`
	Z              float32 `json:"z"`
	DX             float32 `json:"dx"`
	DY             float32 `json:"dy"`
	DZ             float32 `json:"dz"`
	Fuel           int     `json:"fuel"`
	MaxFuel        int     `json:"max_fuel"`
	Piloted        bool    `json:"piloted"`
	OwnerID        string  `json:"owner_id,omitempty"`
	InOrbit        bool    `json:"in_orbit,omitempty"`
	Apoapsis       float32 `json:"apoapsis,omitempty"`  // высота апогея, м
	Periapsis      float32 `json:"periapsis,omitempty"` // высота перигея, м
	Speed          float32 `json:"speed,omitempty"`     // |v| в м/с
	Altitude       float32 `json:"altitude,omitempty"`  // высота над поверхностью
	TargetVelocity float32 `json:"target_velocity,omitempty"`
	VX             float32 `json:"vx,omitempty"`
	VY             float32 `json:"vy,omitempty"`
	VZ             float32 `json:"vz,omitempty"`
	PrimaryBody    string  `json:"primary_body,omitempty"`
	DistSun        float32 `json:"dist_sun,omitempty"`
	SOIRadius      float32 `json:"soi_radius,omitempty"`
}

type PlaceRocket struct {
	X float32 `json:"x"`
	Y float32 `json:"y"`
	Z float32 `json:"z"`
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
