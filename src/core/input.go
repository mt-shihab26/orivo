package core

import (
	"slices"

	rl "github.com/gen2brain/raylib-go/raylib"
)

type Key struct {
	Ch   rune
	Code int32
	Ctrl bool
}

type Input struct {
	Keys []Key
}

var controlKeys = []int32{
	rl.KeyEnter, rl.KeyKpEnter, rl.KeyEscape, rl.KeyBackspace, rl.KeyUp, rl.KeyDown,
}

func ReadKeys() []Key {
	var keys []Key
	ctrl := rl.IsKeyDown(rl.KeyLeftControl) || rl.IsKeyDown(rl.KeyRightControl)

	for code := rl.GetKeyPressed(); code != 0; code = rl.GetKeyPressed() {
		if ctrl || slices.Contains(controlKeys, code) {
			keys = append(keys, Key{Code: code, Ctrl: ctrl})
		}
	}
	for _, code := range controlKeys {
		if !ctrl && rl.IsKeyPressedRepeat(code) {
			keys = append(keys, Key{Code: code})
		}
	}
	for ch := rl.GetCharPressed(); ch != 0; ch = rl.GetCharPressed() {
		if !ctrl {
			keys = append(keys, Key{Ch: ch})
		}
	}
	return keys
}
