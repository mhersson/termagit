# CLAUDE.md

This file provides guidance to Claude Code (claude.ai/code) when working with
code in this repository.

## What is termagit

A standalone terminal Git UI — a port of
[Neogit](https://github.com/NeogitOrg/neogit) built with Bubble Tea and go-git.
Editor-agnostic, runs from any terminal.

## Build & Development Commands

```bash
make build              # Build binary to bin/termagit
make test               # Run short tests with race detector
make test-integration   # Run all tests including integration
make lint               # Run golangci-lint
make run                # Build and run
make install            # Copy to $GOPATH/bin/
```

Run a single test:

```bash
go test -run TestName ./internal/ui/status/
```

Requires Go 1.26+.

## Architecture

### Bubble Tea Model-Update-View

The app uses the [Bubble Tea](https://github.com/charmbracelet/bubbletea) TUI
framework (Elm architecture). Every component implements `tea.Model` with
`Init()`, `Update(tea.Msg)`, and `View()` methods. All state changes flow
through messages — never mutate state directly in handlers; return a `tea.Cmd`
that produces a message.

### Screen State Machine

`internal/app/app.go` — The root `Model` holds all sub-models and routes events
based on the `active Screen` enum:

```
ScreenStatus | ScreenLog | ScreenReflog | ScreenCommitView | ScreenRefsView |
ScreenStashList | ScreenDiffView | ScreenRebaseEditor | ScreenCmdHistory |
ScreenCommitEditor | ScreenCommitSelect | ScreenBranchSelect
```

Views are **lazily initialized** — pointer-typed views (`*logview.Model`,
`*commitview.Model`, etc.) are created on first navigation, not at startup.

### Git Integration

`internal/git/` — Hybrid approach:

- **go-git** (`go-git/go-git/v5`) for most operations (status, staging, log,
  diff)
- **CLI fallback** via `runGit()`/`runGitFull()` for operations go-git doesn't
  support well
- All commands logged to `cmdlog.Logger` for the command history view (`$` key)

The `Repository` struct wraps `go-git.Repository` and provides the full API
surface used by UI code.

### Key Packages

| Package                    | Purpose                                                                |
| -------------------------- | ---------------------------------------------------------------------- |
| `cmd/termagit`             | Entry point — flag parsing, TTY setup, program bootstrap               |
| `internal/app`             | Root model, screen routing, cross-view coordination                    |
| `internal/git`             | Git operations wrapper (go-git + CLI)                                  |
| `internal/ui/status`       | Status buffer — the main view with 12 sections                         |
| `internal/ui/popup`        | 22 git operation popups (commit, branch, push, pull, etc.)             |
| `internal/ui/commit`       | Commit message editor with vim keybindings                             |
| `internal/ui/shared`       | Shared message types across views                                      |
| `internal/ui/notification` | Stack-based notification overlays                                      |
| `internal/ui/nav`          | Navigation helpers (section jumping, cursor movement)                  |
| `internal/config`          | TOML config loading (`~/.config/termagit/config.toml`)                 |
| `internal/theme`           | Theme registry, token compilation, 3 built-in themes                   |
| `internal/watcher`         | fsnotify file watcher — sends `RepoChangedMsg` on working tree changes |
| `internal/cmdlog`          | Git command history logging                                            |
| `internal/platform`        | Cross-platform utilities (clipboard, browser open)                     |
| `internal/graph`           | Git graph rendering                                                    |

### UI Package Convention

Each view package follows a consistent file layout:

- `model.go` — State struct + `Init()`
- `update.go` — `Update()` message handler
- `view.go` — `View()` renderer
- `keymap.go` — Key bindings
- `messages.go` — Custom `tea.Msg` types

### Cursor Restoration

After git operations (stage, unstage, discard), the status view saves the
current file path and section, reloads status from git, then repositions the
cursor on the same item. This is handled via `cursorRestore` in the status
model.

### Render Caching

The status view caches rendered content (`cachedBaseContent`) and only
re-renders when `contentDirty` is set. The cursor is applied as a post-render
overlay.

## Non-Negotiable: Red/Green TDD

Every piece of code is written test-first. No exceptions.

### The cycle

```
1. RED   — Write a failing test. Run it. Confirm it fails for the RIGHT reason.
2. GREEN — Write the minimum code to make it pass. Run it. Confirm it passes.
3. REFACTOR — Clean up. Run tests again. Must still pass.
```

Never write implementation code before a test exists. Never.

## Testing

Tests use `stretchr/testify` for assertions. Short tests run with `-short` flag
(used by `make test`); integration tests that touch git repos run without it.

Test files mirror production files in the same package (e.g., `model.go` →
`model_test.go`).

## Commit Discipline

```bash
make test   # must be clean before every commit
make lint   # must be clean before every commit
make run    # must start without crashing (from Phase 3 onward)
```

**NEVER** commit code that has not had a manual approval from the user. No
exceptions.

Use conventional commits:

```
feat(cmdhistory): Add proper scrolling and vim-style navigation
feat(popup): Add commit popup with exact Neogit switches and actions
fix(theme): Compile styles once at startup not in View()
```

---

## Dependencies

- **TUI**: `charmbracelet/bubbletea`, `charmbracelet/bubbles`,
  `charmbracelet/lipgloss`
- **Git**: `go-git/go-git/v5`, `go-git/go-billy/v5`
- **Config**: `BurntSushi/toml`
- **Testing**: `stretchr/testify`
