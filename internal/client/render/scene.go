package render

import (
	rl "github.com/gen2brain/raylib-go/raylib"
	"math"
	"unsafe"

	"github.com/spacedreamer99/lodeen/internal/shared/protocol"
)

const playerCubeSize = 5.0

type Scene struct {
	planet    rl.Model
	hasPlanet bool

	fallback       rl.Model
	fallbackMat    rl.Material
	fallbackColors []byte

	water rl.Model

	planetPos    rl.Vector3
	planetScale  float32
	sunTex       rl.Texture2D
	colorShader  rl.Shader
	waterShader  rl.Shader
	atmoShader   rl.Shader
	atmosphere   rl.Model
	cloudsShader rl.Shader
	clouds       rl.Model
	earthPos     rl.Vector3
}

func NewScene(planetPath string) *Scene {
	s := &Scene{
		planetPos:   rl.Vector3Zero(),
		planetScale: 1.0,
	}

	// 1. Загружаем плоскую сферу — VaoID уже создан.
	mesh := rl.GenMeshSphere(protocol.PlanetRadius, 96, 96)
	s.fallback = rl.LoadModelFromMesh(mesh)

	// 2. Деформируем уже загруженный меш + UpdateMeshBuffer → GPU.
	fbMeshes := unsafe.Slice(s.fallback.Meshes, int(s.fallback.MeshCount))
	for i := range fbMeshes {
		deformMesh(&fbMeshes[i])
	}

	// 3. Зелёный tint материала.
	// Custom shader: цвет по мировой позиции.
	s.colorShader = rl.LoadShaderFromMemory(shaderVS, shaderFS)
	if s.colorShader.ID != 0 {
		mats := unsafe.Slice(s.fallback.Materials, int(s.fallback.MaterialCount))
		for i := range mats {
			mats[i].Shader = s.colorShader
			mats[i].Maps.Color = rl.White
		}
	}
	s.fallbackMat = rl.LoadMaterialDefault()
	s.fallbackMat.Maps.Color = rl.NewColor(80, 140, 90, 255)

	if planetPath != "" {
		m := rl.LoadModel(planetPath)
		if m.MeshCount > 0 {
			s.planet = m
			s.hasPlanet = true
		}
	}

	waterMesh := rl.GenMeshSphere(protocol.SeaLevel, 64, 64)
	s.water = rl.LoadModelFromMesh(waterMesh)

	// Облака: сфера радиуса PlanetRadius + 15 (ниже атмосферы).
	cloudsMesh := rl.GenMeshSphere(protocol.PlanetRadius+15.0, 64, 64)
	s.clouds = rl.LoadModelFromMesh(cloudsMesh)
	s.cloudsShader = rl.LoadShaderFromMemory(cloudsVS, cloudsFS)
	if s.cloudsShader.ID != 0 && s.clouds.MaterialCount > 0 {
		cmats := unsafe.Slice(s.clouds.Materials, int(s.clouds.MaterialCount))
		for i := range cmats {
			cmats[i].Shader = s.cloudsShader
			cmats[i].Maps.Color = rl.White
		}
	}

	// Атмосфера: сфера радиуса PlanetRadius+atmosphereHeight.
	// atmosphereHeight хардкод 50 (совпадает с физикой ракеты).
	const atmoHeight = 20.0
	atmoMesh := rl.GenMeshSphere(protocol.PlanetRadius+atmoHeight, 64, 64)
	s.atmosphere = rl.LoadModelFromMesh(atmoMesh)
	s.atmoShader = rl.LoadShaderFromMemory(atmoVS, atmoFS)
	rl.TraceLog(rl.LogInfo, "atmoShader ID = %d", s.atmoShader.ID)
	if s.atmoShader.ID != 0 && s.atmosphere.MaterialCount > 0 {
		amats := unsafe.Slice(s.atmosphere.Materials, int(s.atmosphere.MaterialCount))
		for i := range amats {
			amats[i].Shader = s.atmoShader
			amats[i].Maps.Color = rl.White
		}
	}

	// Водный шейдер: движущиеся пиксельные волны.
	s.waterShader = rl.LoadShaderFromMemory(waterVS, waterFS)
	rl.TraceLog(rl.LogInfo, "waterShader ID = %d", s.waterShader.ID)
	if s.waterShader.ID != 0 && s.water.MaterialCount > 0 {
		wmats := unsafe.Slice(s.water.Materials, int(s.water.MaterialCount))
		for i := range wmats {
			wmats[i].Shader = s.waterShader
			wmats[i].Maps.Color = rl.White
		}
	}

	s.sunTex = genSunTexture()

	return s
}

