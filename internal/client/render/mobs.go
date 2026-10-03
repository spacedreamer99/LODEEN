package render

import (
	rl "github.com/gen2brain/raylib-go/raylib"
	"math"

	"github.com/spacedreamer99/lodeen/internal/shared/protocol"
)

// DrawMobs — враждебные/нейтральные боты.
// Чёрный куб 2x2x2 с серыми пятнами на всех гранях — «пиксельная текстура».
// mobAnim — счётчик анимации. phase растёт только когда моб двигается.
type mobAnim struct {
	phase    float32
	lastPos  rl.Vector3
	lastMove float64
}

var mobAnims = make(map[string]*mobAnim)

func DrawMobs(mobs []protocol.Mob) {
	now := rl.GetTime()
	dt := rl.GetFrameTime()

	for _, m := range mobs {
		pos := rl.NewVector3(m.X, m.Y, m.Z)
		up := rl.Vector3Normalize(pos)

		var base, accent rl.Color
		hasHorns := false
		switch m.Kind {
		case "hostile":
			base = rl.NewColor(150, 25, 25, 255)
			accent = rl.NewColor(200, 60, 40, 255)
			hasHorns = true
		case "collector":
			base = rl.NewColor(30, 100, 45, 255)
			accent = rl.NewColor(60, 150, 70, 255)
		case "pink":
			base = rl.NewColor(240, 130, 175, 255)
			accent = rl.NewColor(255, 190, 220, 255)
		default:
			base = rl.NewColor(60, 60, 70, 255)
			accent = rl.NewColor(90, 90, 100, 255)
		}

		anim := mobAnims[m.ID]
		if anim == nil {
			anim = &mobAnim{lastPos: pos, lastMove: now - 10}
			mobAnims[m.ID] = anim
		}
		if rl.Vector3Distance(anim.lastPos, pos) > 0.02 {
			anim.lastPos = pos
			anim.lastMove = now
		}
		if now-anim.lastMove < 0.15 {
			anim.phase += float32(dt) * 2.0
		}

		drawHumanoidMob(pos, up, base, accent, anim.phase, hasHorns)
	}
}

