package chat

import (
	rl "github.com/gen2brain/raylib-go/raylib"

	"github.com/spacedreamer99/lodeen/internal/client/fonts"
	"github.com/spacedreamer99/lodeen/internal/shared/protocol"
)

const (
	maxLines = 8
	maxInput = 200
	lineH    = 22
)

type Chat struct {
	Lines []protocol.ChatMessage
	Input string
	Open  bool
}

func New() *Chat { return &Chat{} }

func (c *Chat) Push(m protocol.ChatMessage) {
	c.Lines = append(c.Lines, m)
	if len(c.Lines) > maxLines {
		c.Lines = c.Lines[len(c.Lines)-maxLines:]
	}
}

func (c *Chat) Begin() {
	c.Open = true
	c.Input = ""
}

func (c *Chat) Cancel() {
	c.Open = false
	c.Input = ""
}

func (c *Chat) Update() (string, bool) {
	if !c.Open {
		return "", false
	}
	for {
		ch := rl.GetCharPressed()
		if ch == 0 {
			break
		}
		if ch >= 32 && ch < 127 && len(c.Input) < maxInput {
			c.Input += string(ch)
		}
	}
	if rl.IsKeyPressed(rl.KeyBackspace) && len(c.Input) > 0 {
		c.Input = c.Input[:len(c.Input)-1]
	}
	if rl.IsKeyPressed(rl.KeyEnter) || rl.IsKeyPressed(rl.KeyKpEnter) {
		text := c.Input
		c.Cancel()
		return text, true
	}
	if rl.IsKeyPressed(rl.KeyEscape) {
		c.Cancel()
	}
	return "", false
}

func (c *Chat) Draw(screenW, screenH int) {
	const pad = 10
	baseY := int32(screenH - pad - lineH)

	for i, m := range c.Lines {
		text := m.From + ": " + m.Text
		y := baseY - int32(len(c.Lines)-1-i)*lineH - lineH
		fonts.Draw(text, pad, y, 18, rl.RayWhite)
	}

	if c.Open {
		rect := rl.NewRectangle(pad-2, float32(baseY-2), float32(screenW-2*pad+4), lineH)
		rl.DrawRectangleRec(rect, rl.Fade(rl.Black, 0.6))
		rl.DrawRectangleLinesEx(rect, 1, rl.Gray)
		fonts.Draw("> "+c.Input+"_", pad, baseY, 18, rl.White)
	}
}
