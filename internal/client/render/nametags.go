package render

import (
	rl "github.com/gen2brain/raylib-go/raylib"

	"github.com/spacedreamer99/lodeen/internal/client/fonts"
	"github.com/spacedreamer99/lodeen/internal/shared/protocol"
)

// DrawPlayerNameTags рисует ник и цветовой маркер над каждым игроком.
// Вызывать ПОСЛЕ EndMode3D (2D-контекст), внутри UI-таргета.
//
//	earthPos — позиция Земли в helio (Pos игроков локальны относительно Земли).
//	screenW/H — размер текущего render-target'а.
func DrawPlayerNameTags(players []protocol.PlayerState, ownID string, cam rl.Camera3D, earthPos rl.Vector3, screenW, screenH int32) {
	camForward := rl.Vector3Normalize(rl.Vector3Subtract(cam.Target, cam.Position))

	for _, p := range players {
		if p.ID == ownID {
			continue
		}

		// Голова игрока в helio.
		local := rl.NewVector3(p.X, p.Y, p.Z)
		up := rl.Vector3Normalize(local)
		headHelio := rl.Vector3Add(
			rl.Vector3Add(local, rl.Vector3Scale(up, 1.7)),
			earthPos,
		)

		// Отсеиваем игроков за камерой (иначе их ники ошибочно на экране).
		toPlayer := rl.Vector3Normalize(rl.Vector3Subtract(headHelio, cam.Position))
		if rl.Vector3DotProduct(camForward, toPlayer) < 0.1 {
			continue
		}

		// Проекция в экранные пиксели.
		screen := rl.GetWorldToScreen(headHelio, cam)
		if screen.X < 0 || screen.X > float32(screenW) ||
			screen.Y < 0 || screen.Y > float32(screenH) {
			continue
		}

		// Затухание по расстоянию: ближе 25м — полностью, дальше 100м — скрыто.
		dist := rl.Vector3Distance(cam.Position, headHelio)
		const near = 25.0
		const far = 100.0
		if dist > far {
			continue
		}
		alpha := uint8(255)
		if dist > near {
			t := float32((dist - near) / (far - near))
			alpha = uint8(255 * (1 - t))
			if alpha < 50 {
				continue
			}
		}

		// Имя (fallback — короткий ID).
		name := p.Nick
		if name == "" {
			if len(p.ID) >= 6 {
				name = p.ID[:6]
			} else {
				name = p.ID
			}
		}

		// Цвет игрока.
		c, ok := HexColor(p.ColorHex)
		if !ok {
			c = colorForID(p.ID)
		}
		c.A = alpha

		// Размеры плашки.
		const fontSize = int32(16)
		const tag = int32(10)
		const pad = int32(5)
		tw := fonts.Measure(name, fontSize)
		bgW := pad + tag + pad + tw + pad
		bgH := fontSize + 6

		bgX := int32(screen.X) - bgW/2
		bgY := int32(screen.Y) - bgH - 4

		// Фон и рамка.
		bg := rl.NewColor(0, 0, 0, uint8(160)*alpha/255)
		border := rl.NewColor(255, 255, 255, alpha/2)
		rl.DrawRectangle(bgX, bgY, bgW, bgH, bg)
		rl.DrawRectangleLines(bgX, bgY, bgW, bgH, border)

		// Цветной маркер слева.
		rl.DrawRectangle(bgX+pad, bgY+3, tag, tag, c)
		rl.DrawRectangleLines(bgX+pad, bgY+3, tag, tag, rl.NewColor(0, 0, 0, alpha))

		// Имя белым.
		textColor := rl.NewColor(255, 255, 255, alpha)
		fonts.Draw(name, bgX+pad+tag+pad, bgY+3, fontSize, textColor)
	}
}
