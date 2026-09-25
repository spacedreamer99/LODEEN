package protocol

import "math"

// SeaLevel — радиус уровня моря.
const SeaLevel float32 = PlanetRadius + 2.0

// MaxRelief — максимальное отклонение рельефа от PlanetRadius.
const MaxRelief float32 = 12.0

// TerrainHeight возвращает радиус поверхности в направлении (nx, ny, nz).
func TerrainHeight(nx, ny, nz float32) float32 {
	h := float32(math.Sin(float64(nx*3.1))*math.Cos(float64(ny*2.7))*math.Sin(float64(nz*3.3))) * MaxRelief * 0.6
	h += float32(math.Sin(float64(nx*7.3+ny*5.1))*math.Cos(float64(nz*6.7))) * MaxRelief * 0.3
	h += float32(math.Cos(float64(nx*11.5+ny*13.1+nz*9.7))) * MaxRelief * 0.1
	return PlanetRadius + h
}

// SurfaceRadius — высота поверхности в точке pos (или уровень моря, если ниже).
func SurfaceRadius(pos Vector3) float32 {
	l := float32(math.Sqrt(float64(pos.X*pos.X + pos.Y*pos.Y + pos.Z*pos.Z)))
	if l < 0.01 {
		return PlanetRadius
	}
	nx := pos.X / l
	ny := pos.Y / l
	nz := pos.Z / l
	h := TerrainHeight(nx, ny, nz)
	if h < SeaLevel {
		return SeaLevel
	}
	return h
}

// ClampToSurface — двигает pos на поверхность (или уровень моря) в его направлении.
func ClampToSurface(pos Vector3) Vector3 {
	l := float32(math.Sqrt(float64(pos.X*pos.X + pos.Y*pos.Y + pos.Z*pos.Z)))
	if l < 0.01 {
		return pos
	}
	nx := pos.X / l
	ny := pos.Y / l
	nz := pos.Z / l
	h := TerrainHeight(nx, ny, nz)
	if h < SeaLevel {
		h = SeaLevel
	}
	return Vector3{X: nx * h, Y: ny * h, Z: nz * h}
}