// drawHumanoidMob — человекообразный моб, всегда ногами к up (нормали планеты).
func drawHumanoidMob(pos, up rl.Vector3, base, accent rl.Color, walkPhase float32, hasHorns bool) {
	dark := rl.NewColor(20, 10, 10, 255)
	hornCol := rl.NewColor(235, 225, 200, 255)
	eyeCol := rl.NewColor(255, 110, 20, 255)

	// Ориентация: локальный +Y → мировой up.
	rl.PushMatrix()
	// pos приходит как центр модели (старый куб). Сдвигаем вниз на 0.75,
	// чтобы ноги humanoid касались поверхности а не висели в воздухе.
	rl.Translatef(pos.X, pos.Y-0.75, pos.Z)
	upN := rl.Vector3Normalize(up)
	yUp := rl.NewVector3(0, 1, 0)
	if rl.Vector3Distance(upN, yUp) > 0.001 {
		if rl.Vector3Distance(upN, rl.NewVector3(0, -1, 0)) < 0.001 {
			rl.Rotatef(180, 1, 0, 0)
		} else {
			axis := rl.Vector3CrossProduct(yUp, upN)
			axisLen := rl.Vector3Length(axis)
			if axisLen > 0.0001 {
				axis = rl.Vector3Scale(axis, 1/axisLen)
				angle := float32(math.Acos(float64(upN.Y))) * 180 / math.Pi
				rl.Rotatef(angle, axis.X, axis.Y, axis.Z)
			}
		}
	}

	sw := float32(math.Sin(float64(walkPhase)))
	legL := sw * 0.35
	legR := -legL
	armL := -legL * 0.6
	armR := -legR * 0.6
	bob := float32(math.Abs(math.Sin(float64(walkPhase*2)))) * 0.03
	hipY := 0.75 + bob

	// НОГИ
	hipL := rl.NewVector3(-0.12, hipY, 0)
	rl.DrawSphere(hipL, 0.09, dark)
	kneeL := drawLimbSegment(hipL, legL, 0.38, 0.1, base)
	rl.DrawSphere(kneeL, 0.08, dark)
	ankleL := drawLimbSegment(kneeL, -legL*0.55, 0.38, 0.085, base)
	rl.DrawCube(rl.NewVector3(ankleL.X, ankleL.Y-0.03, ankleL.Z+0.06), 0.14, 0.06, 0.22, dark)

	hipR := rl.NewVector3(0.12, hipY, 0)
	rl.DrawSphere(hipR, 0.09, dark)
	kneeR := drawLimbSegment(hipR, legR, 0.38, 0.1, base)
	rl.DrawSphere(kneeR, 0.08, dark)
	ankleR := drawLimbSegment(kneeR, -legR*0.55, 0.38, 0.085, base)
	rl.DrawCube(rl.NewVector3(ankleR.X, ankleR.Y-0.03, ankleR.Z+0.06), 0.14, 0.06, 0.22, dark)

	// ТОРС
	rl.DrawCube(rl.NewVector3(0, hipY+0.3, 0), 0.42, 0.6, 0.26, base)
	rl.DrawCube(rl.NewVector3(0, hipY+0.32, 0.14), 0.32, 0.4, 0.02, accent)

	shoulderY := hipY + 0.6
	shL := rl.NewVector3(-0.28, shoulderY, 0)
	shR := rl.NewVector3(0.28, shoulderY, 0)
	rl.DrawSphere(shL, 0.11, dark)
	rl.DrawSphere(shR, 0.11, dark)

	// РУКИ
	elbowL := drawLimbSegment(shL, armL, 0.32, 0.075, base)
	rl.DrawSphere(elbowL, 0.07, dark)
	_ = drawLimbSegment(elbowL, armL+0.5, 0.3, 0.065, base)

	elbowR := drawLimbSegment(shR, armR, 0.32, 0.075, base)
	rl.DrawSphere(elbowR, 0.07, dark)
	_ = drawLimbSegment(elbowR, armR+0.5, 0.3, 0.065, base)

	// ШЕЯ
	neckY := shoulderY + 0.14
	rl.DrawCube(rl.NewVector3(0, neckY, 0), 0.13, 0.12, 0.13, dark)

	// ГОЛОВА
	headY := neckY + 0.2
	rl.DrawSphere(rl.NewVector3(0, headY, 0), 0.22, base)
	rl.DrawSphere(rl.NewVector3(-0.08, headY+0.03, 0.19), 0.045, eyeCol)
	rl.DrawSphere(rl.NewVector3(0.08, headY+0.03, 0.19), 0.045, eyeCol)

	// РОГА (только hostile)
	if hasHorns {
		rl.PushMatrix()
		rl.Translatef(-0.17, headY+0.12, 0)
		rl.Rotatef(45, 0, 0, 1)
		rl.Rotatef(-20, 1, 0, 0)
		rl.DrawCylinder(rl.NewVector3(0, 0.16, 0), 0.012, 0.045, 0.32, 8, hornCol)
		rl.DrawCylinder(rl.NewVector3(0, 0.38, 0), 0.001, 0.012, 0.14, 8, hornCol)
		rl.PopMatrix()

		rl.PushMatrix()
		rl.Translatef(0.17, headY+0.12, 0)
		rl.Rotatef(-45, 0, 0, 1)
		rl.Rotatef(-20, 1, 0, 0)
		rl.DrawCylinder(rl.NewVector3(0, 0.16, 0), 0.012, 0.045, 0.32, 8, hornCol)
		rl.DrawCylinder(rl.NewVector3(0, 0.38, 0), 0.001, 0.012, 0.14, 8, hornCol)
		rl.PopMatrix()
	}

	rl.PopMatrix()
}

// drawLimbSegment — одна "кость". swing=0 → вниз, >0 → вперёд (+Z).
func drawLimbSegment(pivot rl.Vector3, swing, length, r float32, col rl.Color) rl.Vector3 {
	endX := pivot.X
	endY := pivot.Y - length*float32(math.Cos(float64(swing)))
	endZ := pivot.Z + length*float32(math.Sin(float64(swing)))

	rl.PushMatrix()
	rl.Translatef(pivot.X, pivot.Y, pivot.Z)
	rl.Rotatef(float32(-float64(swing)*180/math.Pi), 1, 0, 0)
	rl.DrawCylinder(rl.NewVector3(0, -length/2, 0), r*0.85, r, length, 8, col)
	rl.PopMatrix()

	return rl.NewVector3(endX, endY, endZ)
}
