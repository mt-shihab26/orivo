package ui

import (
	"encoding/binary"
	"math"

	rl "github.com/gen2brain/raylib-go/raylib"
)

// newBeep synthesises the bright, short tone played on every phase transition.
func newBeep() rl.Sound {
	const (
		rate      = 44100
		frequency = 880.0
		amplitude = 0.15
		samples   = rate / 10 // 100ms
		// Ramp in and out, so the tone does not start or end with a click.
		fade = rate / 200
	)

	data := make([]byte, samples*2)
	for i := range samples {
		gain := amplitude * min(1, float64(i)/fade, float64(samples-1-i)/fade)
		value := gain * math.Sin(2*math.Pi*frequency*float64(i)/rate)
		binary.LittleEndian.PutUint16(data[i*2:], uint16(int16(value*math.MaxInt16)))
	}

	return rl.LoadSoundFromWave(rl.NewWave(samples, rate, 16, 1, data))
}
