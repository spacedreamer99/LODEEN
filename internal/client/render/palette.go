package render

import (
	"math"
	"strconv"

	rl "github.com/gen2brain/raylib-go/raylib"

	"github.com/spacedreamer99/lodeen/internal/shared/protocol"
)

// Palette — 12 контрастных цветов для выбора в меню.
// Формат "#RRGGBB" (lowercase).
var Palette = []string{
	// Ряд 1: яркие основные
	"#e62937", // 0  красный
	"#ff851b", // 1  оранжевый
	"#ffcb00", // 2  жёлтый
	"#00e430", // 3  зелёный

	// Ряд 2: холодные и смешанные
	"#00b8a9", // 4  бирюзовый
	"#0079f1", // 5  синий
	"#8250dc", // 6  фиолетовый
	"#ff6dc2", // 7  розовый

	// Ряд 3: тёмные насыщенные
	"#8b0000", // 8  тёмно-красный
	"#006400", // 9  тёмно-зелёный
	"#00008b", // 10 тёмно-синий
	"#5a3a1a", // 11 коричневый

	// Ряд 4: нейтральные
	"#f0f0f0", // 12 белый
	"#a0a0a0", // 13 серый
	"#505050", // 14 тёмно-серый
	"#0a0a0a", // 15 чёрный
}

// PaletteNames — названия для UI (тот же индекс что Palette).
var PaletteNames = []string{
	"Красный", "Оранжевый", "Жёлтый", "Зелёный",
	"Бирюзовый", "Синий", "Фиолетовый", "Розовый",
	"Тёмно-красный", "Тёмно-зелёный", "Тёмно-синий", "Коричневый",
	"Белый", "Серый", "Тёмно-серый", "Чёрный",
}

// DefaultColor — цвет, который показывается первым в меню.
const DefaultColor = "#e62937"

// HexColor парсит "#RRGGBB" в rl.Color. Возвращает ok=false, если формат не тот.
// Используется и в рендере (для игроков), и в UI (для свотчей в меню).
func HexColor(s string) (rl.Color, bool) {
	if len(s) != 7 || s[0] != '#' {
		return rl.Color{}, false
	}
	v, err := strconv.ParseUint(s[1:], 16, 32)
	if err != nil {
		return rl.Color{}, false
	}
	return rl.NewColor(uint8(v>>16), uint8(v>>8), uint8(v), 255), true
}

// playerColor — цвет игрока: из PlayerState.ColorHex, если валиден,
// иначе детерминированный по ID.
func playerColor(p protocol.PlayerState) rl.Color {
	if c, ok := HexColor(p.ColorHex); ok {
		return c
	}
	return colorForID(p.ID)
}

// HSVToColor конвертирует HSV (h 0..360, s/v 0..1) в rl.Color.
func HSVToColor(h, s, v float32) rl.Color {
	if s < 0 {
		s = 0
	}
	if s > 1 {
		s = 1
	}
	if v < 0 {
		v = 0
	}
	if v > 1 {
		v = 1
	}
	h = float32(math.Mod(float64(h), 360))
	if h < 0 {
		h += 360
	}
	if s == 0 {
		c := uint8(v * 255)
		return rl.NewColor(c, c, c, 255)
	}
	hh := h / 60
	i := int(hh)
	f := hh - float32(i)
	pv := v * (1 - s)
	q := v * (1 - s*f)
	t := v * (1 - s*(1-f))
	var r, g, b float32
	switch i {
	case 0:
		r, g, b = v, t, pv
	case 1:
		r, g, b = q, v, pv
	case 2:
		r, g, b = pv, v, t
	case 3:
		r, g, b = pv, q, v
	case 4:
		r, g, b = t, pv, v
	case 5:
		r, g, b = v, pv, q
	}
	return rl.NewColor(uint8(r*255), uint8(g*255), uint8(b*255), 255)
}

// HexFromHSV — HSV → "#rrggbb".
func HexFromHSV(h, s, v float32) string {
	c := HSVToColor(h, s, v)
	return "#" + hex2(c.R) + hex2(c.G) + hex2(c.B)
}

func hex2(v uint8) string {
	const d = "0123456789abcdef"
	return string([]byte{d[v>>4], d[v&0xf]})
}
