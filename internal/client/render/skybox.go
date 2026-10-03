package render

import (
	"math"
	"math/rand"

	rl "github.com/gen2brain/raylib-go/raylib"
)

type skyStar struct {
	dir   rl.Vector3
	color rl.Color
	size  float32
}

var skyStars []skyStar
var skyInit bool

type spectralClass struct {
	color   rl.Color
	weight  float32
	sizeMin float32
	sizeMax float32
}

var spectralClasses = []spectralClass{
	{rl.NewColor(155, 176, 255, 255), 0.00003, 5.0, 6.0}, // O
	{rl.NewColor(170, 191, 255, 255), 0.0013, 4.5, 5.5},  // B
	{rl.NewColor(202, 215, 255, 255), 0.006, 4.0, 4.8},   // A
	{rl.NewColor(248, 247, 255, 255), 0.03, 3.5, 4.2},    // F
	{rl.NewColor(255, 244, 234, 255), 0.076, 3.2, 3.8},   // G
	{rl.NewColor(255, 210, 161, 255), 0.121, 2.8, 3.4},   // K
	{rl.NewColor(255, 180, 120, 255), 0.765, 2.4, 3.0},   // M
}

func initSkybox() {
	rng := rand.New(rand.NewSource(42))
	const total = 1200

	for i := 0; i < total; i++ {
		r := rng.Float32()
		var cum float32
		var cls spectralClass
		found := false
		for _, c := range spectralClasses {
			cum += c.weight
			if r <= cum {
				cls = c
				found = true
				break
			}
		}
		if !found {
			cls = spectralClasses[len(spectralClasses)-1]
		}

		u := rng.Float32()
		v := rng.Float32()
		z := 2*u - 1
		theta := 2 * math.Pi * v
		rxy := float32(math.Sqrt(float64(1 - z*z)))
		x := rxy * float32(math.Cos(float64(theta)))
		y := rxy * float32(math.Sin(float64(theta)))

		size := cls.sizeMin + rng.Float32()*(cls.sizeMax-cls.sizeMin)

		skyStars = append(skyStars, skyStar{
			dir:   rl.NewVector3(x, z, y),
			color: cls.color,
			size:  size,
		})
	}
}

func DrawSkybox(cam rl.Camera3D, screenW, screenH int32) {
	if !skyInit {
		initSkybox()
		skyInit = true
	}

	fwd := rl.Vector3Subtract(cam.Target, cam.Position)
	fwd = rl.Vector3Normalize(fwd)
	worldUp := rl.Vector3Normalize(cam.Up)
	right := rl.Vector3CrossProduct(fwd, worldUp)
	if rl.Vector3Length(right) < 0.001 {
		right = rl.NewVector3(1, 0, 0)
	} else {
		right = rl.Vector3Normalize(right)
	}
	up := rl.Vector3CrossProduct(right, fwd)

	sw := float32(screenW)
	sh := float32(screenH)
	fovRad := float64(cam.Fovy) * math.Pi / 180.0
	fovFactor := sh / (2 * float32(math.Tan(fovRad/2)))
	cx := sw / 2
	cy := sh / 2

	for _, s := range skyStars {
		z := s.dir.X*fwd.X + s.dir.Y*fwd.Y + s.dir.Z*fwd.Z
		if z <= 0.01 {
			continue
		}
		x := s.dir.X*right.X + s.dir.Y*right.Y + s.dir.Z*right.Z
		y := s.dir.X*up.X + s.dir.Y*up.Y + s.dir.Z*up.Z

		sx := cx + (x/z)*fovFactor
		sy := cy - (y/z)*fovFactor

		if sx < -8 || sx > sw+8 || sy < -8 || sy > sh+8 {
			continue
		}

		ix := int32(sx)
		iy := int32(sy)

		glow := s.color
		glow.A = 50
		rl.DrawCircle(ix, iy, s.size, glow)
		rl.DrawCircle(ix, iy, s.size*0.45, s.color)
	}
}
