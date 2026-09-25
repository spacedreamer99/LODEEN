package ui

import rl "github.com/gen2brain/raylib-go/raylib"

type Anchor int

const (
	TopLeft Anchor = iota
	TopRight
	BottomLeft
	BottomRight
	Center
	CenterTop
	CenterBottom
)

// Place возвращает координаты верхнего левого угла элемента размера (w, h),
// привязанного к якорю, с отступом (offX, offY) от угла окна.
// Использует реальный размер окна (rl.GetScreenWidth/Height), поэтому
// элементы остаются в углах при любых пропорциях.
func Place(a Anchor, offX, offY, w, h int32) (int32, int32) {
	sw := int32(rl.GetScreenWidth())
	sh := int32(rl.GetScreenHeight())

	switch a {
	case TopLeft:
		return offX, offY
	case TopRight:
		return sw - w - offX, offY
	case BottomLeft:
		return offX, sh - h - offY
	case BottomRight:
		return sw - w - offX, sh - h - offY
	case Center:
		return (sw - w) / 2, (sh - h) / 2
	case CenterTop:
		return (sw - w) / 2, offY
	case CenterBottom:
		return (sw - w) / 2, sh - h - offY
	}
	return offX, offY
}
