/// IPC worker that serves live timer state over a Unix domain socket.
pub mod ipc;
/// Terminal event worker that forwards key and resize events.
pub mod term;
/// Timer worker that advances pomodoro state and emits tick events.
pub mod timer;