func (s *Scene) Unload() {
	if s.colorShader.ID != 0 {
		rl.UnloadShader(s.colorShader)
	}
	if s.waterShader.ID != 0 {
		rl.UnloadShader(s.waterShader)
	}
	if s.atmoShader.ID != 0 {
		rl.UnloadShader(s.atmoShader)
	}
	if s.atmosphere.MeshCount > 0 {
		rl.UnloadModel(s.atmosphere)
	}
	if s.cloudsShader.ID != 0 {
		rl.UnloadShader(s.cloudsShader)
	}
	if s.clouds.MeshCount > 0 {
		rl.UnloadModel(s.clouds)
	}
	if s.sunTex.ID != 0 {
		rl.UnloadTexture(s.sunTex)
	}
	if s.hasPlanet {
		rl.UnloadModel(s.planet)
	}
	rl.UnloadModel(s.fallback)
}

func (s *Scene) HasPlanet() bool { return s.hasPlanet }

func (s *Scene) SetEarthPos(p protocol.Vector3) {
	s.earthPos = rl.NewVector3(p.X, p.Y, p.Z)
}

func (s *Scene) SetPlanetScale(scale float32) { s.planetScale = scale }

func (s *Scene) Draw() {
	if s.hasPlanet {
		rl.DrawModel(s.planet, s.planetPos, s.planetScale, rl.White)
		return
	}
	// Шейдер красит сам.
	rl.DrawModel(s.fallback, s.planetPos, s.planetScale, rl.White)

	// Wireframe поверх — тонкие тёмные линии рёбер.
	rl.EnableWireMode()
	rl.DrawModel(s.fallback, s.planetPos, s.planetScale,
		rl.NewColor(20, 40, 25, 255))
	rl.DisableWireMode()
}

func DrawPlayers(players []protocol.PlayerState, ownID string, _ rl.Camera3D) {
	for _, p := range players {
		if p.ID == ownID {
			continue
		}
		pos := rl.NewVector3(p.X, p.Y, p.Z)
		col := colorForID(p.ID)

		rl.DrawCube(pos, playerCubeSize, playerCubeSize, playerCubeSize, col)
		rl.DrawCubeWires(pos, playerCubeSize, playerCubeSize, playerCubeSize, rl.Black)
	}
}

// DrawWater рисует полупрозрачную воду на уровне SeaLevel.
func (s *Scene) DrawWater() {
	if s.water.MeshCount == 0 {
		return
	}
	if s.waterShader.ID != 0 {
		loc := rl.GetShaderLocation(s.waterShader, "time")
		t := float32(rl.GetTime())
		rl.SetShaderValue(s.waterShader, loc, []float32{t}, rl.ShaderUniformFloat)
	}
	// Alpha blending для полупрозрачной воды.
	rl.BeginBlendMode(rl.BlendAlpha)
	rl.DrawModel(s.water, s.planetPos, s.planetScale, rl.White)
	rl.EndBlendMode()
}

func DrawResources(resources []protocol.Resource) {
	for _, r := range resources {
		pos := rl.NewVector3(r.X, r.Y, r.Z)
		var col rl.Color
		switch r.Type {
		case "stone":
			col = rl.NewColor(140, 140, 150, 255)
		case "wood":
			col = rl.NewColor(120, 80, 40, 255)
		case "ore":
			col = rl.NewColor(200, 170, 60, 255)
		case "fruit":
			col = rl.NewColor(230, 70, 70, 255)
		case "meat":
			col = rl.NewColor(200, 100, 100, 255)
		case "spear":
			col = rl.NewColor(180, 160, 120, 255)
		case "water":
			col = rl.NewColor(60, 120, 220, 255)
		case "seed":
			if r.Watered {
				col = rl.NewColor(150, 240, 100, 255) // светло-зелёный — полит
			} else {
				col = rl.NewColor(30, 100, 30, 255) // тёмно-зелёный — ждёт воду
			}
		default:
			col = rl.NewColor(255, 0, 255, 255) // magenta — "неизвестный тип"
		}
		size := float32(1.0)
		if r.Type == "seed" {
			size = 0.6
		}
		rl.DrawCube(pos, size, size, size, col)
		rl.DrawCubeWires(pos, size, size, size, rl.Black)
	}
}

func colorForID(id string) rl.Color {
	var h uint32 = 2166136261
	for i := 0; i < len(id); i++ {
		h ^= uint32(id[i])
		h *= 16777619
	}
	r := uint8(96 + (h>>0)&0x7F)
	g := uint8(96 + (h>>8)&0x7F)
	b := uint8(96 + (h>>16)&0x7F)
	return rl.NewColor(r, g, b, 255)
}

