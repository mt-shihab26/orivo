<p>
  <img src="pkg/orivo.svg" width="120" alt="orivo logo"/>
</p>

# orivo

[![Tests](https://github.com/mt-shihab26/orivo/actions/workflows/test.yml/badge.svg)](https://github.com/mt-shihab26/orivo/actions/workflows/test.yml)

A Pomodoro timer for the desktop, written in [Go](https://go.dev) with [raylib](https://www.raylib.com).

Todos are not managed in orivo: the timer can be pointed at a todo from [Todoist](https://todoist.com) that is overdue or due today, and sessions are recorded against it.

## Build

Requires Go and a C compiler (raylib is compiled from source through cgo), plus the usual OpenGL and X11/Wayland development headers.

```sh
$ git clone https://github.com/mt-shihab26/orivo.git
$ cd orivo
$ go build -o orivo .
$ ./orivo
```

## Keys

| Key       | Action                                             |
| --------- | -------------------------------------------------- |
| `Space`   | Start / pause                                      |
| `r`       | Reset the current phase                            |
| `n`       | Skip to the next phase                             |
| `t`       | Pick a todo (overdue and due today)                |
| `T`       | Clear the selected todo                            |
| `m`       | Toggle centiseconds on the clock                   |
| `d`       | Reduce the remaining time (enter `mm:ss`)          |
| `Ctrl+F`  | Toggle the FPS counter                             |
| `Ctrl+Q`  | Quit                                               |

In the todo picker, `j`/`k` or the arrow keys (or the mouse wheel) move, `Enter` or a click selects, and `Esc` cancels.

## Configuration

Config file location: `~/.config/orivo/config.toml`

```toml
# Orivo configuration

show_fps = false # show the FPS counter on startup
font     = ""    # path to a .ttf/.otf font; empty uses the system monospace font

# Pomodoro timer settings — controls session lengths and when long breaks are triggered.
[timer]
show_millis         = false   # show milliseconds in the timer display
work_duration       = 25      # work session length in minutes         (min: 1, max: 120)
break_duration      = 5       # short break length in minutes          (min: 1, max: 60)
long_break_duration = 15      # long break length in minutes           (min: 1, max: 60)
long_break_interval = 4       # work sessions before a long break      (min: 1, max: 10)
daily_session_goal  = 16      # target work sessions to complete today (min: 1, max: 24)
```

### Timer (`[timer]`)

The [Pomodoro technique](https://en.wikipedia.org/wiki/Pomodoro_Technique) breaks work into focused sessions separated by breaks:

- **Work session** → focused work period (default: 25 min)
- **Short break** → rest between sessions (default: 5 min)
- **Long break** → rest after completing a full cycle (default: 15 min)

After every `long_break_interval` work sessions, a long break is triggered instead of a short one.

```
work → break → work → break → work → break → work → LONG BREAK  (cycle of 4)
```

A break starts by itself when a work session ends; the next work session waits for you to start it.

**Daily session goal** (`daily_session_goal`) sets how many work sessions you aim to complete each day. Progress is shown at the top of the window.

## Commands

```
orivo                    Open the timer window
orivo connect-todoist    Save your Todoist API token
orivo sync-todoist       Fetch todos that are overdue or due today, and cache them
orivo version            Print the version
orivo help               Show the list of commands
```

## Todos

Connect once, then sync whenever you want the picker to catch up with Todoist:

```sh
$ orivo connect-todoist   # asks for the API token from Todoist: Settings > Integrations > Developer
$ orivo sync-todoist      # fetches, caches and prints the todos that are overdue or due today
```

`connect-todoist` checks the token with Todoist before saving it to `~/.local/state/orivo/todoist.token`. Setting `TODOIST_API_TOKEN` in the environment takes precedence over that file.

`sync-todoist` writes the cache to `~/.local/state/orivo/todoist.txt`, one todo per line as the due date, the Todoist id and the text, separated by tabs:

```
2026-10-02	6X7rM8997g3RQmvh	Write the quarterly report
```

The picker (`t`) reads that file and lists the todos in two sections, **Overdue** and **Today**. The window itself never talks to Todoist, so the picker shows whatever the last sync fetched.

## Files

Runtime state lives under `~/.local/state/orivo/`:

- `store.json` → the current phase, the selected todo, and the time left per todo
- `sessions.jsonl` → one line per completed session
- `todoist.txt` → the cached Todoist todos (see above)
- `todoist.token` → the Todoist API token
- `orivo.sock` → Unix socket that answers each connection with one JSON line describing the live timer (phase, running, remaining time, todo, sessions today), for bar widgets such as [omarchy-orivo-plugin](https://github.com/mt-shihab26/omarchy-orivo-plugin)
- `orivo.log` → warnings and errors

## Structure

The window is built like a game. All of it lives in `src/domains/root`: `app.go` runs the loop, and everything in it is an entity under `entities/` (the clock, the session bar, the todo picker, …). Every entity is created by its package's `New` function and implements the same three methods, defined in `src/domains/root/core`:

```go
type Entity interface {
	Close()
	Update(dt float32)
	Draw()
}
```

An entity is something that is drawn, and each one keeps its own logic and key handling in its `Update`, reading the keyboard straight from raylib. Entities form a tree: a parent creates, updates, draws and closes its children, and the folders mirror it. Each entity is its own package, nested under its parent.

```
src/domains/root/
├─ app.go
├─ core/                      the entity contract, fonts and screen helpers
└─ entities/
   ├─ topbar/
   ├─ hints/
   ├─ dialog/                    the base both dialogs embed
   └─ sessionbar/
      └─ clock/
         ├─ todolabel/
         │  └─ todopicker/       embeds dialog
         └─ reducedialog/        embeds dialog
```

`Clock` is the timer: it handles Space, `r`, `n` and `m`, counts down, rolls from one phase to the next, records sessions and saves. `Dialog` is the base both dialogs embed; it owns open and closed, Esc to dismiss, and the backdrop, panel, title and hint. Keys follow the tree: a parent reads its own keys only while none of its dialogs is open, and stops updating the other branch meanwhile, so nothing behind an open dialog reacts. `app.go` only opens the window, creates the top-level entities and runs the loop.

The packages under `src/systems` are helpers the entities call: the phase names, durations and colors, the session history, the store, the todo cache, the Todoist client, the status socket, notifications and signal handling.

Each command is one file in `src/commands` implementing the `Command` interface from `command.go`; `commands.go` routes the first argument to the command with that name:

```go
type Command interface {
	Name() string
	Summary() string
	Run(args []string) error
}
```

## Development

```sh
$ go run . --dev    # keep config and state under ./.dev instead of the system paths
$ go test ./...     # run the test suite
$ gofmt -l .        # check formatting
$ go vet ./...      # lint
```
