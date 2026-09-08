<p>
  <img src="pkg/orivo.svg" width="120" alt="orivo logo"/>
</p>

# orivo

[![License](https://img.shields.io/crates/l/orivo)](https://github.com/mt-shihab26/orivo/blob/main/LICENSE)
[![Build](https://github.com/mt-shihab26/orivo/actions/workflows/build.yml/badge.svg)](https://github.com/mt-shihab26/orivo/actions/workflows/build.yml)
[![Tests](https://github.com/mt-shihab26/orivo/actions/workflows/test.yml/badge.svg)](https://github.com/mt-shihab26/orivo/actions/workflows/test.yml)
[![Crates.io](https://img.shields.io/crates/v/orivo)](https://crates.io/crates/orivo)
[![docs.rs](https://img.shields.io/docsrs/orivo)](https://docs.rs/orivo)

A terminal-based (TUI) Todos + Pomodoro timer written in [Rust](https://www.rust-lang.org)

## Installation

### [Omarchy](https://omarchy.org)

<!-- ```sh -->
<!-- omarchy pkg add orivo -->
<!-- ``` -->

```sh
git clone --depth 1 https://github.com/mt-shihab26/orivo.git /tmp/orivo
cd /tmp/orivo/pkg
makepkg -si
```

Installs the AUR package (see [`PKGBUILD`](pkg/PKGBUILD)), which places `orivo` on `/usr/bin`, registers a desktop entry that launches it via `omarchy-launch-terminal` — so it opens in whatever terminal you've configured as default — and ships an `orivo-sync.timer` user unit that's enabled automatically on install to run `orivo sync` hourly (see [Automatic sync](#automatic-sync)).

Usage (app launcher):

1. Press `SUPER + ALT + SPACE` to open the app launcher
2. Search for orivo

Usage (terminal):

```sh
$ orivo
```

Uninstall (also disables the `orivo-sync.timer` unit):

```sh
omarchy pkg drop orivo
```

### Cargo (any OS, builds from source)

**Requires sqlite3** — install it for your OS and ensure it's on your `PATH` before building.

```sh
$ cargo install orivo
```

Run from any terminal:

```sh
$ orivo
```

## Configuration

Config file location: `~/.config/orivo/config.toml`

```toml
# Orivo configuration

show_fps = false # show the FPS counter in the TUI header on startup

# Pomodoro timer settings — controls session lengths and when long breaks are triggered.
[timer]
show_millis         = false   # show milliseconds in the timer display
work_duration       = 25      # work session length in minutes         (min: 1, max: 120)
break_duration      = 5       # short break length in minutes          (min: 1, max: 60)
long_break_duration = 15      # long break length in minutes           (min: 1, max: 60)
long_break_interval = 4       # work sessions before a long break      (min: 1, max: 10)
daily_session_goal  = 16      # target work sessions to complete today (min: 1, max: 24)

# `orivo sync` settings — see the Sync section below.
[sync]
repo_name = "orivo-data" # name of the GitHub repo (under your account) synced with

```

### Root Options

- `show_fps` → show the FPS counter when the TUI starts. You can still toggle it at runtime with `Ctrl+F`.

### Timer (`[timer]`)

The [Pomodoro technique](https://en.wikipedia.org/wiki/Pomodoro_Technique) breaks work into focused sessions separated by breaks:

- **Work session** → focused work period (default: 25 min)
- **Short break** → rest between sessions (default: 5 min)
- **Long break** → rest after completing a full cycle (default: 15 min)

A full cycle = `work_duration` × `long_break_interval` work sessions. After that many sessions, a long break is triggered instead of a short one.

```
work → break → work → break → work → break → work → LONG BREAK  (cycle of 4)
```

**Daily session goal** (`daily_session_goal`) sets how many work sessions you aim to complete each day. Progress is shown as a session tracker in the timer tab. Once the goal is reached, the tracker fills completely.

- Default: `16` sessions
- Range: `1` – `24` sessions

## Sync

orivo can sync its local database with a private GitHub repo, WhatsApp-style: a single gzip-compressed snapshot, overwritten in place each time.

**Requires the [GitHub CLI](https://cli.github.com) (`gh`), signed in** — run `gh auth login` once before your first sync.

```sh
$ orivo sync
```

The first run creates a private repo under your GitHub account (named `orivo-data` by default — see `[sync] repo_name` in Configuration) via `gh repo create`, and uploads the database to it. Later runs compare the local database and the repo against the last synced snapshot: pulls if only the repo changed (e.g. you synced from another machine), pushes if only the local database changed, does nothing if neither did, and asks which side to keep if both did.

See [`docs/sync.md`](docs/sync.md) for the full mechanics (change detection, push/pull, conflict handling, error cases).

### Automatic sync

The Arch package installs a `orivo-sync.timer` user unit that runs `orivo sync` once an hour, and enables it automatically on install (no manual `systemctl` step needed) via a `systemctl --global enable` in the package's post-install hook. Disable it if you'd rather sync manually:

```sh
$ systemctl --user disable --now orivo-sync.timer
```

If both sides have changed since the last sync when the timer fires, `orivo sync` has no terminal to prompt on, so it just cancels that run instead of guessing — nothing is overwritten. Run `orivo sync` yourself to resolve it.

## Development

```sh
$ git clone https://github.com/mt-shihab26/orivo.git
$ cd orivo
```

```sh
$ cargo run
```

Debug builds keep config, database, and log files under `./.dev/` instead of the real system paths, so you can inspect or wipe local state freely.

```sh
$ cargo run seed    # reset the local database and fill it with sample todos/sessions
$ cargo test        # run the test suite
$ cargo fmt          # format code
$ cargo clippy       # lint
```
