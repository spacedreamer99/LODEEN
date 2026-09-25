package ui

import (
	rl "github.com/gen2brain/raylib-go/raylib"

	"github.com/spacedreamer99/lodeen/internal/client/fonts"
)

var (
	mouseScaleX float32 = 1
	mouseScaleY float32 = 1
)

// SetMouseScale устанавливает коэффициент пересчёта координат мыши
// из реального окна в виртуальный UI-буфер.
func SetMouseScale(sx, sy float32) {
	if sx > 0 {
		mouseScaleX = sx
	}
	if sy > 0 {
		mouseScaleY = sy
	}
}

func mousePos() rl.Vector2 {
	m := rl.GetMousePosition()
	return rl.NewVector2(m.X/mouseScaleX, m.Y/mouseScaleY)
}

type Button struct {
	Rect rl.Rectangle
	Text string
}

func (b Button) Hovered() bool {
	return rl.CheckCollisionPointRec(rl.GetMousePosition(), b.Rect)
}

func (b Button) Clicked() bool {
	return b.Hovered() && rl.IsMouseButtonPressed(rl.MouseLeftButton)
}

func (b Button) Draw() {
	fill := rl.NewColor(35, 35, 45, 230)
	if b.Hovered() {
		fill = rl.NewColor(70, 70, 95, 245)
	}
	rl.DrawRectangleRec(b.Rect, fill)
	rl.DrawRectangleLinesEx(b.Rect, 1, rl.NewColor(120, 120, 140, 255))

	tw := fonts.Measure(b.Text, 20)
	tx := int32(b.Rect.X) + (int32(b.Rect.Width)-tw)/2
	ty := int32(b.Rect.Y) + (int32(b.Rect.Height)-20)/2
	fonts.Draw(b.Text, tx, ty, 20, rl.White)
}

func TextField(rect rl.Rectangle, value string, focused bool) bool {
	fill := rl.NewColor(25, 25, 35, 255)
	border := rl.Gray
	if focused {
		fill = rl.NewColor(35, 35, 50, 255)
		border = rl.SkyBlue
	}
	rl.DrawRectangleRec(rect, fill)
	rl.DrawRectangleLinesEx(rect, 1, border)
	label := value
	if focused {
		label += "_"
	}
	fonts.Draw(label, int32(rect.X)+8, int32(rect.Y)+6, 20, rl.White)
	return rl.CheckCollisionPointRec(rl.GetMousePosition(), rect) &&
		rl.IsMouseButtonPressed(rl.MouseLeftButton)
}

func EditField(value *string, focused bool, maxLen int) {
	if !focused {
		return
	}
	for {
		ch := rl.GetCharPressed()
		if ch == 0 {
			break
		}
		if ch >= 32 && ch < 127 && len(*value) < maxLen {
			*value += string(ch)
		}
	}
	if rl.IsKeyPressed(rl.KeyBackspace) && len(*value) > 0 {
		*value = (*value)[:len(*value)-1]
	}
}
