use std::{
    fs,
    io::{self, Error, ErrorKind, Read, Result, Write},
    path::Path,
};

use flate2::{Compression, read::GzDecoder, write::GzEncoder};
use serde::{Deserialize, Serialize};
use sha2::{Digest, Sha256};

use crate::{
    config::Config,
    utils::{
        date::{format_datetime, now},
        gh,
        path::{db_path, sync_dir, sync_state_path},
    },
};

/// Persisted sync state: the content hash of the database as of the last successful sync,
/// so `orivo sync` can tell whether the local database, the GitHub repo, or both have
/// changed since.
#[derive(Debug, Default, Deserialize, Serialize)]
struct SyncState {
    last_synced_hash: Option<String>,
}

impl SyncState {
    /// Loads the sync state from disk, returning the default if the file does not exist.
    fn load() -> Self {
        let path = sync_state_path();
        fs::read_to_string(path)
            .ok()
            .and_then(|s| serde_json::from_str(&s).ok())
            .unwrap_or_default()
    }

    /// Saves the sync state to disk.
    fn save(&self) {
        let path = sync_state_path();
        if let Some(parent) = path.parent() {
            let _ = fs::create_dir_all(parent);
        }
        if let Ok(json) = serde_json::to_string(self) {
            let _ = fs::write(path, json);
        }
    }
}

/// Ensures a private `orivo-data` GitHub repo exists for the signed-in `gh` user, then syncs
/// the local database with it: pulls if the repo has changes this machine doesn't have yet,
/// pushes if this machine has changes the repo doesn't have, does nothing if neither changed,
/// or asks which side to keep if both did.
pub fn run_sync() -> Result<()> {
    println!("checking github CLI sign-in...");
    if !gh::is_authenticated() {
        return Err(io_err(
            "not signed in to the github CLI; run `gh auth login` first",
        ));
    }

    let config = Config::load()?;
    let repo_name = gh::ensure_repo(config.sync.repo_name())?;
    let dir = sync_dir();
    gh::ensure_clone(&dir, &repo_name)?;
    let branch = gh::current_branch(&dir)?;

    println!("comparing local database with github...");
    let local_gz = gzip(&fs::read(db_path())?)?;
    let local_hash = hash(&local_gz);

    let remote_gz = gh::read_remote_file(&dir, &branch);

    // Nothing has ever been pushed to this repo — there's no remote data to lose, so push
    // unconditionally instead of falling through to the conflict check below (a stale local
    // `SyncState` from an earlier failed sync would otherwise look like a real conflict here).
    if remote_gz.is_none() {
        println!("repo on github is empty — pushing");
        return push(&dir, &branch, local_gz, local_hash);
    }

    let remote_hash = remote_gz.as_deref().map(hash);

    let state = SyncState::load();
    let last_hash = state.last_synced_hash.as_deref();

    let local_changed = last_hash != Some(local_hash.as_str());
    let remote_changed = remote_hash.as_deref() != last_hash;

    match (local_changed, remote_changed) {
        (false, false) => {
            println!("nothing changed since the last sync, already up to date");
            Ok(())
        }
        (true, false) => {
            println!("local database changed, github did not — pushing");
            push(&dir, &branch, local_gz, local_hash)
        }
        (false, true) => {
            println!("database on github changed, local did not — pulling");
            pull(remote_gz, remote_hash)
        }
        (true, true) => {
            resolve_conflict(&dir, &branch, local_gz, local_hash, remote_gz, remote_hash)
        }
    }
}

/// Prompts the user to pick a side when both the local database and the repo changed since
/// the last sync, since a binary sqlite file can't be merged automatically.
fn resolve_conflict(
    dir: &Path,
    branch: &str,
    local_gz: Vec<u8>,
    local_hash: String,
    remote_gz: Option<Vec<u8>>,
    remote_hash: Option<String>,
) -> Result<()> {
    println!("both the local database and the github repo have changed since the last sync.");
    print!("keep [l]ocal (push, overwriting the repo) or [r]emote (pull, overwriting local)? ");
    io::stdout().flush()?;

    let mut answer = String::new();
    io::stdin().read_line(&mut answer)?;

    match answer.trim().to_lowercase().as_str() {
        "l" | "local" => push(dir, branch, local_gz, local_hash),
        "r" | "remote" => pull(remote_gz, remote_hash),
        _ => {
            println!("sync cancelled");
            Ok(())
        }
    }
}

/// Commits and force-pushes the local database to the sync repo, overwriting its previous
/// snapshot in place, then records the new hash as synced.
fn push(dir: &Path, branch: &str, gz: Vec<u8>, hash: String) -> Result<()> {
    let message = format!("sync: {}", format_datetime(now()));
    gh::commit_and_push(dir, branch, &gz, &message)?;

    let mut state = SyncState::load();
    state.last_synced_hash = Some(hash);
    state.save();

    println!("pushed local changes to github");
    Ok(())
}

/// Overwrites the local database with the sync repo's copy, then records its hash as synced.
fn pull(gz: Option<Vec<u8>>, hash: Option<String>) -> Result<()> {
    println!("restoring database from github...");
    let gz = gz.ok_or_else(|| io_err("the sync repo's database file is missing"))?;
    let raw = gunzip(&gz)?;

    let path = db_path();
    let tmp_path = path.with_extension("sqlite.tmp");
    fs::write(&tmp_path, raw)?;
    fs::rename(&tmp_path, &path)?;

    let mut state = SyncState::load();
    state.last_synced_hash = hash;
    state.save();

    println!("pulled latest data from github");
    Ok(())
}

/// Compresses `data` with gzip.
fn gzip(data: &[u8]) -> Result<Vec<u8>> {
    let mut encoder = GzEncoder::new(Vec::new(), Compression::default());
    encoder.write_all(data)?;
    encoder.finish()
}

/// Decompresses gzip-compressed `data`.
fn gunzip(data: &[u8]) -> Result<Vec<u8>> {
    let mut out = Vec::new();
    GzDecoder::new(data).read_to_end(&mut out)?;
    Ok(out)
}

/// Returns the lowercase hex-encoded SHA-256 hash of `data`.
fn hash(data: &[u8]) -> String {
    let mut hasher = Sha256::new();
    hasher.update(data);
    hasher
        .finalize()
        .iter()
        .map(|b| format!("{b:02x}"))
        .collect()
}

fn io_err(e: impl std::fmt::Display) -> Error {
    Error::new(ErrorKind::Other, e.to_string())
}
