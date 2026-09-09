use std::{
    fs,
    io::Write,
    os::unix::{
        fs::PermissionsExt,
        net::{UnixListener, UnixStream},
    },
    sync::{Arc, Mutex},
    thread,
};

use serde_json::json;

use crate::{log_error, log_warn, states::timer::TimerState, utils::path::ipc_socket_path};

/// Spawns the IPC worker thread, serving the current timer status over a
/// Unix domain socket so external tools (e.g. an Omarchy bar widget) can
/// query orivo's live state without shelling out or reading `store.json` —
/// `is_running` in particular is only ever known in memory, never persisted.
///
/// Protocol: a client connects, orivo writes one JSON line describing the
/// current status, then closes the connection. No request body is read.
pub fn spawn(state: Arc<Mutex<TimerState>>) {
    let path = ipc_socket_path();

    if let Some(parent) = path.parent() {
        let _ = fs::create_dir_all(parent);
    }
    // Remove a stale socket left behind by a previous unclean shutdown.
    let _ = fs::remove_file(&path);

    let listener = match UnixListener::bind(&path) {
        Ok(listener) => listener,
        Err(e) => {
            log_error!("ipc worker: failed to bind {}: {e}", path.display());
            return;
        }
    };

    // Restrict the socket to the owning user — it's created with the
    // process umask otherwise, which typically leaves it world-readable.
    if let Err(e) = fs::set_permissions(&path, fs::Permissions::from_mode(0o600)) {
        log_warn!("ipc worker: failed to restrict permissions on {}: {e}", path.display());
    }

    thread::spawn(move || {
        for stream in listener.incoming() {
            match stream {
                Ok(stream) => handle_connection(stream, &state),
                Err(e) => log_warn!("ipc worker: accept failed: {e}"),
            }
        }
    });
}

/// Writes one JSON status line describing the current timer state to `stream`.
fn handle_connection(mut stream: UnixStream, state: &Arc<Mutex<TimerState>>) {
    let payload = {
        let state = match state.lock() {
            Ok(guard) => guard,
            Err(poisoned) => {
                log_warn!("ipc worker: timer state mutex poisoned, recovering");
                poisoned.into_inner()
            }
        };

        let phase = state.cycle_phase();

        json!({
            "phase": phase.to_db_str(),
            "label": phase.label(),
            "is_running": state.is_running(),
            "remaining_millis": state.current_millis(),
            "todo_id": state.todo_id(),
            "sessions_today": state.sessions_count(),
            "daily_session_goal": state.daily_session_goal(),
        })
        .to_string()
    };

    if let Err(e) = writeln!(stream, "{payload}") {
        log_warn!("ipc worker: write failed: {e}");
    }
}
