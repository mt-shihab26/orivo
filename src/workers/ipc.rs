use std::{
    fs,
    io::Write,
    os::unix::{
        fs::{FileTypeExt, PermissionsExt},
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
        // Restrict the whole state directory to the owning user. This is
        // also what closes the socket's own creation race below: bind()
        // creates the socket file at the umask-derived mode before we get
        // a chance to chmod it, but a non-traversable parent means no
        // other local account can reach it during that window anyway. It
        // additionally locks down orivo.sqlite/store.json/orivo.log,
        // which were already world-readable before the IPC worker existed.
        let _ = fs::set_permissions(parent, fs::Permissions::from_mode(0o700));
    }
    // Never take the socket away from an instance that is still serving it:
    // binding over a live socket leaves the first instance with a listener
    // nobody can reach. Connecting is the only way to tell a live socket from
    // one a crashed process left behind — they look identical on disk.
    if UnixStream::connect(&path).is_ok() {
        log_warn!(
            "ipc worker: another orivo instance already serves {}; skipping IPC",
            path.display()
        );
        return;
    }

    // Nothing is listening, so clear the stale entry — but only when it really
    // is a socket, rather than deleting an unrelated file that sits there.
    match fs::symlink_metadata(&path) {
        Ok(meta) if meta.file_type().is_socket() => {
            let _ = fs::remove_file(&path);
        }
        Ok(_) => {
            log_error!(
                "ipc worker: {} exists and is not a socket; refusing to replace it",
                path.display()
            );
            return;
        }
        Err(_) => {}
    }

    let listener = match UnixListener::bind(&path) {
        Ok(listener) => listener,
        Err(e) => {
            log_error!("ipc worker: failed to bind {}: {e}", path.display());
            return;
        }
    };

    // Belt and suspenders: also restrict the socket file itself.
    if let Err(e) = fs::set_permissions(&path, fs::Permissions::from_mode(0o600)) {
        log_warn!(
            "ipc worker: failed to restrict permissions on {}: {e}",
            path.display()
        );
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
            "todo_text": state.todo_text(),
            "sessions_today": state.sessions_count(),
            "daily_session_goal": state.daily_session_goal(),
        })
        .to_string()
    };

    if let Err(e) = writeln!(stream, "{payload}") {
        log_warn!("ipc worker: write failed: {e}");
    }
}
