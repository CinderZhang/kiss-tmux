# KISS-TMUX Design Spec

**Date:** 2026-04-08
**Status:** Draft
**Author:** Cinder Zhang + Claude

## Overview

KISS-TMUX (Keep It Simple and Stupid TMUX) is a standalone, web-based terminal multiplexer built in Go for Windows. It lets you run multiple Claude Code sessions (or any command) in a browser grid, monitor them simultaneously, and switch to full-screen focus for interactive work.

No connection to Kanabanana. No kanban. No agents. No database. Just terminals.

## Decisions

| Decision | Choice | Rationale |
|----------|--------|-----------|
| Language | Go | Single binary, ConPTY support, goroutines for session management, fast builds |
| PTY | ConPTY via `UserExistsError/conpty` | Native Windows 10+ pseudo-console, simpler than Unix PTY |
| Platform | Windows only (v1) | User's platform, ConPTY-only keeps code simple |
| Interaction | Hybrid: `kiss-tmux serve` + browser UI | CLI launches, browser controls. CLI subcommands can be added later |
| Default command | `claude` | CC session manager by default, custom command as escape hatch |
| Frontend | Vanilla HTML/JS + xterm.js | No build step, embedded in binary via `go:embed` |
| Layout | Switchable grid/focus modes | Grid for monitoring, focus for working |
| Session count | 4-8 | Grid adapts, max 8 default |
| WebSocket | Single multiplexed connection | One WS, messages routed by session ID |
| Auth | None | Localhost only, bound to 127.0.0.1 |
| Persistence | None | Sessions die when server stops. Ring buffer for reconnect replay only |

## Project Structure

```
kiss-tmux/
├── main.go              # entry point, flag parsing, serve command
├── server.go            # HTTP server, WebSocket handler, static file embed
├── session.go           # session manager, ConPTY lifecycle, ring buffer
├── protocol.go          # WS message type definitions (JSON)
├── web/                 # embedded static files
│   ├── index.html       # single page app
│   ├── app.js           # grid/focus logic, xterm instances, WS client
│   ├── style.css        # layout styles (dark theme)
│   └── vendor/          # vendored xterm.js + xterm-addon-fit
├── go.mod
└── go.sum
```

6 Go files. 1 HTML. 1 JS. 1 CSS. Two Go dependencies.

## WebSocket Protocol

All messages are JSON over a single WebSocket at `ws://localhost:{port}/ws`.

### Client → Server

| Type | Fields | Description |
|------|--------|-------------|
| `spawn` | `session`, `cmd`, `cwd`, `name` | Create a new ConPTY session |
| `input` | `session`, `data` | Send keystrokes to session stdin |
| `resize` | `session`, `cols`, `rows` | Resize session PTY |
| `kill` | `session` | Kill session process |
| `rename` | `session`, `name` | Rename session |
| `list` | — | Request current session list |

### Server → Client

| Type | Fields | Description |
|------|--------|-------------|
| `output` | `session`, `data` | PTY stdout data |
| `exited` | `session`, `code` | Process exited |
| `error` | `session`, `error` | Error message |
| `sessions` | `sessions[]` | Full session list (broadcast on any state change) |

Session IDs are short random strings generated server-side on spawn.

`sessions` message is broadcast on every state change (spawn, kill, exit, rename) so the UI stays in sync without polling.

## Session Manager

```go
type Session struct {
    ID      string
    Name    string
    Cmd     string
    Cwd     string
    Running bool
    ConPTY  *conpty.ConPty
    Buffer  *RingBuffer    // last 64KB for reconnect replay
}

type Manager struct {
    sessions sync.Map       // id → *Session
    broadcast func([]byte)  // push JSON to all WS clients
    maxSessions int         // default 8
}
```

### Lifecycle

1. **Spawn**: Create ConPTY with command + cwd + dimensions. Start read goroutine. Broadcast `sessions`.
2. **Read loop**: ConPTY stdout → append to 64KB ring buffer → broadcast `output` message to all WS clients.
3. **Input**: Write data to ConPTY stdin. For large inputs (>256 chars), chunk with 10ms delays to prevent buffer overflow.
4. **Resize**: Resize ConPTY dimensions.
5. **Kill**: Close ConPTY. Broadcast `exited` + `sessions`.
6. **Natural exit**: Same as kill — broadcast `exited` + `sessions`.
7. **Reconnect**: Send `sessions` list. For each session, replay ring buffer contents so client isn't staring at blank terminals.

### No persistence

No crash recovery, no session files, no database. When `kiss-tmux` stops, all sessions stop. Start fresh. The ring buffer exists only for browser tab reconnection during a running server.

## Web UI

### Two modes

**Grid Mode** (default):
- Auto-arranging grid (CSS grid, adapts to session count)
- Each cell shows: session name, running indicator (green/grey dot), last ~10 lines of output
- Click any cell → switch to focus mode on that session
- Empty cell with `+ New Session` button
- Keyboard: `Ctrl+G` toggle, `Ctrl+N` new, `Ctrl+1-8` jump to session

**Focus Mode**:
- Thin sidebar (~48px) showing 2-letter session abbreviations with color coding
- Full-size xterm.js terminal filling remaining space
- Active session highlighted in sidebar
- Keyboard: `Esc` back to grid, `Ctrl+Up/Down` previous/next session

### New Session Dialog

Modal with three fields:
- **Name** (optional): display name, defaults to command name
- **Working Directory**: defaults to current directory
- **Command**: defaults to `claude`

### Interactions

| Action | Effect |
|--------|--------|
| Click terminal in grid | Focus mode on that session |
| Esc / Ctrl+G | Toggle grid/focus |
| Ctrl+N | New session dialog |
| Ctrl+1 through Ctrl+8 | Jump to session by index |
| Ctrl+Up/Down | Previous/next session in focus mode |
| Double-click session name | Rename inline |
| Right-click terminal header | Context menu: rename, kill |

### Theming

Dark theme only. Terminal colors match xterm.js defaults. Minimal chrome — the terminals are the UI.

## Build & Distribution

```bash
# Build
cd kiss-tmux
go build -o kiss-tmux.exe .

# Run
./kiss-tmux serve              # default port 7777
./kiss-tmux serve --port 8080  # custom port
```

The binary embeds all static files via `go:embed`. xterm.js and xterm-addon-fit are vendored as release `.js` files in `web/vendor/` — no npm, no node_modules, no build step.

### Go Dependencies

- `github.com/UserExistsError/conpty` — Windows ConPTY wrapper
- `github.com/gorilla/websocket` — WebSocket server

That's it. Two dependencies.

### CLI

```
kiss-tmux serve [flags]

Flags:
  --port int    Port to listen on (default 7777)
  --open        Auto-open browser (default true)
```

No config file. No data directory. The binary IS the app.

## Non-Goals

- Cross-platform support (v1 is Windows only)
- Authentication / multi-user
- Session persistence / crash recovery
- Agent-specific parsing or output analysis
- Task management / kanban integration
- Database or file-based state
- Voice input / dictation
- React / build toolchain
