package protocol

import (
	"math"
	"testing"
)

// --- TerrainHeight ---

// TestTerrainHeight_Deterministic — одна и та же нормаль даёт одно и то же значение.
func TestTerrainHeight_Deterministic(t *testing.T) {
	type point struct{ x, y, z float32 }
	pts := []point{
		{1, 0, 0}, {0, 1, 0}, {0, 0, 1},
		{-1, 0, 0}, {0.577, 0.577, 0.577},
		{0.5, -0.5, 0.707},
	}
	for _, p := range pts {
		h1 := TerrainHeight(p.x, p.y, p.z)
		h2 := TerrainHeight(p.x, p.y, p.z)
		if h1 != h2 {
			t.Fatalf("TerrainHeight(%v,%v,%v) not deterministic: %v vs %v", p.x, p.y, p.z, h1, h2)
		}
	}
}

// TestTerrainHeight_Bounded — высота всегда в [PlanetRadius-MaxRelief, PlanetRadius+MaxRelief].
// Это гарантия, что физика ракеты не проваливается сквозь планету.
func TestTerrainHeight_Bounded(t *testing.T) {
	minH := PlanetRadius - MaxRelief
	maxH := PlanetRadius + MaxRelief
	for x := float32(-1); x <= 1; x += 0.1 {
		for y := float32(-1); y <= 1; y += 0.1 {
			for z := float32(-1); z <= 1; z += 0.1 {
				l2 := x*x + y*y + z*z
				if l2 < 0.01 {
					continue
				}
				l := float32(math.Sqrt(float64(l2)))
				h := TerrainHeight(x/l, y/l, z/l)
				if h < minH || h > maxH {
					t.Fatalf("TerrainHeight(%v,%v,%v) = %v out of [%v, %v]",
						x/l, y/l, z/l, h, minH, maxH)
				}
			}
		}
	}
}

// --- SurfaceRadius ---

func TestSurfaceRadius_ZeroVector(t *testing.T) {
	r := SurfaceRadius(Vector3{})
	if r != PlanetRadius {
		t.Fatalf("SurfaceRadius(zero) = %v, want %v", r, PlanetRadius)
	}
}

func TestSurfaceRadius_TinyVector(t *testing.T) {
	// l < 0.01 → PlanetRadius
	r := SurfaceRadius(Vector3{X: 0.001, Y: 0, Z: 0})
	if r != PlanetRadius {
		t.Fatalf("SurfaceRadius(tiny) = %v, want %v", r, PlanetRadius)
	}
}

// TestSurfaceRadius_NeverBelowSeaLevel — ключевое свойство: поверхность никогда ниже уровня моря.
func TestSurfaceRadius_NeverBelowSeaLevel(t *testing.T) {
	for x := float32(-1); x <= 1; x += 0.15 {
		for y := float32(-1); y <= 1; y += 0.15 {
			for z := float32(-1); z <= 1; z += 0.15 {
				l2 := x*x + y*y + z*z
				if l2 < 0.01 {
					continue
				}
				l := float32(math.Sqrt(float64(l2)))
				r := SurfaceRadius(Vector3{X: x / l * 100, Y: y / l * 100, Z: z / l * 100})
				if r < SeaLevel {
					t.Fatalf("SurfaceRadius(%v,%v,%v) = %v < SeaLevel %v",
						x/l, y/l, z/l, r, SeaLevel)
				}
			}
		}
	}
}

// TestSurfaceRadius_ScaleInvariant — только направление вектора важно, не длина.
func TestSurfaceRadius_ScaleInvariant(t *testing.T) {
	dir := Vector3{X: 0.577, Y: 0.577, Z: 0.577}
	r1 := SurfaceRadius(Vector3{X: dir.X * 10, Y: dir.Y * 10, Z: dir.Z * 10})
	r2 := SurfaceRadius(Vector3{X: dir.X * 1e6, Y: dir.Y * 1e6, Z: dir.Z * 1e6})
	if r1 != r2 {
		t.Fatalf("SurfaceRadius not scale-invariant: %v vs %v", r1, r2)
	}
}