func SpawnFromID(id string) rl.Vector3 {
	// Точка на экваторе — стабильное "северное" направление.
	r := protocol.PlanetRadius + protocol.PlayerHeight
	return rl.NewVector3(r, 0, 0)
}

func SpawnYawFromID(id string) float32 {
	h := hashID(id)
	angle := float64(h%360) * 3.141592653589793 / 180.0
	return float32(angle)
}

func hashID(id string) uint32 {
	var h uint32 = 2166136261
	for i := 0; i < len(id); i++ {
		h ^= uint32(id[i])
		h *= 16777619
	}
	return h
}

// DrawLeash — линия от точки A до мамонта.
func DrawLeash(from, to rl.Vector3) {
	rl.DrawLine3D(from, to, rl.NewColor(150, 110, 70, 255))
}

// DrawHouses рисует дома: 6 плоских кубов + дверь.
func DrawHouses(houses []protocol.House) {
	wood := rl.NewColor(140, 95, 55, 255)
	woodDark := rl.NewColor(90, 60, 35, 255)
	doorOpen := rl.NewColor(190, 130, 75, 255)

	for _, h := range houses {
		rl.PushMatrix()
		rl.Translatef(h.X, h.Y, h.Z)
		rl.Rotatef(h.Yaw*180.0/3.14159265, 0, 1, 0)

		// Пол
		rl.DrawCube(rl.NewVector3(0, -1.5, 0), 6, 0.2, 6, wood)
		rl.DrawCubeWires(rl.NewVector3(0, -1.5, 0), 6, 0.2, 6, woodDark)
		// Потолок
		rl.DrawCube(rl.NewVector3(0, 1.5, 0), 6, 0.2, 6, wood)
		rl.DrawCubeWires(rl.NewVector3(0, 1.5, 0), 6, 0.2, 6, woodDark)
		// Задняя стена
		rl.DrawCube(rl.NewVector3(0, 0, -3), 6, 3, 0.2, wood)
		rl.DrawCubeWires(rl.NewVector3(0, 0, -3), 6, 3, 0.2, woodDark)
		// Левая стена
		rl.DrawCube(rl.NewVector3(-3, 0, 0), 0.2, 3, 6, wood)
		rl.DrawCubeWires(rl.NewVector3(-3, 0, 0), 0.2, 3, 6, woodDark)
		// Правая стена
		rl.DrawCube(rl.NewVector3(3, 0, 0), 0.2, 3, 6, wood)
		rl.DrawCubeWires(rl.NewVector3(3, 0, 0), 0.2, 3, 6, woodDark)
		// Передняя стена с проёмом
		rl.DrawCube(rl.NewVector3(-1.8, 0, 3), 2.4, 3, 0.2, wood)
		rl.DrawCubeWires(rl.NewVector3(-1.8, 0, 3), 2.4, 3, 0.2, woodDark)
		rl.DrawCube(rl.NewVector3(1.8, 0, 3), 2.4, 3, 0.2, wood)
		rl.DrawCubeWires(rl.NewVector3(1.8, 0, 3), 2.4, 3, 0.2, woodDark)
		rl.DrawCube(rl.NewVector3(0, 1.1, 3), 1.2, 0.8, 0.2, wood)
		rl.DrawCubeWires(rl.NewVector3(0, 1.1, 3), 1.2, 0.8, 0.2, woodDark)

		// Дверь (петля слева, поворот при открытии)
		angle := float32(0)
		if h.DoorOpen {
			angle = -80
		}
		rl.PushMatrix()
		rl.Translatef(-0.6, -0.4, 3)
		rl.Rotatef(angle, 0, 1, 0)
		rl.DrawCube(rl.NewVector3(0.6, 0, 0), 1.2, 2.2, 0.1, doorOpen)
		rl.DrawCubeWires(rl.NewVector3(0.6, 0, 0), 1.2, 2.2, 0.1, woodDark)
		rl.PopMatrix()

		rl.PopMatrix()
	}
}

func DrawBoats(boats []protocol.Boat) {
	for _, b := range boats {
		rl.PushMatrix()
		rl.Translatef(b.X, b.Y, b.Z)
		rl.Rotatef(b.Yaw*180.0/3.14159265, 0, 1, 0)
		// Днище
		rl.DrawCube(rl.NewVector3(0, -0.5, 0), 3, 0.4, 6,
			rl.NewColor(120, 80, 40, 255))
		rl.DrawCubeWires(rl.NewVector3(0, -0.5, 0), 3, 0.4, 6, rl.Black)
		// Борта
		rl.DrawCube(rl.NewVector3(-1.4, 0, 0), 0.2, 0.8, 6,
			rl.NewColor(140, 95, 55, 255))
		rl.DrawCube(rl.NewVector3(1.4, 0, 0), 0.2, 0.8, 6,
			rl.NewColor(140, 95, 55, 255))
		// Нос и корма
		rl.DrawCube(rl.NewVector3(0, 0, -2.9), 3, 0.8, 0.2,
			rl.NewColor(140, 95, 55, 255))
		rl.DrawCube(rl.NewVector3(0, 0, 2.9), 3, 0.8, 0.2,
			rl.NewColor(140, 95, 55, 255))
		rl.PopMatrix()
	}
}

