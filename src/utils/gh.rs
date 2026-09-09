use std::{
    fs,
    io::{Error, ErrorKind, Result},
    path::Path,
    process::Command,
};

/// Returns whether the `gh` CLI is installed and signed in to a GitHub account.
pub fn is_authenticated() -> bool {
    Command::new("gh")
        .args(["auth", "status"])
        .output()
        .map(|output| output.status.success())
        .unwrap_or(false)
}

/// Returns the signed-in GitHub username.
fn username() -> Result<String> {
    run_gh(&["api", "user", "-q", ".login"])
}

/// Returns whether `owner/orivo-data` already exists on GitHub.
fn repo_exists(full_name: &str) -> bool {
    Command::new("gh")
        .args(["repo", "view", full_name])
        .output()
        .map(|output| output.status.success())
        .unwrap_or(false)
}

/// Returns `owner/<repo_name>` for the signed-in user, creating the repo as private if it
/// doesn't exist yet.
pub fn ensure_repo(repo_name: &str) -> Result<String> {
    println!("checking for github repo {repo_name}...");
    let full_name = format!("{}/{repo_name}", username()?);

    if repo_exists(&full_name) {
        println!("found existing repo {full_name}");
    } else {
        println!("creating private repo {full_name}...");
        run_gh(&[
            "repo",
            "create",
            &full_name,
            "--private",
            "--description",
            "orivo sync data \u{2014} managed automatically by `orivo sync`, do not edit",
        ])?;
    }

    Ok(full_name)
}

/// Ensures `dir` is a local clone of `repo_full_name` (`owner/name`): clones it via `gh` if
/// missing, otherwise fetches the latest remote state into it.
pub fn ensure_clone(dir: &Path, repo_full_name: &str) -> Result<()> {
    if dir.join(".git").exists() {
        println!("fetching latest changes from {repo_full_name}...");
        run_git(dir, &["fetch", "origin"])?;
        return Ok(());
    }

    println!("cloning {repo_full_name}...");
    if let Some(parent) = dir.parent() {
        fs::create_dir_all(parent)?;
    }

    let output = Command::new("gh")
        .args(["repo", "clone", repo_full_name, &dir.to_string_lossy()])
        .output()
        .map_err(|_| io_err("`gh` (github CLI) is not installed; see https://cli.github.com"))?;

    if !output.status.success() {
        return Err(io_err(
            String::from_utf8_lossy(&output.stderr).trim().to_string(),
        ));
    }

    Ok(())
}

/// Returns the clone's current branch name (works even for a freshly created, commit-less
/// repo, since `HEAD` still symbolically points at the default branch).
pub fn current_branch(dir: &Path) -> Result<String> {
    run_git(dir, &["symbolic-ref", "--short", "HEAD"])
}

/// Reads the sync file's bytes as committed on `origin/<branch>`, or `None` if it doesn't
/// exist yet (e.g. nothing has been pushed to the repo before).
pub fn read_remote_file(dir: &Path, branch: &str, file_name: &str) -> Option<Vec<u8>> {
    Command::new("git")
        .arg("-C")
        .arg(dir)
        .args(["show", &format!("origin/{branch}:{file_name}")])
        .output()
        .ok()
        .filter(|output| output.status.success())
        .map(|output| output.stdout)
}

/// Writes `bytes` as the sync file and pushes it to `branch`, adding a new commit each time
/// so the repo keeps full sync history.
pub fn commit_and_push(
    dir: &Path,
    branch: &str,
    bytes: &[u8],
    message: &str,
    file_name: &str,
) -> Result<()> {
    fs::write(dir.join(file_name), bytes)?;
    run_git(dir, &["add", file_name])?;
    run_git(dir, &["commit", "-m", message])?;

    println!("pushing to github...");
    run_git(dir, &["push", "-u", "origin", &format!("HEAD:{branch}")])?;

    Ok(())
}

/// Runs `gh` with the given args, returning trimmed stdout on success.
fn run_gh(args: &[&str]) -> Result<String> {
    let output = Command::new("gh")
        .args(args)
        .output()
        .map_err(|_| io_err("`gh` (github CLI) is not installed; see https://cli.github.com"))?;

    if !output.status.success() {
        return Err(io_err(
            String::from_utf8_lossy(&output.stderr).trim().to_string(),
        ));
    }

    Ok(String::from_utf8_lossy(&output.stdout).trim().to_string())
}

/// Runs `git -C dir <args>`, returning trimmed stdout on success.
fn run_git(dir: &Path, args: &[&str]) -> Result<String> {
    let output = Command::new("git")
        .arg("-C")
        .arg(dir)
        .args(args)
        .output()
        .map_err(|_| io_err("`git` is not installed"))?;

    if !output.status.success() {
        return Err(io_err(
            String::from_utf8_lossy(&output.stderr).trim().to_string(),
        ));
    }

    Ok(String::from_utf8_lossy(&output.stdout).trim().to_string())
}

fn io_err(e: impl std::fmt::Display) -> Error {
    Error::new(ErrorKind::Other, e.to_string())
}
