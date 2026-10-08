package root

// raylib-go compiles GLFW in but does not expose these, so they are declared
// here. glfwPostEmptyEvent ends the wait of glfwWaitEvents from any thread.

/*
void glfwInitHint(int hint, int value);
void glfwPostEmptyEvent(void);

// From glfw3.h.
#define GLFW_WAYLAND_LIBDECOR         0x00053001
#define GLFW_WAYLAND_DISABLE_LIBDECOR 0x00038002

static void disableLibdecor(void) {
	glfwInitHint(GLFW_WAYLAND_LIBDECOR, GLFW_WAYLAND_DISABLE_LIBDECOR);
}
*/
import "C"

import "sync"

// Posting an event before the window opens or after it closes would reach a
// GLFW that is not running. The lock makes closing wait for a post already
// under way, so none slips in between the check and the call.
var (
	glfw       sync.Mutex
	windowOpen bool
)

func wake() {
	glfw.Lock()
	defer glfw.Unlock()
	if windowOpen {
		C.glfwPostEmptyEvent()
	}
}

func setWindowOpen(open bool) {
	glfw.Lock()
	defer glfw.Unlock()
	windowOpen = open
}

// disableLibdecor keeps GLFW from loading libdecor, whose GTK plugin takes
// about a quarter second and 30 MiB to draw a title bar. It must run before
// the window opens.
func disableLibdecor() {
	C.disableLibdecor()
}
