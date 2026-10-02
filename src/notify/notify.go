// Package notify sends desktop notifications.
package notify

import (
	"os/exec"
	"strings"

	"github.com/mt-shihab26/orivo/src/logx"
)

// escape neutralises the markup the freedesktop notification spec allows in
// bodies (<b>, <i>, <a>, <img>), so todo text is shown literally instead of
// being interpreted by the notification daemon.
var escape = strings.NewReplacer("&", "&amp;", "<", "&lt;", ">", "&gt;")

// Send shows a desktop notification with the app name prepended to the summary.
func Send(summary, body string) {
	cmd := exec.Command("notify-send", "--app-name=orivo", "--icon=orivo",
		"--", "orivo — "+summary, escape.Replace(body))

	go func() {
		if err := cmd.Run(); err != nil {
			logx.Error("failed to send notification: %v", err)
		}
	}()
}