// DrawMobs — враждебные/нейтральные боты.
// Чёрный куб 2x2x2 с серыми пятнами на всех гранях — «пиксельная текстура».
func DrawMobs(mobs []protocol.Mob) {
	spot := rl.NewColor(15, 15, 15, 255)
	redEye := rl.NewColor(220, 40, 40, 255)
	wire := rl.NewColor(20, 20, 20, 255)

	for _, m := range mobs {
		pos := rl.NewVector3(m.X, m.Y, m.Z)
		size := float32(2.0)

		// Цвет базы.
		base := rl.NewColor(180, 30, 30, 255) // красный (hostile)
		if m.Kind == "collector" {
			base = rl.NewColor(150, 150, 150, 255) // серый
		} else if m.Kind == "pink" {
			base = rl.NewColor(240, 130, 175, 255) // розовый
		} else if m.Kind != "hostile" {
			base = rl.NewColor(50, 50, 60, 255)
		}

		// Основа.
		rl.DrawCube(pos, size, size, size, base)

		// Пятна — только для hostile/collector.
		if m.Kind != "pink" {
			h := size/2 + 0.02
			s := size * 0.18
			offsets := [][2]float32{
				{-0.6, -0.6}, {-0.6, 0.6},
				{0.6, -0.6}, {0.6, 0.6},
				{0.0, 0.0},
			}
			for _, face := range []int{0, 1, 2, 3, 4, 5} {
				for _, o := range offsets {
					var off rl.Vector3
					a := o[0] * size / 2
					b := o[1] * size / 2
					switch face {
					case 0:
						off = rl.NewVector3(a, b, h)
					case 1:
						off = rl.NewVector3(a, b, -h)
					case 2:
						off = rl.NewVector3(h, a, b)
					case 3:
						off = rl.NewVector3(-h, a, b)
					case 4:
						off = rl.NewVector3(a, h, b)
					case 5:
						off = rl.NewVector3(a, -h, b)
					}
					rl.DrawCube(rl.Vector3Add(pos, off), s, s, s, spot)
				}
			}
		}

		// Обводка куба — всем.
		rl.DrawCubeWires(pos, size, size, size, wire)

		// Красные глаза — только hostile.
		if m.Kind == "hostile" {
			h := size/2 + 0.02
			eyeL := rl.NewVector3(pos.X-size*0.25, pos.Y+size*0.3, pos.Z+h)
			eyeR := rl.NewVector3(pos.X+size*0.25, pos.Y+size*0.3, pos.Z+h)
			rl.DrawCube(eyeL, 0.25, 0.25, 0.1, redEye)
			rl.DrawCube(eyeR, 0.25, 0.25, 0.1, redEye)
		}
	}
}

func DrawProjectiles(projs []protocol.MobProjectile) {
	for _, p := range projs {
		pos := rl.NewVector3(p.X, p.Y, p.Z)
		rl.DrawSphere(pos, 0.3, rl.NewColor(255, 200, 60, 255))
		rl.DrawSphereWires(pos, 0.3, 8, 8, rl.NewColor(180, 120, 20, 255))
	}
}

func DrawSolar(panels []protocol.Solar) {
	for _, s := range panels {
		rl.PushMatrix()
		rl.Translatef(s.X, s.Y, s.Z)
		rl.Rotatef(s.Yaw*180.0/3.14159265, 0, 1, 0)
		// Основание
		rl.DrawCube(rl.NewVector3(0, -0.6, 0), 3, 0.3, 3, rl.NewColor(60, 60, 70, 255))
		// Панель (наклонная — просто плоский куб сверху)
		rl.DrawCube(rl.NewVector3(0, 0, 0), 3, 0.15, 3, rl.NewColor(30, 60, 130, 255))
		rl.DrawCubeWires(rl.NewVector3(0, 0, 0), 3, 0.15, 3, rl.NewColor(20, 40, 90, 255))
		// Солнечные «клетки» (пятна)
		rl.DrawCube(rl.NewVector3(-0.7, 0.1, -0.7), 0.8, 0.05, 0.8, rl.NewColor(60, 130, 220, 255))
		rl.DrawCube(rl.NewVector3(0.7, 0.1, -0.7), 0.8, 0.05, 0.8, rl.NewColor(60, 130, 220, 255))
		rl.DrawCube(rl.NewVector3(-0.7, 0.1, 0.7), 0.8, 0.05, 0.8, rl.NewColor(60, 130, 220, 255))
		rl.DrawCube(rl.NewVector3(0.7, 0.1, 0.7), 0.8, 0.05, 0.8, rl.NewColor(60, 130, 220, 255))
		rl.PopMatrix()
	}
}

