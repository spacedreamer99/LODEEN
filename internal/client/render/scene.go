package render

import (
	rl "github.com/gen2brain/raylib-go/raylib"

	"github.com/spacedreamer99/lodeen/internal/client/fonts"
	"github.com/spacedreamer99/lodeen/internal/shared/protocol"
)

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

func DrawPlayers(players []protocol.PlayerState, ownID string, camera rl.Camera3D) {
	for _, p := range players {
		if p.ID == ownID {
			continue
		}
		pos := rl.NewVector3(p.X, p.Y, p.Z)
		col := colorForID(p.ID)

		rl.DrawCube(pos, 1.2, 1.2, 1.2, col)
		rl.DrawCubeWires(pos, 1.2, 1.2, 1.2, rl.Black)

		label := p.Nick
		if label == "" {
			label = p.ID
		}
		labelPos := rl.NewVector3(pos.X, pos.Y+1.5, pos.Z)
		screen := rl.GetWorldToScreen(labelPos, camera)
		if screen.X > 0 && screen.Y > 0 {
			tw := fonts.Measure(label, 16)
			fonts.Draw(label, int32(screen.X)-tw/2, int32(screen.Y), 16, rl.RayWhite)
		}
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
	x := float32(int32(h%400) - 200)
	z := float32(int32((h>>9)%400) - 200)
	return rl.NewVector3(x, 5, z)
}
