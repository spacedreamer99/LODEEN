package protocol

import "math"

var SunDirection = Vector3{X: 0.70, Y: 0.00, Z: 0.70}

const (
	SunDistance  float32 = 5000.0
	SunRadius    float32 = 250.0
	PlanetRadius float32 = 50.0
	PlayerHeight float32 = 1.5

	MuEarth float32 = 100000.0
	MuSun   float32 = 2e6

	SOIEarth float32 = 400.0
)

var SunPos = func() Vector3 {
	l := float32(math.Sqrt(float64(
		SunDirection.X*SunDirection.X +
			SunDirection.Y*SunDirection.Y +
			SunDirection.Z*SunDirection.Z)))
	return Vector3{
		X: SunDirection.X / l * SunDistance,
		Y: SunDirection.Y / l * SunDistance,
		Z: SunDirection.Z / l * SunDistance,
	}
}()
