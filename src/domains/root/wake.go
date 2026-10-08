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

import "sync/atomic"

// Set while the window is open; posting an event before or after that
// would reach a GLFW that is not running.
var windowOpen atomic.Bool

func wake() {
	if windowOpen.Load() {
		C.glfwPostEmptyEvent()
	}
}

// disableLibdecor keeps GLFW from loading libdecor, whose GTK plugin takes
// about a quarter second and 30 MiB to draw a title bar. It must run before
// the window opens.
func disableLibdecor() {
	C.disableLibdecor()
}
