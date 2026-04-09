# KISS-TMUX

**Keep It Simple and Stupid TMUX** — a web-based terminal multiplexer in Go for Windows. Run multiple Claude Code sessions (or any command) in a browser grid, monitor them simultaneously, and switch to full-screen focus for interactive work.

No install. No config. No database. Just a single `.exe` that serves a terminal grid at `http://127.0.0.1:7777`.

## Why

When working with multiple long-running agents (like several Claude Code sessions), you want to *see* them all at once without juggling terminal windows. KISS-TMUX gives you:

- **Grid mode** — watch all sessions at a glance
- **Focus mode** — full-size terminal with a sidebar to jump between sessions
- **One binary, no dependencies** — download and run

## Quick Start

1. Download `kiss-tmux-windows-amd64.exe` from the [latest release](https://github.com/CinderZhang/kiss-tmux/releases/latest)
2. Run it:
   ```
   kiss-tmux-windows-amd64.exe
   ```
3. Your browser opens to `http://127.0.0.1:7777`
4. Click `+ New Session` to spawn a terminal

### CLI flags

```
kiss-tmux [flags]

  --port int    Port to listen on (default 7777)
  --open        Auto-open browser (default true)
```

## Features

- **Grid mode** — auto-arranging CSS grid, up to 8 concurrent sessions
- **Focus mode** — thin sidebar with session abbreviations + full-size terminal
- **Sidebar buttons** — grid toggle (☰), new session (+), one tab per session
- **Hover tooltips** — full session names on hover
- **Context menu** — right-click a session header to rename or kill
- **Inline rename** — double-click a session name in the grid
- **Live reconnect** — refresh the browser and sessions replay from a 64KB ring buffer
- **Multiple browser tabs** — all tabs stay in sync via WebSocket broadcast
- **Default command** — `claude --dangerously-skip-permissions` (customizable per-session)

## Requirements

- **Windows 10 build 1809 or later** (for ConPTY support)
- A command to run in each session (defaults to Claude Code)

Only Windows is supported in v1. Cross-platform is a non-goal — ConPTY keeps the code simple.

## Architecture

Six files. Two Go dependencies. One HTML, one CSS, one JS. No npm, no build toolchain.

```
kiss-tmux/
├── main.go              # CLI entry, flag parsing, wiring
├── server.go            # Hub, Client, WebSocket handler, HTTP embed
├── session.go           # RingBuffer, Session, Manager (ConPTY lifecycle)
├── protocol.go          # JSON message types
├── web/
│   ├── index.html       # Grid view, focus view, dialog, context menu
│   ├── app.js           # WebSocket client, xterm.js terminal management
│   ├── style.css        # Dark theme
│   └── vendor/          # xterm.js + fit addon (vendored, no npm)
├── go.mod
└── go.sum
```

### Data flow

```
Browser (xterm.js)
  ↕ WebSocket JSON
Server (Hub → Client read/write pumps)
  ↕ Hub.broadcast channel
Manager (Session goroutines)
  ↕ io.Reader/io.Writer
ConPTY (Windows pseudo-console)
  ↕ kernel32.dll CreatePseudoConsole
Child Process (claude, cmd.exe, ...)
```

### Dependencies

- [`github.com/UserExistsError/conpty`](https://github.com/UserExistsError/conpty) — Windows ConPTY wrapper
- [`github.com/gorilla/websocket`](https://github.com/gorilla/websocket) — WebSocket server

That's it. Two Go dependencies. Frontend uses vanilla JS with [xterm.js](https://xtermjs.org/) vendored from jsDelivr.

## Build from source

```bash
git clone https://github.com/CinderZhang/kiss-tmux
cd kiss-tmux
go build -o kiss-tmux.exe .
```

For a stripped release build:

```bash
go build -ldflags="-s -w" -o kiss-tmux.exe .
```

All web assets are embedded via `go:embed` — the binary is fully self-contained.

## Testing

```bash
go test -v
```

18 tests covering the protocol (JSON roundtrip), ring buffer (circular overflow), and session manager (ConPTY lifecycle with real `cmd.exe` spawns).

## Non-goals

- Cross-platform support (Windows only)
- Authentication / multi-user (localhost only, bound to 127.0.0.1)
- Session persistence across server restarts
- Task management, kanban, or agent-specific parsing
- React, build toolchains, or npm

## License

MIT

## Acknowledgements

Built with [Claude Code](https://claude.com/claude-code). See `docs/design.md` and `docs/superpowers/plans/` for the full design spec and implementation plan.
