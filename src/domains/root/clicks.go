package root

// raylib sees a click only as a change in the button's state from one poll
// of events to the next. A touchpad tap sends the press and the release
// together, so both land in one poll and raylib, and raygui with it, never
// sees the click. This sits between GLFW and raylib's button callback and
// holds such a release back until the poll after, so the frame between them
// sees the button pressed and the next one sees it released.
//
// It also scales the mouse to the size the window is drawn at.

/*
#include <stddef.h>

typedef struct GLFWwindow GLFWwindow;
typedef void (*buttonFun)(GLFWwindow*, int, int, int);

GLFWwindow* glfwGetCurrentContext(void);
buttonFun glfwSetMouseButtonCallback(GLFWwindow* window, buttonFun callback);
void glfwGetWindowSize(GLFWwindow* window, int* width, int* height);

#define BUTTONS 8
#define RELEASE 0
#define PRESS   1

static buttonFun raylibButton;
static GLFWwindow* clickWindow;
// Pressed since the last poll, so a release now would hide the press.
static int pressedThisPoll[BUTTONS];
// Releases held back: 1 in the poll they came in, 2 once a poll has passed.
static int held[BUTTONS];
static int heldMods[BUTTONS];

static void onButton(GLFWwindow* window, int button, int action, int mods) {
	if (button < 0 || button >= BUTTONS) {
		raylibButton(window, button, action, mods);
		return;
	}
	if (action == RELEASE && pressedThisPoll[button]) {
		held[button] = 1;
		heldMods[button] = mods;
		return;
	}
	if (action == PRESS) {
		pressedThisPoll[button] = 1;
	}
	raylibButton(window, button, action, mods);
}

static void watchClicks(void) {
	clickWindow = glfwGetCurrentContext();
	raylibButton = glfwSetMouseButtonCallback(clickWindow, onButton);
	if (raylibButton == NULL) {
		// Nothing to sit in front of; give raylib nothing different.
		glfwSetMouseButtonCallback(clickWindow, NULL);
	}
}

static void windowSize(int* width, int* height) {
	glfwGetWindowSize(clickWindow, width, height);
}

// afterPoll runs once after every poll of events. It hands raylib the
// releases held back a poll ago, and reports whether any is still held, so
// the next poll must not wait for input.
static int afterPoll(void) {
	int waiting = 0;
	for (int b = 0; b < BUTTONS; b++) {
		pressedThisPoll[b] = 0;
		if (held[b] == 2) {
			held[b] = 0;
			raylibButton(clickWindow, b, RELEASE, heldMods[b]);
		} else if (held[b] == 1) {
			held[b] = 2;
			waiting = 1;
		}
	}
	return waiting;
}
*/
import "C"

import (
	"fmt"
	"os"

	rl "github.com/gen2brain/raylib-go/raylib"

	"orivo/src/systems/logx"
)

// watchClicks puts the release holding in front of raylib's callback. It
// must run after the window opens.
func watchClicks() {
	C.watchClicks()
}

// afterPoll runs once per loop, after the events are polled. A release held
// back wakes the loop, so the next frame comes without waiting for input.
func afterPoll() {
	if C.afterPoll() != 0 {
		wake()
	}
}

// debugInput logs each click as raylib sees it, with ORIVO_DEBUG_INPUT=1.
var debugInput = os.Getenv("ORIVO_DEBUG_INPUT") == "1"

func logClicks() {
	if !debugInput {
		return
	}
	pos := rl.GetMousePosition()
	if rl.IsMouseButtonPressed(rl.MouseButtonLeft) {
		logx.Info("input: left button pressed at %.0f,%.0f", pos.X, pos.Y)
	}
	if rl.IsMouseButtonReleased(rl.MouseButtonLeft) {
		logx.Info("input: left button released at %.0f,%.0f", pos.X, pos.Y)
	}
}

// The scale last logged, to two decimals, so a resize does not repeat it.
var loggedScale string

// scaleMouse matches the mouse to what is drawn. Without its high-DPI flag,
// raylib draws a Wayland window at the scaled size, 1.25 times the logical
// one at 125%, but reads the mouse in logical units, so every click would
// land short of where it was aimed. The ratio is 1 on X11 and at 100%. It
// runs every frame, as a resize or another monitor changes it.
func scaleMouse() {
	var width, height C.int
	C.windowSize(&width, &height)
	if width <= 0 || height <= 0 {
		return
	}
	x, y := float32(rl.GetScreenWidth())/float32(width), float32(rl.GetScreenHeight())/float32(height)
	if scale := fmt.Sprintf("%.2f", x); scale != loggedScale {
		logx.Info("window: drawn at %dx%d for %dx%d, so the mouse is scaled by %s", rl.GetScreenWidth(), rl.GetScreenHeight(), width, height, scale)
		loggedScale = scale
	}
	rl.SetMouseScale(x, y)
}