// --- ClampToSurface ---

func TestClampToSurface_ZeroReturnsZero(t *testing.T) {
	got := ClampToSurface(Vector3{})
	if got != (Vector3{}) {
		t.Fatalf("ClampToSurface(zero) = %+v, want zero", got)
	}
}

func TestClampToSurface_PutsOnSurface(t *testing.T) {
	cases := []Vector3{
		{X: 10, Y: 0, Z: 0},
		{X: 0, Y: 100, Z: 0},
		{X: 50, Y: 50, Z: 50},
		{X: -30, Y: 40, Z: -50},
		{X: 1, Y: -1, Z: 0.5},
	}
	for _, pos := range cases {
		got := ClampToSurface(pos)
		lGot := float32(math.Sqrt(float64(got.X*got.X + got.Y*got.Y + got.Z*got.Z)))
		want := SurfaceRadius(pos)
		if math.Abs(float64(lGot-want)) > 0.01 {
			t.Fatalf("ClampToSurface(%+v): |result| = %v, want %v", pos, lGot, want)
		}
	}
}

// TestClampToSurface_PreservesDirection — результат коллинеарен исходному вектору (с точностью float32).
func TestClampToSurface_PreservesDirection(t *testing.T) {
	cases := []Vector3{
		{X: 10, Y: 20, Z: 30},
		{X: -5, Y: 15, Z: -25},
		{X: 0.1, Y: 0.2, Z: 0.3},
	}
	for _, pos := range cases {
		got := ClampToSurface(pos)
		// Нормализуем оба и сравним
		l1 := float32(math.Sqrt(float64(pos.X*pos.X + pos.Y*pos.Y + pos.Z*pos.Z)))
		l2 := float32(math.Sqrt(float64(got.X*got.X + got.Y*got.Y + got.Z*got.Z)))
		n1x, n1y, n1z := pos.X/l1, pos.Y/l1, pos.Z/l1
		n2x, n2y, n2z := got.X/l2, got.Y/l2, got.Z/l2
		diff := math.Abs(float64(n1x-n2x)) + math.Abs(float64(n1y-n2y)) + math.Abs(float64(n1z-n2z))
		if diff > 1e-5 {
			t.Fatalf("ClampToSurface(%+v): direction changed, diff = %v", pos, diff)
		}
	}
}

// TestClampToSurface_Idempotent — повторный вызов не смещает точку
// больше, чем на float32-погрешность. Строгое бит-равенство тут недостижимо:
// float32-нормализация даёт дрейф ~1e-6 на 6-м знаке после второго прохода.
func TestClampToSurface_Idempotent(t *testing.T) {
	cases := []Vector3{
		{X: 10, Y: 20, Z: 30},
		{X: 100, Y: 0, Z: 0},
		{X: 1, Y: 1, Z: 1},
	}
	const tol = float32(1e-5)
	for _, pos := range cases {
		once := ClampToSurface(pos)
		twice := ClampToSurface(once)
		if absF32(once.X-twice.X) > tol ||
			absF32(once.Y-twice.Y) > tol ||
			absF32(once.Z-twice.Z) > tol {
			t.Fatalf("ClampToSurface drift > %v for %+v:\n once=%+v\n twice=%+v", tol, pos, once, twice)
		}
	}
}

func absF32(v float32) float32 {
	if v < 0 {
		return -v
	}
	return v
}

// TestClampToSurface_LiftsBelowSurface — точка внутри планеты поднимается на поверхность.
func TestClampToSurface_LiftsBelowSurface(t *testing.T) {
	below := Vector3{X: 5, Y: 0, Z: 0} // сильно меньше PlanetRadius
	got := ClampToSurface(below)
	lGot := float32(math.Sqrt(float64(got.X*got.X + got.Y*got.Y + got.Z*got.Z)))
	if lGot < SeaLevel {
		t.Fatalf("ClampToSurface did not lift below-surface point: |result| = %v, SeaLevel = %v", lGot, SeaLevel)
	}
}
