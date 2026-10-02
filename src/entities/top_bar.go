package entities

import (
	"fmt"

	rl "github.com/gen2brain/raylib-go/raylib"

	"github.com/mt-shihab26/orivo/src/core"
)

type TopBar struct {
	input   *core.Input
	fonts   *core.Fonts
	quit    func()
	showFPS bool
}

func NewTopBar(input *core.Input, fonts *core.Fonts, showFPS bool, quit func()) *TopBar {
	return &TopBar{input: input, fonts: fonts, quit: quit, showFPS: showFPS}
}

func (t *TopBar) Close() {}

func (t *TopBar) Update(dt float32) {
	for _, key := range t.input.KeysFor(t) {
		switch {
		case key.Ctrl && (key.Code == rl.KeyQ || key.Code == rl.KeyC):
			t.quit()
		case key.Ctrl && key.Code == rl.KeyF:
			t.showFPS = !t.showFPS
		}
	}
}

func (t *TopBar) Draw() {
	screen := core.CurrentScreen()
	s := screen.Scale
	small := t.fonts.Small

	small.Draw("^q quit   ^f fps", 14*s, 12*s, core.ColorDim)
	if t.showFPS {
		fps := fmt.Sprintf("%d fps", rl.GetFPS())
		small.Draw(fps, screen.Width-14*s-small.Width(fps), 12*s, core.ColorDim)
	}
}
