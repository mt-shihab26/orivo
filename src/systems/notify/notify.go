package notify

import (
	"encoding/binary"
	"math"
	"os/exec"
	"strings"

	rl "github.com/gen2brain/raylib-go/raylib"

	"github.com/mt-shihab26/orivo/src/systems/logx"
)

const soundName = "message-new-instant"

var escape = strings.NewReplacer("&", "&amp;", "<", "&lt;", ">", "&gt;")

var (
	tone       rl.Sound
	toneLoaded bool
)

func LoadSound() {
	tone = newTone()
	toneLoaded = true
}

func UnloadSound() {
	if toneLoaded {
		rl.UnloadSound(tone)
		toneLoaded = false
	}
}

func Send(summary, body string) {
	show(summary, body)
	if toneLoaded {
		rl.PlaySound(tone)
	}
}

func show(summary, body string) {
	cmd := exec.Command("notify-send", "--app-name=orivo", "--icon=orivo",
		"--hint=string:sound-name:"+soundName,
		"--", "orivo — "+summary, escape.Replace(body))

	go func() {
		if err := cmd.Run(); err != nil {
			logx.Error("failed to send notification: %v", err)
		}
	}()
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
