package entities

import (
	"encoding/binary"
	"math"

	rl "github.com/gen2brain/raylib-go/raylib"

	"github.com/mt-shihab26/orivo/src/core"
	"github.com/mt-shihab26/orivo/src/systems/notify"
)

type Bell struct {
	pomodoro *Pomodoro
	beep     rl.Sound
}

func NewBell(world *core.World, pomodoro *Pomodoro) *Bell {
	b := &Bell{pomodoro: pomodoro, beep: newBeep()}

	pomodoro.OnPhaseEnd = func(summary, body string) {
		notify.Send(summary, body)
		rl.PlaySound(b.beep)
	}
	return b
}

func (b *Bell) Close() {
	b.pomodoro.OnPhaseEnd = nil
	rl.UnloadSound(b.beep)
}

func (b *Bell) Update(dt float32) {}

func (b *Bell) Draw() {}

func newBeep() rl.Sound {
	const (
		rate      = 44100
		frequency = 880.0
		amplitude = 0.15
		samples   = rate / 10
		fade      = rate / 200
	)

	data := make([]byte, samples*2)
	for i := range samples {
		gain := amplitude * min(1, float64(i)/fade, float64(samples-1-i)/fade)
		value := gain * math.Sin(2*math.Pi*frequency*float64(i)/rate)
		binary.LittleEndian.PutUint16(data[i*2:], uint16(int16(value*math.MaxInt16)))
	}

	return rl.LoadSoundFromWave(rl.NewWave(samples, rate, 16, 1, data))
}
