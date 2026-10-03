package render

import (
	rl "github.com/gen2brain/raylib-go/raylib"
	"math"
	"unsafe"

	"github.com/spacedreamer99/lodeen/internal/shared/protocol"
)

// genSunTexture — процедурная текстура солнца (radial gradient с alpha).
func genSunTexture() rl.Texture2D {
	const size = 256
	img := rl.GenImageColor(size, size, rl.Blank)
	cx := float32(size) / 2
	cy := float32(size) / 2

	for y := 0; y < size; y++ {
		for x := 0; x < size; x++ {
			dx := float32(x) - cx
			dy := float32(y) - cy
			d := float32(math.Sqrt(float64(dx*dx+dy*dy))) / cx
			if d > 1.0 {
				continue
			}
			// Плавное затухание: (1-d)^2
			f := 1.0 - float64(d)
			a := uint8(255 * f * f)
			// Цвет: бело-жёлтый центр, оранжевый край.
			r := uint8(255)
			g := uint8(240 - 60*d)
			b := uint8(200 - 150*d)
			rl.ImageDrawPixel(img, int32(x), int32(y), rl.NewColor(r, g, b, a))
		}
	}

	tex := rl.LoadTextureFromImage(img)
	rl.UnloadImage(img)
	return tex
}

// DrawSun3D — Солнце как 3D-сфера. Для далёких расстояний подтягивает
// позицию к камере (обходит far plane raylib), сохраняя угловой размер.
func (s *Scene) DrawSun3D(cam rl.Camera3D) {
	sunPos := rl.NewVector3(protocol.SunPos.X, protocol.SunPos.Y, protocol.SunPos.Z)

	// Проверка «перед камерой».
	fwdX := cam.Target.X - cam.Position.X
	fwdY := cam.Target.Y - cam.Position.Y
	fwdZ := cam.Target.Z - cam.Position.Z
	toX := sunPos.X - cam.Position.X
	toY := sunPos.Y - cam.Position.Y
	toZ := sunPos.Z - cam.Position.Z
	if fwdX*toX+fwdY*toY+fwdZ*toZ <= 0 {
		return
	}

	dirX := sunPos.X - cam.Position.X
	dirY := sunPos.Y - cam.Position.Y
	dirZ := sunPos.Z - cam.Position.Z
	dist := float32(math.Sqrt(float64(dirX*dirX + dirY*dirY + dirZ*dirZ)))
	if dist < 0.01 {
		return
	}

	const maxDist = 850.0
	var drawPos rl.Vector3
	var drawRadius float32

	if dist <= maxDist {
		drawPos = sunPos
		drawRadius = protocol.SunRadius
	} else {
		k := maxDist / dist
		drawPos = rl.NewVector3(
			cam.Position.X+dirX*k,
			cam.Position.Y+dirY*k,
			cam.Position.Z+dirZ*k,
		)
		drawRadius = protocol.SunRadius * k
	}

	// Ядро.
	rl.DrawSphere(drawPos, drawRadius, rl.NewColor(255, 220, 80, 255))
	// Лёгкие wireframe для объёма.
	rl.DrawSphereWires(drawPos, drawRadius, 24, 24, rl.NewColor(255, 160, 40, 160))
	// Внешнее свечение (тонкая оболочка).
	if drawRadius < maxDist*0.9 {
		rl.DrawSphereWires(drawPos, drawRadius*1.12, 16, 16, rl.NewColor(255, 200, 100, 70))
	}
}

// deformMesh — только смещает вершины по TerrainHeight.
func deformMesh(mesh *rl.Mesh) {
	if mesh == nil || mesh.Vertices == nil || mesh.VertexCount == 0 {
		return
	}
	count := int(mesh.VertexCount)
	verts := unsafe.Slice((*float32)(mesh.Vertices), count*3)

	for i := 0; i < count; i++ {
		x := verts[i*3+0]
		y := verts[i*3+1]
		z := verts[i*3+2]
		l := float32(math.Sqrt(float64(x*x + y*y + z*z)))
		if l < 0.01 {
			continue
		}
		nx := x / l
		ny := y / l
		nz := z / l
		h := protocol.TerrainHeight(nx, ny, nz)
		verts[i*3+0] = nx * h
		verts[i*3+1] = ny * h
		verts[i*3+2] = nz * h
	}

	if mesh.VaoID == 0 {
		return
	}
	vb := unsafe.Slice((*byte)(unsafe.Pointer(mesh.Vertices)), count*3*4)
	rl.UpdateMeshBuffer(*mesh, 0, vb, 0)
}

// DrawAtmosphere — рисует атмосферную сферу с Fresnel-эффектом.
func (s *Scene) DrawAtmosphere(cam rl.Camera3D) {
	if s.atmosphere.MeshCount == 0 {
		return
	}
	if s.atmoShader.ID != 0 {
		loc := rl.GetShaderLocation(s.atmoShader, "cameraPos")
		ep := rl.NewVector3(s.earthPos.X, s.earthPos.Y, s.earthPos.Z)
		cp := rl.Vector3Subtract(cam.Position, ep)
		rl.SetShaderValue(s.atmoShader, loc,
			[]float32{cp.X, cp.Y, cp.Z}, rl.ShaderUniformVec3)
	}
	rl.BeginBlendMode(rl.BlendAlpha)
	rl.DisableDepthMask()
	rl.DisableBackfaceCulling()
	rl.DrawModel(s.atmosphere, rl.Vector3Zero(), 1.0, rl.White)
	rl.EnableBackfaceCulling()
	rl.EnableDepthMask()
	rl.EndBlendMode()
}

