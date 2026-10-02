package entities

import (
	"encoding/binary"
	"math"

	rl "github.com/gen2brain/raylib-go/raylib"

	"github.com/mt-shihab26/orivo/src/core"
	"github.com/mt-shihab26/orivo/src/notify"
)

const saveEvery = 60

type Pomodoro struct {
	world     *core.World
	beep      rl.Sound
	sinceSave float32
}

func NewPomodoro(world *core.World) *Pomodoro {
	p := &Pomodoro{world: world, beep: newBeep()}

	world.Timer.OnPhaseEnd(func(summary, body string) {
		notify.Send(summary, body)
		rl.PlaySound(p.beep)
	})
	return p
}

func (p *Pomodoro) Close() {
	p.world.Timer.OnPhaseEnd(nil)
	rl.UnloadSound(p.beep)
	p.world.Timer.Save()
}

func (p *Pomodoro) Update(dt float32) {
	timer := p.world.Timer

	for _, key := range p.world.Keys {
		switch key.Ch {
		case ' ':
			timer.Toggle()
		case 'r':
			timer.Reset()
		case 'n':
			timer.Skip()
		case 'm':
			timer.ToggleMillis()
		}
	}

	timer.Tick()

	p.sinceSave += dt
	if p.sinceSave >= saveEvery {
		p.sinceSave = 0
		timer.Save()
	}
}

func (p *Pomodoro) Draw() {}

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
