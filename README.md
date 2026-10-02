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

## Todos

The picker (`t`) lists pending Todoist tasks in two sections, **Overdue** and **Today**, read from a cached task list at `~/.local/state/orivo/todoist.json`. Fetching that cache from Todoist is not implemented yet; until then the file can be written by hand or by a script. It holds Todoist tasks as Todoist's own API returns them — a JSON array, or an object with the tasks under `items` or `results`:

```json
[
  { "id": "6X7rM8997g3RQmvh", "content": "Write the quarterly report", "due": { "date": "2026-10-02" } }
]
```

## Files

Runtime state lives under `~/.local/state/orivo/`:

- `store.json` → the current phase, the selected todo, and the time left per todo
- `sessions.jsonl` → one line per completed session
- `todoist.json` → the cached Todoist tasks (see above)
- `orivo.sock` → Unix socket that answers each connection with one JSON line describing the live timer (phase, running, remaining time, todo, sessions today), for bar widgets such as [omarchy-orivo-plugin](https://github.com/mt-shihab26/omarchy-orivo-plugin)
- `orivo.log` → warnings and errors

## Structure

The window is built like a game: `src/app` runs the loop, and everything in it is an entity from `src/entities` (the clock, the session bar, the todo picker, …). Every entity is created by its `New` function and implements the same three methods, defined in `src/core`:

```go
type Entity interface {
	Close()
	Update(dt float32)
	Draw()
}
```

Entities share a `core.World`, which holds the timer, the todo source and the state of the current frame. The packages under `src/systems` are what runs behind it: the timer state machine, the store, the session log and the Todoist cache.

## Development

```sh
$ go run . --dev    # keep config and state under ./.dev instead of the system paths
$ go test ./...     # run the test suite
$ gofmt -l .        # check formatting
$ go vet ./...      # lint
```
