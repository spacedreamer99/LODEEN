package fonts

import (
	"image"
	"image/color"
	"runtime"
	"sync"

	rl "github.com/gen2brain/raylib-go/raylib"
	"github.com/gogpu/gg"
	"github.com/gogpu/gg/text"
)

// supersample — рендерим текст в N раз больше, показываем в 1/N.
// Даёт N² сэмплов на пиксель — гладкие края даже на мелком тексте.
const supersample = 2.0
const textPad = 8

var (
	fontSource *text.FontSource
	mu         sync.Mutex
	cache      = map[cacheKey]rl.Texture2D{}
)

type cacheKey struct {
	text string
	size int32
}

func Load(_ int32) rl.Font {
	paths := []string{
		"/usr/share/fonts/truetype/dejavu/DejaVuSansMono-Bold.ttf",
		"/usr/share/fonts/truetype/dejavu/DejaVuSansMono.ttf",
		"/usr/share/fonts/truetype/liberation/LiberationMono-Bold.ttf",
	}
	for _, p := range paths {
		src, err := text.NewFontSourceFromFile(p)
		if err == nil {
			fontSource = src
			break
		}
	}
	return rl.GetFontDefault()
}

func Draw(text string, x, y, size int32, col rl.Color) {
	if text == "" {
		return
	}
	tex, _, _ := getTexture(text, size, col)
	if tex.ID == 0 {
		rl.DrawTextEx(rl.GetFontDefault(), text,
			rl.NewVector2(float32(x), float32(y)), float32(size), 0, col)
		return
	}
	// Сжимаем текстуру с масштабом 1/supersample.
	rl.DrawTextureEx(tex, rl.NewVector2(float32(x), float32(y)), 0, 1.0/supersample, rl.White)
}

func Measure(text string, size int32) int32 {
	if text == "" {
		return 0
	}
	_, w, _ := getTexture(text, size, rl.White)
	return int32(float64(w) / supersample)
}

func getTexture(text string, size int32, col rl.Color) (rl.Texture2D, int, int) {
	key := cacheKey{text: text, size: size}
	mu.Lock()
	if t, ok := cache[key]; ok {
		mu.Unlock()
		return t, int(t.Width) - textPad*2, int(t.Height) - textPad*2
	}
	mu.Unlock()

	if fontSource == nil {
		return rl.Texture2D{}, 0, 0
	}

	// Рендерим в supersample раз больше.
	renderSize := float64(size) * supersample
	face := fontSource.Face(renderSize)

	tmp := gg.NewContext(1, 1)
	tmp.SetFont(face)
	w, h := tmp.MeasureString(text)

	pad := textPad
	dc := gg.NewContext(int(w)+pad*2, int(h)+pad*2)
	dc.SetFont(face)
	dc.SetColor(color.NRGBA{R: col.R, G: col.G, B: col.B, A: col.A})

	// (0,0) = верхний левый угол элемента.
	dc.DrawStringAnchored(text, float64(pad), float64(pad), 0, 0)

	img := dc.Image()
	rgba, ok := img.(*image.RGBA)
	if !ok {
		return rl.Texture2D{}, 0, 0
	}

	var pinner runtime.Pinner
	pinner.Pin(&rgba.Pix[0])
	defer pinner.Unpin()

	rlImg := rl.NewImage(rgba.Pix, int32(rgba.Rect.Dx()), int32(rgba.Rect.Dy()), 1, rl.UncompressedR8g8b8a8)
	tex := rl.LoadTextureFromImage(rlImg)
	rl.GenTextureMipmaps(&tex)
	rl.SetTextureFilter(tex, rl.FilterTrilinear)

	mu.Lock()
	cache[key] = tex
	mu.Unlock()

	return tex, int(w), int(h)
}

// DebugDump сохраняет рендеры текста для визуальной проверки.
func DebugDump() {
	for _, s := range []int32{16, 20, 32, 64} {
		tex, w, h := getTexture("Ag0", s, rl.White)
		if tex.ID == 0 {
			println("no texture for size", s)
			continue
		}
		img := rl.LoadImageFromTexture(tex)
		path := "/tmp/font_debug_" + itoa(int(s)) + ".png"
		rl.ExportImage(*img, path)
		rl.UnloadImage(img)
		println("dumped", path, "logical:", w, h, "tex:", tex.Width, tex.Height)
	}
}

func itoa(n int) string {
	if n == 0 {
		return "0"
	}
	neg := n < 0
	if neg {
		n = -n
	}
	var b [20]byte
	i := len(b)
	for n > 0 {
		i--
		b[i] = byte('0' + n%10)
		n /= 10
	}
	if neg {
		i--
		b[i] = '-'
	}
	return string(b[i:])
}
