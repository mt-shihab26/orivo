package entities

import (
	"fmt"

	rl "github.com/gen2brain/raylib-go/raylib"

	"github.com/mt-shihab26/orivo/src/core"
)

type TopBar struct {
	world   *core.World
	showFPS bool
}

func NewTopBar(world *core.World) *TopBar {
	return &TopBar{world: world, showFPS: world.Config.ShowFPS}
}

func (t *TopBar) Close() {}

func (t *TopBar) Update(dt float32) {
	for _, key := range t.world.Keys {
		switch {
		case key.Ctrl && (key.Code == rl.KeyQ || key.Code == rl.KeyC):
			t.world.Quit()
		case key.Ctrl && key.Code == rl.KeyF:
			t.showFPS = !t.showFPS
		}
	}
}

func (t *TopBar) Draw() {
	w := t.world
	small := w.Fonts.Small

	small.Draw("^q quit   ^f fps", 14*w.Scale, 12*w.Scale, core.ColorDim)
	if t.showFPS {
		fps := fmt.Sprintf("%d fps", rl.GetFPS())
		small.Draw(fps, w.Width-14*w.Scale-small.Width(fps), 12*w.Scale, core.ColorDim)
	}
}
