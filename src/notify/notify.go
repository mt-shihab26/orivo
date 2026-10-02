package notify

import (
	"os/exec"
	"strings"

	"github.com/mt-shihab26/orivo/src/logx"
)

var escape = strings.NewReplacer("&", "&amp;", "<", "&lt;", ">", "&gt;")

func Send(summary, body string) {
	cmd := exec.Command("notify-send", "--app-name=orivo", "--icon=orivo",
		"--", "orivo — "+summary, escape.Replace(body))

	go func() {
		if err := cmd.Run(); err != nil {
			logx.Error("failed to send notification: %v", err)
		}
	}()
}
