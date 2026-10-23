package notify

import (
	"encoding/binary"
	"math"
	"os/exec"
	"strings"
	"sync/atomic"
	"time"

	rl "github.com/gen2brain/raylib-go/raylib"

	"orivo/src/systems/logx"
)

const soundName = "message-new-instant"

var escape = strings.NewReplacer("&", "&amp;", "<", "&lt;", ">", "&gt;")

// Set while a tone plays, so tones never overlap.
var playing atomic.Bool

// Send shows a desktop notification and plays the tone, both in the
// background. Opening the audio device takes tens of milliseconds, longer
// when the sound server is slow, so the window never waits for it; the
// tone's goroutine is the only one that touches audio.
func Send(summary, body string) {
	go func() {
		if err := Show(summary, body); err != nil {
			logx.Error("failed to send notification: %v", err)
		}
	}()

	if playing.CompareAndSwap(false, true) {
		go func() {
			defer playing.Store(false)
			playTone()
		}()
	}
}

// playTone opens the audio device only for the tone, so no mixing thread
// stays busy between phases.
func playTone() {
	rl.InitAudioDevice()
	if !rl.IsAudioDeviceReady() {
		logx.Warn("failed to open the audio device for the tone")
		return
	}
	defer rl.CloseAudioDevice()

	tone := newTone()
	defer rl.UnloadSound(tone)
	rl.PlaySound(tone)
	for rl.IsSoundPlaying(tone) {
		time.Sleep(10 * time.Millisecond)
	}
	// Let the device play out what it has buffered before it closes.
	time.Sleep(100 * time.Millisecond)
}

// Show sends a desktop notification and waits for notify-send to finish.
func Show(summary, body string) error {
	return exec.Command("notify-send", "--app-name=orivo", "--icon=orivo",
		"--hint=string:sound-name:"+soundName,
		"--", "orivo — "+summary, escape.Replace(body)).Run()
}

func newTone() rl.Sound {
	const (
		rate      = 44100
		frequency = 880.0
		amplitude = 0.15
		frames    = rate / 10
		fade      = rate / 200
	)

	data := make([]byte, frames*2)
	for i := range frames {
		gain := amplitude * min(1, float64(i)/fade, float64(frames-1-i)/fade)
		value := gain * math.Sin(2*math.Pi*frequency*float64(i)/rate)
		binary.LittleEndian.PutUint16(data[i*2:], uint16(int16(value*math.MaxInt16)))
	}

	return rl.LoadSoundFromWave(rl.NewWave(frames, rate, 16, 1, data))
}
