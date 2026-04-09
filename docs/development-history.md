# Development History

KISS-TMUX was built end-to-end with [Claude Code](https://claude.com/claude-code) following a disciplined DRIVER workflow. This document captures how the project came together, what worked, what broke, and the lessons learned along the way.

## Origin

The need was simple: run multiple Claude Code sessions side-by-side on Windows without juggling terminal windows. Existing tmux ports and clones either don't support Windows, require WSL, or carry too much baggage for what should be a tiny tool.

The name is the philosophy: **K**eep **I**t **S**imple and **S**tupid. Six Go files, two dependencies, one HTML, one JS, one CSS. No npm, no build toolchain, no database.

## Design Spec

The design was authored before any code was written. See [`docs/design.md`](./design.md) for the full spec.

**Key decisions from the spec:**

| Decision | Choice | Rationale |
|----------|--------|-----------|
| Language | Go | Single binary, ConPTY support, goroutines, fast builds |
| PTY | ConPTY (`UserExistsError/conpty`) | Native Windows 10+ pseudo-console |
| Platform | Windows only | User's platform, keeps code simple |
| Frontend | Vanilla HTML/JS + xterm.js | No build step, embedded via `go:embed` |
| WebSocket | Single multiplexed connection | One WS, messages routed by session ID |
| Auth | None | Localhost only, bound to 127.0.0.1 |
| Persistence | None | Ring buffer for reconnect replay only |

## Implementation Plan

A detailed implementation plan ([`docs/superpowers/plans/2026-04-08-kiss-tmux.md`](./superpowers/plans/2026-04-08-kiss-tmux.md)) broke the work into 9 bite-sized tasks with full code blocks, test cases, and commit messages.

The plan used the `superpowers:writing-plans` skill to ensure every step was concrete — no placeholders like "add error handling" or "implement later."

## Execution via Subagent-Driven Development

Tasks were executed using `superpowers:subagent-driven-development` — a pattern where:

1. A fresh subagent is dispatched per task with the full task text and context
2. The subagent implements, tests, commits, and self-reviews
3. A spec compliance reviewer verifies the code matches the task
4. A code quality reviewer checks for issues
5. Only after both reviews pass does the next task start

This keeps the controller's context clean (no file-read accumulation) while ensuring each task gets independent verification.

### Task flow summary

| Task | What | Outcome |
|------|------|---------|
| 1 | Go module + protocol types | TDD, 6 JSON roundtrip tests ✅ |
| 2 | Ring buffer (64KB circular) | TDD, 8 tests including 100KB overflow ✅ |
| 3 | Session manager (ConPTY lifecycle) | 4 integration tests spawning real `cmd.exe` ✅ |
| 4 | WebSocket hub + HTTP server | Hub pattern from gorilla/websocket chat example ✅ |
| 5 | Main entry point + CLI | 8.9MB binary on first build ✅ |
| 6 | Vendor xterm.js + HTML/CSS | Downloaded from jsDelivr, no npm ✅ |
| 7 | JavaScript (WS client, grid/focus, dialogs) | 435 lines of vanilla JS ✅ |
| 8 | CSS grid column tuning | Data-count attribute for responsive layout ✅ |
| 9 | Build + smoke test | 18/18 Go tests pass, HTTP serves correctly ✅ |

All 9 tasks were completed in sequence with passing tests and clean builds.

## Bugs Found and Fixed

The first smoke test exposed real bugs that TDD couldn't have caught.

### Bug 1: Terminal content blanks on grid/focus switch

**Symptom:** After switching between grid and focus modes a few times, terminal cells were empty. Content came back only after refreshing the browser tab.

**Root cause:** The JavaScript `attachTerminal()` function called `term.open(container)` on every mode switch. xterm.js `open()` can only be called **once** per Terminal instance — subsequent calls corrupt internal state.

**Fix:** Call `open()` once at terminal creation into a persistent wrapper div. On mode switches, move the wrapper to the new container with `appendChild` — the terminal's DOM subtree is preserved, just re-parented.

### Bug 2: Terminal blanks again on session switch in focus mode

Same symptom on a different code path. The guard `if (!t || t.el === container) return` in `attachTerminal` short-circuited when switching back to a session that was previously focused in the same container — but `focusDiv.innerHTML = ''` had already detached the wrapper, so `t.el` was stale.

**Fix:** Remove the stale guard. Always re-append. `appendChild` is idempotent when the child is already in the parent.

### Bug 3: WebSocket deadlock risk and closed-channel panics

A code review of the backend found two critical concurrency issues in `server.go`:

1. `sendInitialState` was called **before** `writePump` started. If broadcasts arrived during that window and filled the send channel, the hub would close the channel and kill the connection before writePump ran.
2. `readPump` wrote directly to `c.send` without any guard. If the hub had already closed the channel, the send would panic.

**Fix:** Start `writePump` before hub registration. Add a `safeSend()` method with `recover()` and a `select` with default — gracefully drops messages if the channel is closed.

### Bug 4: Text rendering felt jank compared to modern terminals

**Symptom:** Wrapping and scrolling weren't as smooth as VSCode's terminal.

**Root cause:** xterm.js ships with a default DOM/Canvas renderer. Modern apps like VSCode use `xterm-addon-webgl` for GPU-accelerated rendering.

**Fix:** Vendored `xterm-addon-webgl.js` and loaded it into every terminal after `open()`. Also handles GPU context loss gracefully.

## UX Iterations

Even with the plan, UX details needed iteration based on real use:

1. **Keyboard shortcuts conflicted with terminal apps.** The original plan had Ctrl+G (toggle mode), Ctrl+N (new session), Ctrl+1-8 (jump to session), Ctrl+Up/Down (prev/next). These collided with shell bindings. Replaced with sidebar buttons (grid toggle at top, + at bottom). Only Escape remains — and only for closing the dialog.

2. **Sidebar abbreviations too short.** Started at 2 chars, widened to 4, then to 6 chars based on feedback. Added CSS hover tooltips showing the full session name as an instant flyout.

3. **Copy/paste didn't work out of the box.** xterm.js doesn't wire up the clipboard automatically. Added `paste`, `copy`, and `keydown` listeners that route clipboard operations to the active terminal via `term.paste()` and `term.getSelection()`.

## What Worked

- **DRIVER workflow** forced research before coding. No "just start building" — every stage had a gate.
- **Subagent-driven execution** kept the controller's context clean and gave each task independent verification.
- **TDD on backend data layers** (protocol, ring buffer, session manager) caught regressions instantly. 18 tests run in under 2 seconds.
- **Vanilla frontend** meant zero build step. Every change is just `go build` and refresh.
- **`go:embed`** produces one file to distribute. No "where do I put the web assets" problem.

## What Didn't Work

- **TDD on the frontend** wasn't worth the setup cost for this scale. Tests would have duplicated the structure without catching real bugs.
- **Keyboard shortcuts by default** — terminal apps steal them. Buttons are more discoverable and don't conflict.
- **Trusting `term.open()` to be re-callable** — the xterm.js docs say it's one-shot but the error doesn't surface obviously. The bug only appears after several mode switches.

## Tech Stack Lessons

| Lesson | Where it came from |
|--------|---------------------|
| xterm.js `open()` is one-shot — call it into a stable wrapper and reparent via `appendChild` | Bug 1 & 2 |
| gorilla/websocket connections are not safe for concurrent writes — use a writePump + channel | Spec research |
| For quant/CLI/tool work, a Go + single-binary + vanilla-JS stack beats any npm-based web framework | DRIVER tech-stack guidance |
| WebGL renderer is the smoothness gap between hobby and production xterm.js setups | Bug 4 |
| Canvas text selection in xterm.js is separate from DOM selection — clipboard needs explicit wiring | Copy/paste bug |

## Timeline

All of this happened over two days (2026-04-08 and 2026-04-09):

- **Day 1** — Design spec authored, implementation plan written, all 9 tasks executed, first smoke test, two terminal-blanking bugs found and fixed, v0.1.0 released.
- **Day 2** — Copy/paste support, README, WebGL renderer, v0.2.0 and v0.3.0 released.

## References

- Design spec: [`docs/design.md`](./design.md)
- Implementation plan: [`docs/superpowers/plans/2026-04-08-kiss-tmux.md`](./superpowers/plans/2026-04-08-kiss-tmux.md)
- Changelog: [`../CHANGELOG.md`](../CHANGELOG.md)
- Releases: https://github.com/CinderZhang/kiss-tmux/releases
