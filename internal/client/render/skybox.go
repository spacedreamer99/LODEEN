package render

import (
	"math"

	rl "github.com/gen2brain/raylib-go/raylib"
)

// Диагностика: 10 ярких звёзд в известных направлениях.
type skyStar struct {
	dir   rl.Vector3
	color rl.Color
	size  float32
}

var skyStars []skyStar
var skyInit bool

func initSkybox() {
	const total = 10
	for i := 0; i < total; i++ {
		angle := float64(i) * 2 * 3.14159265 / float64(total)
		skyStars = append(skyStars, skyStar{
			dir:   rl.NewVector3(float32(math.Cos(angle)), 0, float32(math.Sin(angle))),
			color: rl.NewColor(255, 255, 255, 255),
			size:  8.0,
		})
	}
}

func DrawSkybox(cam rl.Camera3D, screenW, screenH int32) {
	if !skyInit {
		initSkybox()
		skyInit = true
	}

	// Camera basis.
	fwd := rl.Vector3Subtract(cam.Target, cam.Position)
	fwd = rl.Vector3Normalize(fwd)
	// ВАЖНО: используем cam.Up, а не мировой (0,1,0).
	// Иначе при крене (Q/E) звёзды не крутятся вместе с горизонтом.
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
	fovFactor := sh / (2 * 0.7) // tan(35°)≈0.7
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

		rl.DrawCircle(int32(sx), int32(sy), s.size, s.color)
	}
}
