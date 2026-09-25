package render

import (
	rl "github.com/gen2brain/raylib-go/raylib"

	"github.com/spacedreamer99/lodeen/internal/shared/protocol"
)

const playerCubeSize = 5.0

type Scene struct {
	planet    rl.Model
	hasPlanet bool

	planetPos   rl.Vector3
	planetScale float32
}

func NewScene(planetPath string) *Scene {
	s := &Scene{
		planetPos:   rl.Vector3Zero(),
		planetScale: 1.0,
	}
	if planetPath != "" {
		m := rl.LoadModel(planetPath)
		if m.MeshCount > 0 {
			s.planet = m
			s.hasPlanet = true
		}
	}
	return s
}

func (s *Scene) Unload() {
	if s.hasPlanet {
		rl.UnloadModel(s.planet)
	}
}

func (s *Scene) HasPlanet() bool { return s.hasPlanet }

func (s *Scene) SetPlanetScale(scale float32) { s.planetScale = scale }

func (s *Scene) Draw() {
	rl.DrawLine3D(rl.Vector3Zero(), rl.NewVector3(100, 0, 0), rl.Red)
	rl.DrawLine3D(rl.Vector3Zero(), rl.NewVector3(0, 100, 0), rl.Green)
	rl.DrawLine3D(rl.Vector3Zero(), rl.NewVector3(0, 0, 100), rl.Blue)

	if s.hasPlanet {
		rl.DrawModel(s.planet, s.planetPos, s.planetScale, rl.White)
	} else {
		rl.DrawSphere(rl.Vector3Zero(), 20, rl.NewColor(60, 90, 140, 255))
		rl.DrawSphereWires(rl.Vector3Zero(), 20, 24, 24, rl.NewColor(120, 160, 200, 120))
	}
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
		default:
			col = rl.White
		}
		rl.DrawCube(pos, 1.0, 1.0, 1.0, col)
		rl.DrawCubeWires(pos, 1.0, 1.0, 1.0, rl.Black)
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
	var h uint32 = 2166136261
	for i := 0; i < len(id); i++ {
		h ^= uint32(id[i])
		h *= 16777619
	}
	dx := float32(int32(h%13) - 6)
	return rl.NewVector3(dx, 30, 60)
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
