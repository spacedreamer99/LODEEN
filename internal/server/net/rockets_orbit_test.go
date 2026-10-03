package net

import (
	"math"
	"testing"

	"github.com/spacedreamer99/lodeen/internal/shared/protocol"
)

const eps = 1e-3 // относительная погрешность для float32

func almostEq(a, b float32) bool {
	d := a - b
	if d < 0 {
		d = -d
	}
	m := a
	if m < 0 {
		m = -m
	}
	if b < 0 {
		if -b > m {
			m = -b
		}
	}
	if m < 1 {
		return d < eps
	}
	return d/m < eps
}

// TestOrbitalParams_CircularOrbit — круговая орбита: r=8000, v=sqrt(mu/r).
// Ожидаем apo=peri=alt, speed=v_circ, targetV=v_circ.
func TestOrbitalParams_CircularOrbit(t *testing.T) {
	const mu = 1e6
	const bodyR = 1000
	const r = 8000
	pos := protocol.Vector3{X: r, Y: 0, Z: 0}
	vCirc := float32(math.Sqrt(float64(mu / r)))
	vel := protocol.Vector3{X: 0, Y: vCirc, Z: 0}

	apo, peri, speed, alt, targetV := orbitalParams(pos, vel, protocol.Vector3{}, mu, bodyR)

	wantAlt := float32(r - bodyR)
	if !almostEq(alt, wantAlt) {
		t.Fatalf("alt = %v, want %v", alt, wantAlt)
	}
	if !almostEq(speed, vCirc) {
		t.Fatalf("speed = %v, want %v", speed, vCirc)
	}
	if !almostEq(targetV, vCirc) {
		t.Fatalf("targetV = %v, want %v", targetV, vCirc)
	}
	if !almostEq(apo, wantAlt) {
		t.Fatalf("apo = %v, want %v", apo, wantAlt)
	}
	if !almostEq(peri, wantAlt) {
		t.Fatalf("peri = %v, want %v", peri, wantAlt)
	}
}

// TestOrbitalParams_EllipticOrbit — эллипс a=10000, перицентр r=5000.
// vis-viva: v^2 = mu*(2/r - 1/a).
// Ожидаем apo=15000-bodyR, peri=5000-bodyR.
func TestOrbitalParams_EllipticOrbit(t *testing.T) {
	const mu = 1e6
	const bodyR = 1000
	const r = 5000
	const a = 10000
	pos := protocol.Vector3{X: r, Y: 0, Z: 0}
	vPeri := float32(math.Sqrt(float64(mu * (2.0/r - 1.0/a))))
	vel := protocol.Vector3{X: 0, Y: vPeri, Z: 0}

	apo, peri, _, _, _ := orbitalParams(pos, vel, protocol.Vector3{}, mu, bodyR)

	wantApo := float32(a*1.5) - bodyR  // 14000
	wantPeri := float32(a*0.5) - bodyR // 4000
	if !almostEq(apo, wantApo) {
		t.Fatalf("apo = %v, want %v", apo, wantApo)
	}
	if !almostEq(peri, wantPeri) {
		t.Fatalf("peri = %v, want %v", peri, wantPeri)
	}
}

// TestOrbitalParams_EscapeVelocity — параболическая скорость: E=0.
// Функция должна вернуть (0, 0) как «орбиты нет».
func TestOrbitalParams_EscapeVelocity(t *testing.T) {
	const mu = 1e6
	const bodyR = 1000
	const r = 5000
	pos := protocol.Vector3{X: r, Y: 0, Z: 0}
	vEsc := float32(math.Sqrt(float64(2 * mu / r)))
	vel := protocol.Vector3{X: 0, Y: vEsc, Z: 0}

	apo, peri, speed, _, _ := orbitalParams(pos, vel, protocol.Vector3{}, mu, bodyR)

	if apo != 0 || peri != 0 {
		t.Fatalf("expected (0,0) for parabolic orbit, got (%v,%v)", apo, peri)
	}
	if !almostEq(speed, vEsc) {
		t.Fatalf("speed = %v, want %v", speed, vEsc)
	}
}

// TestOrbitalParams_Hyperbolic — скорость выше escape. Тоже (0,0).
func TestOrbitalParams_Hyperbolic(t *testing.T) {
	const mu = 1e6
	const bodyR = 1000
	const r = 5000
	pos := protocol.Vector3{X: r, Y: 0, Z: 0}
	vHyper := float32(math.Sqrt(float64(2*mu/r))) * 1.5
	vel := protocol.Vector3{X: 0, Y: vHyper, Z: 0}

	apo, peri, _, _, _ := orbitalParams(pos, vel, protocol.Vector3{}, mu, bodyR)
	if apo != 0 || peri != 0 {
		t.Fatalf("expected (0,0) for hyperbolic orbit, got (%v,%v)", apo, peri)
	}
}

// TestOrbitalParams_TooCloseToCenter — r < 1: возвращаем (0,0) без паники.
func TestOrbitalParams_TooCloseToCenter(t *testing.T) {
	pos := protocol.Vector3{X: 0.5, Y: 0, Z: 0}
	vel := protocol.Vector3{X: 0, Y: 100, Z: 0}

	apo, peri, speed, _, _ := orbitalParams(pos, vel, protocol.Vector3{}, 1e6, 1000)
	if apo != 0 || peri != 0 {
		t.Fatalf("expected (0,0), got (%v,%v)", apo, peri)
	}
	if speed == 0 {
		t.Fatalf("speed should still be computed")
	}
}

// TestOrbitalParams_TranslationInvariance — результат не зависит
// от сдвига primaryPos (используем относительные координаты).
func TestOrbitalParams_TranslationInvariance(t *testing.T) {
	const mu = 1e6
	const bodyR = 1000
	primary1 := protocol.Vector3{}
	primary2 := protocol.Vector3{X: 1e6, Y: -5e5, Z: 3e5}
	relPos := protocol.Vector3{X: 5000, Y: 0, Z: 0}
	relVel := protocol.Vector3{X: 0, Y: 400, Z: 0}

	a1, p1, s1, alt1, tv1 := orbitalParams(relPos, relVel, primary1, mu, bodyR)
	a2, p2, s2, alt2, tv2 := orbitalParams(
		protocol.Vector3{X: primary2.X + relPos.X, Y: primary2.Y + relPos.Y, Z: primary2.Z + relPos.Z},
		relVel, primary2, mu, bodyR)

	if a1 != a2 || p1 != p2 || s1 != s2 || alt1 != alt2 || tv1 != tv2 {
		t.Fatalf("result not translation-invariant:\n p1=(%v,%v,%v,%v,%v)\n p2=(%v,%v,%v,%v,%v)",
			a1, p1, s1, alt1, tv1, a2, p2, s2, alt2, tv2)
	}
}