func DrawBatteries(batteries []protocol.Battery) {
	for _, b := range batteries {
		rl.PushMatrix()
		rl.Translatef(b.X, b.Y, b.Z)
		rl.Rotatef(b.Yaw*180.0/3.14159265, 0, 1, 0)
		// Корпус
		rl.DrawCube(rl.NewVector3(0, 0, 0), 2, 3, 2, rl.NewColor(60, 60, 80, 255))
		rl.DrawCubeWires(rl.NewVector3(0, 0, 0), 2, 3, 2, rl.NewColor(20, 20, 30, 255))
		// Индикатор заряда — цвет зависит от энергии
		level := float32(0)
		if b.MaxEnergy > 0 {
			level = float32(b.Energy) / float32(b.MaxEnergy)
		}
		var ind rl.Color
		switch {
		case level > 0.66:
			ind = rl.NewColor(60, 220, 100, 255)
		case level > 0.33:
			ind = rl.NewColor(230, 200, 60, 255)
		default:
			ind = rl.NewColor(220, 60, 60, 255)
		}
		rl.DrawCube(rl.NewVector3(0, 1.1, 0), 1.2, 0.4, 1.2, ind)
		rl.PopMatrix()
	}
}

func DrawFactories(factories []protocol.Factory) {
	for _, f := range factories {
		rl.PushMatrix()
		rl.Translatef(f.X, f.Y, f.Z)
		rl.Rotatef(f.Yaw*180.0/3.14159265, 0, 1, 0)
		// Корпус
		rl.DrawCube(rl.NewVector3(0, 0, 0), 4, 3, 4, rl.NewColor(110, 100, 90, 255))
		rl.DrawCubeWires(rl.NewVector3(0, 0, 0), 4, 3, 4, rl.NewColor(60, 50, 40, 255))
		// Трубы
		rl.DrawCube(rl.NewVector3(-1, 2.2, -1), 0.6, 1.5, 0.6, rl.NewColor(80, 70, 60, 255))
		rl.DrawCube(rl.NewVector3(1, 2.2, -1), 0.6, 1.5, 0.6, rl.NewColor(80, 70, 60, 255))
		// Окно-индикатор
		if f.Crafting != "" {
			rl.DrawCube(rl.NewVector3(0, 0.5, 2.05), 1.5, 0.8, 0.1, rl.NewColor(255, 200, 60, 255))
		} else {
			rl.DrawCube(rl.NewVector3(0, 0.5, 2.05), 1.5, 0.8, 0.1, rl.NewColor(40, 80, 40, 255))
		}
		rl.PopMatrix()
	}
}

func DrawRockets(rockets []protocol.Rocket) {
	for _, r := range rockets {
		// Строим ракету вдоль вектора Up (DX,DY,DZ).
		base := rl.NewVector3(r.X, r.Y, r.Z)
		top := rl.NewVector3(
			r.X+r.DX*8, r.Y+r.DY*8, r.Z+r.DZ*8,
		)
		// Фюзеляж
		rl.DrawCylinderEx(base, top, 0.6, 0.6, 8,
			rl.NewColor(220, 220, 230, 255))
		// Нос (конус) — от top до top+2*up
		nosetip := rl.NewVector3(
			r.X+r.DX*11, r.Y+r.DY*11, r.Z+r.DZ*11,
		)
		rl.DrawCylinderEx(top, nosetip, 0.6, 0.05, 8,
			rl.NewColor(200, 60, 60, 255))
		// Сопло
		nozzleBase := rl.NewVector3(
			r.X-r.DX*0.5, r.Y-r.DY*0.5, r.Z-r.DZ*0.5,
		)
		rl.DrawCylinderEx(base, nozzleBase, 0.6, 0.9, 8,
			rl.NewColor(60, 60, 70, 255))

	}
}

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

