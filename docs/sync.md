# How `orivo sync` works

`orivo sync` keeps the local SQLite database (todos + pomodoro sessions) in sync with a
private GitHub repo: a gzip-compressed snapshot, committed anew each sync so the repo keeps
full history. It shells out to the [GitHub CLI](https://cli.github.com) (`gh`) and `git`
rather than talking to any API directly.

This doc describes the current implementation. Source: `src/domains/sync.rs` (orchestration),
`src/utils/gh.rs` (`gh`/`git` process wrappers), `src/config/sync.rs` (config), `src/cmds/sync.rs`
(CLI entry point).

## Prerequisites

- **`gh`, signed in.** `orivo sync` never handles GitHub authentication itself — it checks
  `gh auth status` and, if that fails, tells you to run `gh auth login` and exits. There's no
  OAuth client, no secrets, no keyring entry anywhere in orivo's own code for this.
- **`git`**, for the local clone/commit/push mechanics.

## What gets synced

Only the database file (`db_path()` — `orivo.sqlite`). Nothing else: `config.toml` and
`store.json` are per-machine preferences, not synced.

## Configuration

```toml
[sync]
repo_name = "orivo-data"      # name of the GitHub repo (under your account) synced with
file_name = "orivo.sqlite.gz" # name of the synced database blob inside that repo
```

`repo_name` (`src/config/sync.rs`) defaults to `"orivo-data"` in release builds and
`"orivo-data-dev"` in debug builds, so a local `cargo run sync` during development can never
touch the same repo a real install would use. It's always created/looked up under the
signed-in `gh` user's own account — there's no way to point it at someone else's repo.

`file_name` defaults to `"orivo.sqlite.gz"` and is the same in release and debug builds.

## Where things live on disk

| What                     | Path (release)                      | Path (debug)          |
| ------------------------ | ----------------------------------- | --------------------- |
| Database                 | `~/.local/state/orivo/orivo.sqlite` | `./.dev/orivo.sqlite` |
| Sync state (`sync.json`) | `~/.local/state/orivo/sync.json`    | `./.dev/sync.json`    |
| Local git clone          | `~/.local/state/orivo/sync/`        | `./.dev/sync/`        |

The local git clone is reused across runs (`git fetch` instead of a fresh `gh repo clone`
every time), so a sync after the first one only downloads new commits, not the whole
repo/database again.

## The GitHub repo itself

- Named `owner/<repo_name>` — created via `gh repo create <name> --private` the first time
  `orivo sync` runs and the repo doesn't already exist (checked with `gh repo view`).
- Always **private**.
- Holds exactly one file: `orivo.sqlite.gz` by default (configurable via `[sync] file_name`,
  the database, gzip-compressed).
- **Full commit history.** Every push is a new commit (`git commit` + `git push`), so the repo
  builds up a real history of every synced snapshot over time rather than staying at a single
  overwritten commit.

## The sync algorithm

Everything happens in `run_sync()` (`src/domains/sync.rs`):

1. **Check `gh` sign-in.** `gh::is_authenticated()` runs `gh auth status`. If it fails, sync
   stops immediately with an error telling you to run `gh auth login`.
2. **Ensure the repo exists.** `gh::ensure_repo(repo_name)` resolves the signed-in username
   (`gh api user -q .login`), checks whether `owner/<repo_name>` exists (`gh repo view`), and
   creates it private if not.
3. **Ensure the local clone is current.** `gh::ensure_clone()`: if `<sync_dir>/.git` already
   exists, `git fetch origin`; otherwise `gh repo clone owner/<repo_name> <sync_dir>` (this
   works fine even against a brand-new, completely empty repo — it just produces a clone with
   no commits yet).
4. **Compare.** This is the core of the "pull if needed, push if needed" behavior — see
   [Change detection](#change-detection) below.
5. **Act**: push, pull, prompt for a conflict, or do nothing, depending on step 4's result.

## Change detection

Nothing is diffed file-by-file. Three SHA-256 hashes of the whole gzipped database blob decide
everything:

1. **`local_hash`** — read the live `orivo.sqlite`, gzip it in memory, hash the gzipped bytes.
2. **`remote_hash`** — `git show origin/<branch>:<file_name>` inside the local clone (reads
   the committed blob straight out of git's object store — no checkout needed), hash the
   result. `None` if the repo has never had anything pushed to it.
3. **`last_synced_hash`** — loaded from `sync.json`: the hash recorded after the _last
   successful_ sync. This is orivo's own memory of "what I already synced," independent of
   git.

```rust
let local_changed  = last_synced_hash != Some(local_hash);
let remote_changed = remote_hash      != last_synced_hash;
```

Local and remote are never compared to each other directly — both are compared against the
last-known-synced hash. That's what lets orivo tell _which side_ changed instead of just _that_
they differ.

| Repo state                                                      | `local_changed` | `remote_changed` | Action                                |
| --------------------------------------------------------------- | :-------------: | :--------------: | ------------------------------------- |
| Repo has nothing pushed yet                                     |        —        |        —         | **push**, unconditionally (see below) |
| Neither side changed                                            |       no        |        no        | nothing — "already up to date"        |
| Only the local database changed                                 |       yes       |        no        | **push**                              |
| Only the GitHub repo changed (e.g. synced from another machine) |       no        |       yes        | **pull**                              |
| Both changed                                                    |       yes       |       yes        | **prompt**: keep local or keep remote |

### The empty-repo special case

If `remote_gz` comes back `None` (nothing has ever been pushed), `run_sync` pushes
immediately and skips the hash-comparison table above entirely. This matters because a stale
`sync.json` — e.g. left over from a sync that failed after recording a hash for a repo that
then got deleted or recreated — would otherwise make an empty repo look like a "both changed"
conflict for no real reason. An empty repo has no data to protect, so there's nothing to ask
about.

## Push

`push()` in `src/domains/sync.rs` calls `gh::commit_and_push()`:

1. Write the gzipped bytes to `<sync_dir>/<file_name>`.
2. `git add <file_name>`.
3. `git commit` (always a new commit, keeping full history).
4. `git push -u origin HEAD:<branch>` (a plain fast-forward push — no force needed, since each
   push only ever adds a commit on top of what was last fetched).
5. On success, record `local_hash` as the new `last_synced_hash` in `sync.json`.

## Pull

`pull()` in `src/domains/sync.rs`:

1. Gunzip the remote blob (read during the comparison step, via `git show`).
2. Write it to a temp file next to the database and `fs::rename` it over `db_path()`
   (atomic — never leaves a partially-written database on failure).
3. Record `remote_hash` as the new `last_synced_hash` in `sync.json`.

## Conflict resolution

When both sides changed since the last sync, `resolve_conflict()` prints:

```
both the local database and the github repo have changed since the last sync.
keep [l]ocal (push, overwriting the repo) or [r]emote (pull, overwriting local)?
```

and reads a line from stdin: `l`/`local` pushes, `r`/`remote` pulls, anything else (including
just pressing enter) cancels the sync with no changes made on either side. There's no
automatic merge — the database is a single opaque binary blob, so "merging" isn't meaningful;
one side has to be chosen to fully overwrite the other.

When `orivo sync` runs with no terminal attached (e.g. from the systemd timer below),
`io::stdin().read_line()` just reads EOF, `answer.trim()` comes back empty, and that falls
into the same "anything else" branch — the sync is cancelled, nothing gets overwritten. A
conflict is never resolved automatically in either direction; it always needs an interactive
`orivo sync` run.

## Running automatically: the systemd timer

The Arch package (`pkg/PKGBUILD`) installs two **user** systemd units, vendored directly
alongside `PKGBUILD` (not fetched from a GitHub release, since they're packaging metadata
rather than build output):

- `pkg/orivo-sync.service` → `/usr/lib/systemd/user/orivo-sync.service` — a oneshot unit that
  runs `orivo sync`.
- `pkg/orivo-sync.timer` → `/usr/lib/systemd/user/orivo-sync.timer` — fires it `OnCalendar=hourly`,
  with `RandomizedDelaySec=5m` (spreads out load rather than every install syncing at exactly
  `:00`) and `Persistent=true` (a run missed while the machine was off/asleep fires shortly
  after boot instead of waiting for the next hour).

These are **user** units (`/usr/lib/systemd/user/`, not `/usr/lib/systemd/system/`) because
`orivo sync` depends on per-user state: the `gh` CLI's own credentials (from `gh auth login`)
and orivo's database live under the invoking user's home directory, not anywhere a system
service could reach without extra configuration.

### Enabled automatically on install

`pkg/orivo.install`'s `post_install`/`post_upgrade` hooks run:

```sh
systemctl --global enable orivo-sync.timer
```

`--global` is what makes this work from a pacman hook at all: pacman hooks run as root during
installation, with no single logged-in user to target, and there's no guarantee any user has
an active systemd `--user` session at that point for a plain `systemctl --user enable` to
reach. `--global` instead symlinks the unit under `/etc/systemd/user/timers.target.wants/`,
which is a pure filesystem change — it enables the timer for every user on the system as soon
as their session starts, without needing one running right now. `pre_remove` undoes it with
`systemctl --global disable orivo-sync.timer` on package removal.

`--global` only manages that symlink, though — it has no runtime component and can't start the
timer, so on its own the timer would sit enabled-but-inactive until the next login. `post_install`
also best-effort starts it immediately for whoever is actually running the install:
`makepkg -si` and AUR helpers invoke pacman via `sudo`, so `$SUDO_USER` identifies that user
(falling back to `logname` for a direct root install); if their systemd `--user` session is
already up (`/run/user/<uid>/bus` exists), `sudo -u "$user" systemctl --user start` starts the
timer in it. Any other already-logged-in users still just get it on their next login, same as
before.

To opt out, disable it per-user:

```sh
$ systemctl --user disable --now orivo-sync.timer
```

## Terminal output

Every step prints as it happens (rather than staying silent until the end), so a slow network
call doesn't look like a hang. The shared setup steps (sign-in check, repo lookup, clone/fetch,
"comparing...") are the same every time — what differs is the line printed right after the
compare step, which announces what the hash comparison found before acting on it:

**Only the local database changed → push**

```
checking github CLI sign-in...
checking for github repo orivo-data...
found existing repo <owner>/orivo-data
fetching latest changes from <owner>/orivo-data...
comparing local database with github...
local database changed, github did not — pushing
pushing to github...
pushed local changes to github
```

**Only the GitHub repo changed → pull**

```
checking github CLI sign-in...
checking for github repo orivo-data...
found existing repo <owner>/orivo-data
fetching latest changes from <owner>/orivo-data...
comparing local database with github...
database on github changed, local did not — pulling
restoring database from github...
pulled latest data from github
```

**Neither side changed → no-op**

```
checking github CLI sign-in...
checking for github repo orivo-data...
found existing repo <owner>/orivo-data
fetching latest changes from <owner>/orivo-data...
comparing local database with github...
nothing changed since the last sync, already up to date
```

**Both sides changed → conflict prompt**

```
checking github CLI sign-in...
checking for github repo orivo-data...
found existing repo <owner>/orivo-data
fetching latest changes from <owner>/orivo-data...
comparing local database with github...
both the local database and the github repo have changed since the last sync.
keep [l]ocal (push, overwriting the repo) or [r]emote (pull, overwriting local)?
```

**Repo has nothing pushed to it yet → push, skipping the compare table entirely**

```
checking github CLI sign-in...
checking for github repo orivo-data...
creating private repo <owner>/orivo-data...
cloning <owner>/orivo-data...
comparing local database with github...
repo on github is empty — pushing
pushing to github...
pushed local changes to github
```

All messages are lowercase, including "github" itself, matching the rest of the CLI's output
style.

## Error cases

- **`gh` not signed in** → "not signed in to the github CLI; run `gh auth login` first", exits
  before touching the repo or the network.
- **`gh` or `git` not installed** → the process spawn fails and surfaces a message naming
  which binary is missing and, for `gh`, a link to install it.
- **Any `gh`/`git` command exits non-zero** (repo creation fails, clone fails, push fails,
  etc.) → its stderr is surfaced as the error message; `orivo sync` stops rather than trying to
  recover automatically.