// DrawClouds — слой облаков с движением.
func (s *Scene) DrawClouds(cam rl.Camera3D) {
	if s.clouds.MeshCount == 0 {
		return
	}
	if s.cloudsShader.ID != 0 {
		loc := rl.GetShaderLocation(s.cloudsShader, "time")
		rl.SetShaderValue(s.cloudsShader, loc,
			[]float32{float32(rl.GetTime())}, rl.ShaderUniformFloat)
	}
	rl.BeginBlendMode(rl.BlendAlpha)
	rl.DisableDepthMask()
	rl.DisableBackfaceCulling()
	rl.DrawModel(s.clouds, rl.Vector3Zero(), 1.0, rl.White)
	rl.EnableBackfaceCulling()
	rl.EnableDepthMask()
	rl.EndBlendMode()
}

// DrawStar2 — вторая звезда как 3D-сфера. Тот же трюк с far plane,
// что и у DrawSun3D: подтягиваем позицию к камере, сохраняя угловой размер.
func (s *Scene) DrawStar2(cam rl.Camera3D) {
	star2Pos := rl.NewVector3(protocol.Star2Pos.X, protocol.Star2Pos.Y, protocol.Star2Pos.Z)

	fwdX := cam.Target.X - cam.Position.X
	fwdY := cam.Target.Y - cam.Position.Y
	fwdZ := cam.Target.Z - cam.Position.Z
	toX := star2Pos.X - cam.Position.X
	toY := star2Pos.Y - cam.Position.Y
	toZ := star2Pos.Z - cam.Position.Z
	if fwdX*toX+fwdY*toY+fwdZ*toZ <= 0 {
		return
	}

	dirX := star2Pos.X - cam.Position.X
	dirY := star2Pos.Y - cam.Position.Y
	dirZ := star2Pos.Z - cam.Position.Z
	dist := float32(math.Sqrt(float64(dirX*dirX + dirY*dirY + dirZ*dirZ)))
	if dist < 0.01 {
		return
	}

	const maxDist = 850.0
	var drawPos rl.Vector3
	var drawRadius float32

	if dist <= maxDist {
		drawPos = star2Pos
		drawRadius = protocol.Star2Radius
	} else {
		k := maxDist / dist
		drawPos = rl.NewVector3(
			cam.Position.X+dirX*k,
			cam.Position.Y+dirY*k,
			cam.Position.Z+dirZ*k,
		)
		drawRadius = protocol.Star2Radius * k
	}

	// Оранжевый карлик.
	rl.DrawSphere(drawPos, drawRadius, rl.NewColor(255, 140, 60, 255))
	rl.DrawSphereWires(drawPos, drawRadius, 24, 24, rl.NewColor(255, 90, 20, 200))
	if drawRadius < maxDist*0.9 {
		rl.DrawSphereWires(drawPos, drawRadius*1.12, 16, 16,
			rl.NewColor(255, 180, 100, 80))
	}
}

// DrawPlanet2 — вторая планета как 3D-сфера.
// planet2Pos — helio-координаты (из snapshot сервера).
func (s *Scene) DrawPlanet2(cam rl.Camera3D, planet2Pos protocol.Vector3) {
	p2 := rl.NewVector3(planet2Pos.X, planet2Pos.Y, planet2Pos.Z)

	fwdX := cam.Target.X - cam.Position.X
	fwdY := cam.Target.Y - cam.Position.Y
	fwdZ := cam.Target.Z - cam.Position.Z
	toX := p2.X - cam.Position.X
	toY := p2.Y - cam.Position.Y
	toZ := p2.Z - cam.Position.Z
	if fwdX*toX+fwdY*toY+fwdZ*toZ <= 0 {
		return
	}

	dirX := p2.X - cam.Position.X
	dirY := p2.Y - cam.Position.Y
	dirZ := p2.Z - cam.Position.Z
	dist := float32(math.Sqrt(float64(dirX*dirX + dirY*dirY + dirZ*dirZ)))
	if dist < 0.01 {
		return
	}

	const maxDist = 850.0
	var drawPos rl.Vector3
	var drawRadius float32

	if dist <= maxDist {
		drawPos = p2
		drawRadius = protocol.Planet2Radius
	} else {
		k := maxDist / dist
		drawPos = rl.NewVector3(
			cam.Position.X+dirX*k,
			cam.Position.Y+dirY*k,
			cam.Position.Z+dirZ*k,
		)
		drawRadius = protocol.Planet2Radius * k
	}

	// Голубоватая планета с ободком.
	rl.DrawSphere(drawPos, drawRadius, rl.NewColor(80, 180, 200, 255))
	rl.DrawSphereWires(drawPos, drawRadius, 24, 24, rl.NewColor(40, 120, 160, 200))
	// Атмосфера (тонкое кольцо).
	if drawRadius < maxDist*0.9 {
		rl.DrawSphereWires(drawPos, drawRadius*1.08, 16, 16,
			rl.NewColor(160, 220, 240, 60))
	}
}