func DrawWells(wells []protocol.Well) {
	blue := rl.NewColor(40, 110, 220, 255)
	blueDark := rl.NewColor(20, 60, 140, 255)
	for _, w := range wells {
		pos := rl.NewVector3(w.X, w.Y, w.Z)
		// Просто большой яркий куб — проще увидеть, чем цилиндр.
		// 4x4x4 вместо пирамиды, чтобы точно заметить.
		rl.DrawCube(pos, 4.0, 4.0, 4.0, blue)
		rl.DrawCubeWires(pos, 4.0, 4.0, 4.0, blueDark)
	}
}

func DrawMammoths(mammoths []protocol.Mammoth) {
	for _, m := range mammoths {
		pos := rl.NewVector3(m.X, m.Y, m.Z)
		var col rl.Color
		if m.Sex == "f" {
			col = rl.NewColor(150, 110, 80, 255)
		} else {
			col = rl.NewColor(110, 80, 60, 255)
		}
		if m.Tamed {
			col.R += 40
			col.G += 40
			col.B += 40
		}
		size := float32(3.0)
		if m.Baby {
			size = 1.5
		}
		rl.DrawCube(pos, size, size, size, col)
		rl.DrawCubeWires(pos, size, size, size, rl.Black)
		if m.Saddle {
			// небольшой коричневый куб-седло сверху
			saddlePos := rl.NewVector3(m.X, m.Y+size*0.6, m.Z)
			rl.DrawCube(saddlePos, size*0.5, size*0.3, size*0.5,
				rl.NewColor(90, 60, 40, 255))
			rl.DrawCubeWires(saddlePos, size*0.5, size*0.3, size*0.5, rl.Black)
		}
	}
}

