package notify

import (
	"encoding/binary"
	"math"
	"os/exec"
	"strings"
	"time"

	rl "github.com/gen2brain/raylib-go/raylib"

	"orivo/src/systems/logx"
)

const soundName = "message-new-instant"

// How long the audio device stays open after the tone starts. The tone lasts
// a tenth of a second; an open device keeps a mixing thread busy.
const audioOpenFor = 2 * time.Second

var escape = strings.NewReplacer("&", "&amp;", "<", "&lt;", ">", "&gt;")

var (
	tone     rl.Sound
	openedAt time.Time
)

// Send shows a desktop notification and plays the tone. The audio device is
// opened only for the tone, which costs a few milliseconds each time instead
// of a busy thread all along. It must run on the render loop's goroutine.
func Send(summary, body string) {
	go func() {
		if err := Show(summary, body); err != nil {
			logx.Error("failed to send notification: %v", err)
		}
	}()

	if !rl.IsAudioDeviceReady() {
		rl.InitAudioDevice()
		if !rl.IsAudioDeviceReady() {
			return
		}
		tone = newTone()
	}
	openedAt = time.Now()
	rl.PlaySound(tone)
}

// Release closes the audio device once the tone is done. Call it every
// update, from the render loop's goroutine.
func Release() {
	if openedAt.IsZero() || time.Since(openedAt) < audioOpenFor || rl.IsSoundPlaying(tone) {
		return
	}
	Close()
}

func Close() {
	if openedAt.IsZero() {
		return
	}
	rl.UnloadSound(tone)
	rl.CloseAudioDevice()
	openedAt = time.Time{}
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
