package render

import (
	rl "github.com/gen2brain/raylib-go/raylib"

	"github.com/spacedreamer99/lodeen/internal/shared/protocol"
)

func DrawPlayers(players []protocol.PlayerState, ownID string, _ rl.Camera3D) {
	now := rl.GetTime()
	dt := rl.GetFrameTime()
	for _, p := range players {
		if p.ID == ownID {
			continue
		}
		pos := rl.NewVector3(p.X, p.Y, p.Z)
		up := rl.Vector3Normalize(pos)

		base := playerColor(p)
		accent := rl.NewColor(
			uint8(float32(base.R)*0.65),
			uint8(float32(base.G)*0.65),
			uint8(float32(base.B)*0.65),
			255,
		)

		anim := mobAnims[p.ID]
		if anim == nil {
			anim = &mobAnim{lastPos: pos, lastMove: now - 10}
			mobAnims[p.ID] = anim
		}
		if rl.Vector3Distance(anim.lastPos, pos) > 0.05 {
			anim.lastPos = pos
			anim.lastMove = now
		}
		if now-anim.lastMove < 0.15 {
			anim.phase += float32(dt) * 2.0
		}

		drawHumanoidMob(pos, up, base, accent, anim.phase, false)
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
			size = 1.4 // заметный росток, а не точка
		}
		rl.DrawCube(pos, size, size, size, col)
		rl.DrawCubeWires(pos, size, size, size, rl.Black)
	}
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
