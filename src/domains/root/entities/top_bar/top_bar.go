package top_bar

import (
	"fmt"

	rl "github.com/gen2brain/raylib-go/raylib"

	"orivo/src/domains/root/core"
)

type TopBar struct {
	fonts   *core.Fonts
	quit    func()
	showFPS bool
}

func New(fonts *core.Fonts, showFPS bool, quit func()) *TopBar {
	return &TopBar{fonts: fonts, quit: quit, showFPS: showFPS}
}

func (t *TopBar) Close() {}

func (t *TopBar) Update(dt float32) {
	if !rl.IsKeyDown(rl.KeyLeftControl) && !rl.IsKeyDown(rl.KeyRightControl) {
		return
	}
	if rl.IsKeyPressed(rl.KeyQ) || rl.IsKeyPressed(rl.KeyC) {
		t.quit()
	}
	if rl.IsKeyPressed(rl.KeyF) {
		t.showFPS = !t.showFPS
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
