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

// ═══════════════════════════════════════════════════════════════
// Вторая звёздная система: Star2 (красный карлик) + Planet2 (землеподобная)
// ═══════════════════════════════════════════════════════════════

var Star2Direction = Vector3{X: -0.707, Y: 0, Z: -0.707}

const (
	Star2Distance float32 = 30000.0 // расстояние от центра мира
	Star2Radius   float32 = 180.0   // визуальный радиус
	MuStar2       float32 = 1.5e6   // гравитационный параметр

	Planet2OrbitRadius float32 = 3000.0 // радиус орбиты вокруг Star2
	Planet2Radius      float32 = 60.0   // радиус планеты
	MuPlanet2          float32 = 150000.0
	SOIPlanet2         float32 = 500.0
)

var Star2Pos = func() Vector3 {
	l := float32(math.Sqrt(float64(
		Star2Direction.X*Star2Direction.X +
			Star2Direction.Y*Star2Direction.Y +
			Star2Direction.Z*Star2Direction.Z)))
	return Vector3{
		X: Star2Direction.X / l * Star2Distance,
		Y: Star2Direction.Y / l * Star2Distance,
		Z: Star2Direction.Z / l * Star2Distance,
	}
}()