func min32(a, b float32) float32 {
	if a < b {
		return a
	}
	return b
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

const shaderVS = `#version 330
in vec3 vertexPosition;
in vec2 vertexTexCoord;
in vec3 vertexNormal;
in vec4 vertexColor;

uniform mat4 mvp;

out float vHeight;
out vec3 vWorldPos;

void main() {
    vHeight = length(vertexPosition);
    vWorldPos = vertexPosition;
    gl_Position = mvp * vec4(vertexPosition, 1.0);
}
`

const shaderFS = `#version 330
in float vHeight;
in vec3 vWorldPos;
out vec4 finalColor;

float hash(vec3 p) {
    p = fract(p * 0.3183099 + 0.1);
    p *= 17.0;
    return fract(p.x * p.y * p.z * (p.x + p.y + p.z));
}

// 3 уровня яркости — выбор по хэшу.
float pixelLevel(vec3 cell, float l1, float l2, float l3) {
    float n = hash(cell);
    if (n < 0.33) return l1;
    if (n < 0.66) return l2;
    return l3;
}

void main() {
    float h = vHeight - 50.0;

    // Мелкие клетки для всех поверхностей.
    vec3 cellFine = floor(vWorldPos * 24.0);   // песок — мелкая крошка
    vec3 cellMed  = floor(vWorldPos * 12.0);   // трава — средние пучки
    vec3 cellCoarse = floor(vWorldPos * 6.0); // камни

    vec3 col;

    if (h < -4.0) {
        // Глубокое дно — тёмный крупный песок.
        float lvl = pixelLevel(cellCoarse, 0.22, 0.26, 0.30);
        col = vec3(lvl * 1.0, lvl * 0.88, lvl * 0.65);
    } else if (h < -0.5) {
        // Мелководье — мокрый песок, мелкая крошка.
        float lvl = pixelLevel(cellFine, 0.60, 0.66, 0.72);
        col = vec3(lvl, lvl * 0.90, lvl * 0.62);
    } else if (h < 1.5) {
        // Пляж — сухой песок, самый светлый.
        // Два уровня детализации: базовая + редкие тёмные крупинки.
        float base = pixelLevel(cellMed, 0.86, 0.90, 0.94);
        float grain = hash(cellFine);
        if (grain > 0.90) {
            base *= 0.82; // редкая тёмная крупинка
        }
        col = vec3(base, base * 0.92, base * 0.68);
    } else if (h < 4.0) {
        // Низкая трава — зелёная с пучками.
        float base = pixelLevel(cellFine, 0.42, 0.52, 0.60);
        float patch = hash(cellMed);
        // Пятна более тёмной/светлой травы.
        if (patch < 0.25) {
            base *= 0.75;
        } else if (patch > 0.75) {
            base *= 1.15;
        }
        col = vec3(base * 0.62, base * 1.05, base * 0.50);
    } else if (h < 7.0) {
        // Средняя трава — темнее, с проплешинами.
        float base = pixelLevel(cellFine, 0.30, 0.40, 0.48);
        float patch = hash(cellMed);
        if (patch < 0.30) {
            base *= 0.7; // проплешина земли
        }
        col = vec3(base * 0.55, base * 0.95, base * 0.45);
    } else {
        // Камни — серый, крупные клетки.
        float base = pixelLevel(cellCoarse, 0.35, 0.45, 0.55);
        float grain = hash(cellFine);
        if (grain > 0.85) {
            base *= 0.75;
        }
        col = vec3(base, base, base * 1.05);
    }

    // Финальное квантование — сильная пиксельная палитра.
    col = floor(col * 6.0 + 0.5) / 6.0;

    finalColor = vec4(col, 1.0);
}
`

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

const waterVS = `#version 330
in vec3 vertexPosition;
in vec2 vertexTexCoord;
in vec3 vertexNormal;
in vec4 vertexColor;

uniform mat4 mvp;
uniform float time;

out vec3 vWorldPos;

void main() {
    vWorldPos = vertexPosition;
    gl_Position = mvp * vec4(vertexPosition, 1.0);
}
`

const waterFS = `#version 330
in vec3 vWorldPos;
out vec4 finalColor;

uniform float time;

float hash(vec3 p) {
    p = fract(p * 0.3183099 + 0.1);
    p *= 17.0;
    return fract(p.x * p.y * p.z * (p.x + p.y + p.z));
}

// Гладкий value-noise с линейной интерполяцией.
float vnoise(vec3 x) {
    vec3 i = floor(x);
    vec3 f = fract(x);
    f = f * f * (3.0 - 2.0 * f);
    return mix(
        mix(mix(hash(i), hash(i + vec3(1,0,0)), f.x),
            mix(hash(i + vec3(0,1,0)), hash(i + vec3(1,1,0)), f.x), f.y),
        mix(mix(hash(i + vec3(0,0,1)), hash(i + vec3(1,0,1)), f.x),
            mix(hash(i + vec3(0,1,1)), hash(i + vec3(1,1,1)), f.x), f.y),
        f.z);
}

// 2 октавы FBM — мягкий, без пиксельных артефактов.
float fbm(vec3 p) {
    float v = 0.5 * vnoise(p);
    v += 0.25 * vnoise(p * 2.02 + vec3(1.7, 9.2, 3.3));
    return v * 1.333;
}

vec3 gradient(float t) {
    t = clamp(t, 0.0, 1.0);
    vec3 c1 = vec3(0.03, 0.09, 0.28);
    vec3 c2 = vec3(0.06, 0.17, 0.42);
    vec3 c3 = vec3(0.10, 0.26, 0.52);
    vec3 c4 = vec3(0.14, 0.35, 0.60);
    vec3 c5 = vec3(0.20, 0.44, 0.66);
    if (t < 0.25) return mix(c1, c2, t / 0.25);
    if (t < 0.50) return mix(c2, c3, (t - 0.25) / 0.25);
    if (t < 0.75) return mix(c3, c4, (t - 0.50) / 0.25);
    return mix(c4, c5, (t - 0.75) / 0.25);
}

void main() {
    vec3 sp = vWorldPos * 0.35;
    float t = time * 0.15;

    // Плавный FBM с медленным сдвигом — кипит на месте.
    vec3 shift = vec3(
        sin(t * 0.22),
        sin(t * 0.17 + 1.7),
        sin(t * 0.19 + 3.1)
    );
    float base = fbm(sp + shift * 0.8);

    // Мелкая рябь.
    vec3 fineShift = vec3(
        sin(t * 0.55 + 1.1),
        sin(t * 0.50 + 3.5),
        sin(t * 0.58 + 5.9)
    ) * 0.9;
    float fine = vnoise(sp * 4.0 + fineShift);

    float w = base * 0.75 + fine * 0.25;
    w = smoothstep(0.20, 0.85, w);

    // Ступеньки: 20 уровней. Больше — мягче переходы.
    const float STEPS = 20.0;
    w = floor(w * STEPS + 0.5) / STEPS;

    vec3 col = gradient(w);

    // Слабый блик на верхнем уровне.
    if (w > 0.85) {
        col += vec3(0.04);
    }

    // 99% непрозрачная вода.
    finalColor = vec4(col, 0.96);
}
`

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

const atmoVS = `#version 330
in vec3 vertexPosition;
in vec2 vertexTexCoord;
in vec3 vertexNormal;
in vec4 vertexColor;

uniform mat4 mvp;

out vec3 vWorldPos;
out vec3 vNormal;

void main() {
    vWorldPos = vertexPosition;
    vNormal = normalize(vertexPosition);
    gl_Position = mvp * vec4(vertexPosition, 1.0);
}
`

const atmoFS = `#version 330
in vec3 vWorldPos;
in vec3 vNormal;
out vec4 finalColor;

uniform vec3 cameraPos;

void main() {
    float dist = length(vWorldPos);
    float sea = 50.0;
    float atmoTop = 70.0;
    float h = (dist - sea) / (atmoTop - sea);
    h = clamp(h, 0.0, 1.0);

    float density = pow(1.0 - h, 2.0);

    vec3 viewDir = normalize(cameraPos - vWorldPos);
    float fresnel = 1.0 - abs(dot(viewDir, vNormal));
    fresnel = pow(fresnel, 2.0);

    // Цвет атмосферы: голубой у поверхности → синий выше.
    vec3 lowCol  = vec3(0.55, 0.78, 1.00);
    vec3 highCol = vec3(0.25, 0.50, 0.90);
    vec3 col = mix(lowCol, highCol, h);

    // ДИАГНОСТИКА: alpha = 0.5 всегда, чтобы видеть есть ли шейдер.
    float alpha = 0.5;
    finalColor = vec4(col, alpha);
}
`

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

const cloudsVS = `#version 330
in vec3 vertexPosition;
in vec2 vertexTexCoord;
in vec3 vertexNormal;
in vec4 vertexColor;

uniform mat4 mvp;

out vec3 vWorldPos;
out vec3 vNormal;

void main() {
    vWorldPos = vertexPosition;
    vNormal = normalize(vertexPosition);
    gl_Position = mvp * vec4(vertexPosition, 1.0);
}
`

const cloudsFS = `#version 330
in vec3 vWorldPos;
in vec3 vNormal;
out vec4 finalColor;

uniform float time;

float hash(vec3 p) {
    p = fract(p * 0.3183099 + 0.1);
    p *= 17.0;
    return fract(p.x * p.y * p.z * (p.x + p.y + p.z));
}

float vnoise(vec3 x) {
    vec3 i = floor(x);
    vec3 f = fract(x);
    f = f * f * (3.0 - 2.0 * f);
    return mix(
        mix(mix(hash(i), hash(i + vec3(1,0,0)), f.x),
            mix(hash(i + vec3(0,1,0)), hash(i + vec3(1,1,0)), f.x), f.y),
        mix(mix(hash(i + vec3(0,0,1)), hash(i + vec3(1,0,1)), f.x),
            mix(hash(i + vec3(0,1,1)), hash(i + vec3(1,1,1)), f.x), f.y),
        f.z);
}

float fbm(vec3 p) {
    float v = 0.5 * vnoise(p);
    v += 0.25 * vnoise(p * 2.03 + vec3(1.7, 9.2, 3.3));
    v += 0.125 * vnoise(p * 4.05 + vec3(5.1, 2.8, 7.4));
    return v * 1.143;
}

// Domain warping — сильно меняет форму со временем.
// Внешний шум сдвигает координаты сэмпла внутреннего шума.
// Это даёт "кипящие" облака — они реально меняют очертания.
float warpedFBM(vec3 p, float t) {
    // Медленное течение для warping.
    vec3 w1 = vec3(
        vnoise(p * 0.8 + vec3(t * 0.05, 0.0, 0.0)),
        vnoise(p * 0.8 + vec3(0.0, t * 0.05, 0.0)),
        vnoise(p * 0.8 + vec3(0.0, 0.0, t * 0.05))
    );

    // Второй уровень warping — медленнее, шире.
    vec3 w2 = vec3(
        vnoise(p * 0.4 + vec3(t * 0.02 + 3.7, 0.0, 0.0)),
        vnoise(p * 0.4 + vec3(0.0, t * 0.02 + 5.1, 0.0)),
        vnoise(p * 0.4 + vec3(0.0, 0.0, t * 0.02 + 7.3))
    );

    // Сэмплим FBM в warped координатах — форма меняется.
    vec3 warped = p + (w1 - 0.5) * 0.9 + (w2 - 0.5) * 1.4;

    return fbm(warped);
}

void main() {
    // ОЧЕНЬ медленное вращение — облака еле дрейфуют.
    float angle = time * 0.008; // ~13 минут на оборот
    float ca = cos(angle);
    float sa = sin(angle);

    vec3 rotated = vec3(
        vWorldPos.x * ca - vWorldPos.z * sa,
        vWorldPos.y,
        vWorldPos.x * sa + vWorldPos.z * ca
    );

    vec3 sp = rotated * 0.6;

    // Меняющаяся форма через domain warping.
    float n = warpedFBM(sp, time);

    float clouds = smoothstep(0.45, 0.70, n);

    // Плотность: медленная пульсация.
    float density = clouds * (0.60 + 0.10 * sin(time * 0.15));
    density = clamp(density, 0.0, 0.75);

    vec3 col = vec3(0.95, 0.97, 1.00);

    finalColor = vec4(col, density);
}
`
