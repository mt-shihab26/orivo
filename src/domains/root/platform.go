package root

// GLFW is compiled with both Wayland and X11 and picks one when the window
// opens; these choose it and report the choice.

/*
void glfwInitHint(int hint, int value);
int glfwGetPlatform(void);

// From glfw3.h.
#define GLFW_PLATFORM         0x00050003
#define GLFW_PLATFORM_WAYLAND 0x00060003
#define GLFW_PLATFORM_X11     0x00060004

static void choosePlatform(int platform) {
	glfwInitHint(GLFW_PLATFORM, platform);
}
*/
import "C"

import (
	"fmt"
	"os"
	"path/filepath"
	"runtime"
	"strings"
)

// pinPlatform has the window talk to the display server natively: Wayland
// when a Wayland display is there to connect to, X11 when only an X display
// is. GLFW would follow XDG_SESSION_TYPE first, so a stale one could put a
// Wayland session through XWayland. With neither, GLFW looks for itself. It
// returns why, for the log.
func pinPlatform() (why string) {
	wayland, waylandWhy := waylandDisplay()
	switch {
	case wayland:
		C.choosePlatform(C.GLFW_PLATFORM_WAYLAND)
		return waylandWhy
	case os.Getenv("DISPLAY") != "":
		C.choosePlatform(C.GLFW_PLATFORM_X11)
		return fmt.Sprintf("%s; DISPLAY is %s", waylandWhy, os.Getenv("DISPLAY"))
	}
	return waylandWhy + " and DISPLAY is not set, so GLFW chose"
}

// waylandDisplay reports whether the socket WAYLAND_DISPLAY names exists,
// and says so.
func waylandDisplay() (bool, string) {
	name := os.Getenv("WAYLAND_DISPLAY")
	if name == "" {
		return false, "WAYLAND_DISPLAY is not set"
	}
	path := name
	if !filepath.IsAbs(path) {
		path = filepath.Join(os.Getenv("XDG_RUNTIME_DIR"), path)
	}
	if _, err := os.Stat(path); err != nil {
		return false, fmt.Sprintf("WAYLAND_DISPLAY is %s but %s is not there", name, path)
	}
	return true, fmt.Sprintf("WAYLAND_DISPLAY is %s and %s is there", name, path)
}

// osName is the system's own name for itself from /etc/os-release, with the
// Go OS and architecture.
func osName() string {
	target := runtime.GOOS + "/" + runtime.GOARCH
	raw, err := os.ReadFile("/etc/os-release")
	if err != nil {
		return target
	}
	for _, line := range strings.Split(string(raw), "\n") {
		if value, ok := strings.CutPrefix(line, "PRETTY_NAME="); ok {
			return strings.Trim(value, `"'`) + " (" + target + ")"
		}
	}
	return target
}

// platformName is the display server the open window talks to.
func platformName() string {
	switch C.glfwGetPlatform() {
	case C.GLFW_PLATFORM_WAYLAND:
		return "Wayland"
	case C.GLFW_PLATFORM_X11:
		return "X11"
	}
	return "unknown"
}
