# termagit


A standalone terminal Git UI, feature-complete and visually identical to
[Neogit](https://github.com/NeogitOrg/neogit). Built with
[Bubble Tea](https://github.com/charmbracelet/bubbletea) and
[go-git](https://github.com/go-git/go-git).

## Why this exists

I love [Magit](https://magit.vc/). It is, without question, the best Git
interface ever made. When I moved away from Emacs I found
[Neogit](https://github.com/NeogitOrg/neogit) — a fantastic Magit port for
Neovim — and it became an essential part of my workflow. But I kept wishing for
something editor-agnostic. Something I could launch from any terminal, in any
project, without needing Neovim running. termagit is that thing.

Every line of code in this project was written by
[Claude Code](https://docs.anthropic.com/en/docs/claude-code). I used this
project to learn how to work with LLMs, to try agentic coding on a real
codebase from scratch, and to see how good these models have actually become. My
role was writing specs, designing prompts, reviewing output, and testing — Claude
did the coding.

I'll be honest: I suffer from a fair bit of imposter syndrome about releasing
this. The code isn't mine. The idea isn't mine either — it's a port of Neogit,
which is itself a port of Magit. But writing the specs, the prompts, and doing
all the testing turned out to be a significant amount of work. It's a completely
new way of building software, and I learned a lot from doing it.

## Features

- **Full status buffer** with all 12 Neogit sections (untracked, unstaged,
  staged, stashes, recent commits, upstream/push-remote tracking, sequencer,
  rebase, bisect)
- **22 popups** matching Neogit's layout exactly — commit, branch, push, pull,
  fetch, merge, rebase, stash, reset, revert, cherry-pick, tag, remote, diff,
  log, bisect, worktree, ignore, and more
- **Inline diffs** — expand files and hunks directly in the status buffer
- **Hunk-level staging** — stage, unstage, and discard individual hunks
- **Multiple views** — log, reflog, commit detail, refs, stash list, diff,
  command history
- **Commit editor** with (a limited set of) vim keybindings and staged diff preview
- **Interactive rebase editor**
- **File watcher** — auto-refreshes when the working tree changes
- **Themeable** — ships with catppuccin-mocha, everforest-dark, and tokyo-night;
  supports custom themes via TOML
- **All Neogit key bindings** — if you know Neogit, you know termagit

## Installation

### From source

```bash
go install github.com/mhersson/termagit/cmd/termagit@latest
```

### Build locally

```bash
git clone https://github.com/mhersson/termagit.git
cd termagit
make build        # binary at bin/termagit
make install      # copies to $GOPATH/bin/
```

Requires Go 1.26+.

## Usage

```bash
termagit                        # open in current directory
termagit -path /path/to/repo    # open a specific repository
termagit -theme everforest-dark # override the color theme
termagit -version               # print version
```

## Configuration

termagit uses a TOML config file at `~/.config/termagit/config.toml` (or
`$XDG_CONFIG_HOME/termagit/config.toml`). Missing fields fall back to sensible
defaults — you only need to specify what you want to change.

See [docs/configuration.md](docs/configuration.md) for the full reference with
all options and their defaults.

Quick example:

```toml
theme = "everforest-dark"

[ui]
recent_commit_count = 20

[sections.stashes]
folded = false
```

## Themes

Three built-in themes are available:

- `catppuccin-mocha` (default)
- `everforest-dark`
- `tokyo-night`

Custom themes can be placed in `~/.config/termagit/themes/` as TOML files. The
easiest way is to define a palette of ~21 colors — termagit maps them to all UI
elements automatically. See [docs/themes.md](docs/themes.md) for the full guide.

## Key Bindings

termagit uses the same key bindings as Neogit. Here are the essentials:

| Key | Action |
|-----|--------|
| `j` / `k` | Move down / up |
| `tab` | Toggle fold |
| `s` / `S` | Stage item / Stage all unstaged |
| `u` / `U` | Unstage item / Unstage all |
| `x` | Discard changes |
| `Enter` | Go to file |
| `c` | Commit popup |
| `b` | Branch popup |
| `P` | Push popup |
| `p` | Pull popup |
| `f` | Fetch popup |
| `m` | Merge popup |
| `r` | Rebase popup |
| `Z` | Stash popup |
| `l` | Log popup |
| `d` | Diff popup |
| `X` | Reset popup |
| `v` | Revert popup |
| `A` | Cherry-pick popup |
| `B` | Bisect popup |
| `t` | Tag popup |
| `w` | Worktree popup |
| `i` | Ignore popup |
| `M` | Remote popup |
| `?` | Help popup |
| `$` | Command history |
| `q` | Close |

## Acknowledgements

- [Magit](https://magit.vc/) — the original and best Git porcelain
- [Neogit](https://github.com/NeogitOrg/neogit) — the Neovim port that termagit
  mirrors
- [Bubble Tea](https://github.com/charmbracelet/bubbletea) and
  [Lip Gloss](https://github.com/charmbracelet/lipgloss) — the TUI framework
  and styling library
- [go-git](https://github.com/go-git/go-git) — pure Go git implementation
- [Claude Code](https://docs.anthropic.com/en/docs/claude-code) — wrote all the
  code
