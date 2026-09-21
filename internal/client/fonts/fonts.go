package fonts

import (
	"os"

	rl "github.com/gen2brain/raylib-go/raylib"
)

var Default rl.Font

func Load(size int32) rl.Font {
	paths := []string{
		"/usr/share/fonts/truetype/dejavu/DejaVuSansMono.ttf",
		"/usr/share/fonts/TTF/DejaVuSansMono.ttf",
		"/usr/share/fonts/dejavu/DejaVuSansMono.ttf",
		"/usr/share/fonts/truetype/liberation/LiberationMono-Regular.ttf",
		"/usr/share/fonts/truetype/ubuntu/UbuntuMono-R.ttf",
	}
	for _, p := range paths {
		if _, err := os.Stat(p); err == nil {
			f := rl.LoadFontEx(p, size, nil, 0)
			if f.Texture.ID != 0 {
				Default = f
				return f
			}
		}
	}
	f := rl.GetFontDefault()
	rl.SetTextureFilter(f.Texture, rl.FilterBilinear)
	Default = f
	return f
}

func Draw(text string, x, y, size int32, color rl.Color) {
	if Default.Texture.ID == 0 {
		Default = rl.GetFontDefault()
	}
	rl.DrawTextEx(Default, text, rl.NewVector2(float32(x), float32(y)), float32(size), 0, color)
}

func Measure(text string, size int32) int32 {
	if Default.Texture.ID == 0 {
		Default = rl.GetFontDefault()
	}
	v := rl.MeasureTextEx(Default, text, float32(size), 0)
	return int32(v.X)
}
