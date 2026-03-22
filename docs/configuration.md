# Configuration

termagit is configured via a TOML file at:

```
~/.config/termagit/config.toml
```

Or more precisely, `$XDG_CONFIG_HOME/termagit/config.toml`. If the file doesn't
exist, all defaults are used.

You only need to specify the values you want to change. Missing fields keep
their defaults — the config is merged on top of the defaults, not replaced.

## Theme

| Key     | Type   | Default              | Description                                                                                                                                         |
| ------- | ------ | -------------------- | --------------------------------------------------------------------------------------------------------------------------------------------------- |
| `theme` | string | `"catppuccin-mocha"` | Color theme. Built-in options: `catppuccin-mocha`, `everforest-dark`, `tokyo-night`. Can also reference a custom theme file name (without `.toml`). |

```toml
theme = "everforest-dark"
```

Custom themes are loaded from `~/.config/termagit/themes/*.toml`. A partial
theme file inherits missing values from the default (catppuccin-mocha).

## Git

Settings related to git operations.

| Key                 | Type   | Default            | Description                                                                               |
| ------------------- | ------ | ------------------ | ----------------------------------------------------------------------------------------- |
| `git.executable`    | string | `"git"`            | Path to the git binary. Not yet implemented.                                              |
| `git.sort_branches` | string | `"-committerdate"` | Branch sort order. Not yet implemented.                                                   |
| `git.commit_order`  | string | `"topo"`           | Commit log ordering. Values: `topo`, `date`, `author-date`, or `""`. Not yet implemented. |
| `git.graph_style`   | string | `"unicode"`        | Graph drawing style. Values: `ascii`, `unicode`, `kitty`. Not yet implemented.            |

```toml
[git]
# None of the git options are implemented yet.
# They are reserved for future use.
```

## UI

General user interface settings.

| Key                               | Type   | Default | Description                                                                              |
| --------------------------------- | ------ | ------- | ---------------------------------------------------------------------------------------- |
| `ui.disable_hint`                 | bool   | `false` | Hide the hint bar at the bottom of the status buffer.                                    |
| `ui.disable_context_highlighting` | bool   | `false` | Disable highlighting of the item under the cursor. Not yet implemented.                  |
| `ui.disable_signs`                | bool   | `false` | Hide the sign column (the `+`/`-` markers in diffs). Not yet implemented.                |
| `ui.disable_line_numbers`         | bool   | `false` | Hide line numbers in diff views.                                                         |
| `ui.show_head_commit_hash`        | bool   | `false` | Show the abbreviated commit hash next to HEAD in the status header. Not yet implemented. |
| `ui.recent_commit_count`          | int    | `10`    | Number of commits shown in the "Recent commits" section.                                 |
| `ui.HEAD_padding`                 | int    | `10`    | Padding width for HEAD info labels (Head, Merge, Push, Tag).                             |
| `ui.HEAD_folded`                  | bool   | `false` | Start with the HEAD info section folded. Not yet implemented.                            |
| `ui.mode_padding`                 | int    | `3`     | Padding width for file mode labels (e.g., `modified`, `new file`). Not yet implemented.  |
| `ui.notification_icon`            | string | `"󰐗"`   | Icon shown in notification toasts. Not yet implemented.                                  |
| `ui.console_timeout`              | int    | `5000`  | How long console notifications stay visible, in milliseconds. Not yet implemented.       |
| `ui.auto_show_console`            | bool   | `true`  | Automatically show the console when a git command runs. Not yet implemented.             |
| `ui.auto_close_console`           | bool   | `true`  | Automatically hide the console when a git command finishes. Not yet implemented.         |

```toml
[ui]
recent_commit_count = 20
disable_hint = true
```

## Commit Editor

Settings for the built-in commit message editor.

| Key                                             | Type   | Default   | Description                                                           |
| ----------------------------------------------- | ------ | --------- | --------------------------------------------------------------------- |
| `commit_editor.show_staged_diff`                | bool   | `true`    | Show the staged diff alongside the commit message editor.             |
| `commit_editor.staged_diff_split_kind`          | string | `"split"` | Reserved for future use. Not yet implemented.                         |
| `commit_editor.spell_check`                     | bool   | `false`   | Reserved for future use. Not yet implemented.                         |
| `commit_editor.disable_insert_on_commit`        | bool   | `false`   | Don't enter insert mode automatically when opening the commit editor. |
| `commit_editor.generate_commit_message_command` | string | `""`      | External command to generate commit messages. Empty means disabled.   |

```toml
[commit_editor]
show_staged_diff = true
disable_insert_on_commit = true
```

## Commit View

Settings for the commit detail view.

| Key                         | Type | Default | Description                                                      |
| --------------------------- | ---- | ------- | ---------------------------------------------------------------- |
| `commit_view.verify_commit` | bool | `true`  | Verify GPG signatures when viewing commits. Not yet implemented. |

```toml
[commit_view]
# verify_commit is not yet implemented
```

## File Watcher

| Key                   | Type | Default | Description                                                            |
| --------------------- | ---- | ------- | ---------------------------------------------------------------------- |
| `filewatcher.enabled` | bool | `true`  | Watch the working tree for changes and auto-refresh the status buffer. |

```toml
[filewatcher]
enabled = false
```

## Sections

Each of the 12 status buffer sections can be independently configured with
`folded` (start collapsed) and `hidden` (don't show at all).

| Section                        | TOML key                       | Default `folded` | Default `hidden` |
| ------------------------------ | ------------------------------ | :--------------: | :--------------: |
| Sequencer (cherry-pick/revert) | `sections.sequencer`           |     `false`      |     `false`      |
| Untracked files                | `sections.untracked`           |     `false`      |     `false`      |
| Unstaged changes               | `sections.unstaged`            |     `false`      |     `false`      |
| Staged changes                 | `sections.staged`              |     `false`      |     `false`      |
| Stashes                        | `sections.stashes`             |      `true`      |     `false`      |
| Unpulled from upstream         | `sections.unpulled_upstream`   |      `true`      |     `false`      |
| Unmerged into upstream         | `sections.unmerged_upstream`   |     `false`      |     `false`      |
| Unpulled from push remote      | `sections.unpulled_pushremote` |      `true`      |     `false`      |
| Unmerged into push remote      | `sections.unmerged_pushremote` |     `false`      |     `false`      |
| Recent commits                 | `sections.recent`              |     `false`      |     `false`      |
| Rebase                         | `sections.rebase`              |     `false`      |     `false`      |
| Bisect                         | `sections.bisect`              |     `false`      |     `false`      |

```toml
[sections.stashes]
folded = false

[sections.recent]
hidden = true
```

## Command Log

Settings for the internal command log (stored at
`~/.local/state/termagit/commands.log`).

| Key            | Type   | Default  | Description                                                                |
| -------------- | ------ | -------- | -------------------------------------------------------------------------- |
| `log.max_size` | string | `"10MB"` | Maximum log file size before rotation. Supports `KB`, `MB`, `GB` suffixes. |
| `log.keep`     | int    | `3`      | Number of rotated log files to keep.                                       |

```toml
[log]
max_size = "50MB"
keep = 5
```

## Full Example

A config file with every option set to its default value:

```toml
theme = "catppuccin-mocha"

[git]
executable = "git"
sort_branches = "-committerdate"
commit_order = "topo"
graph_style = "unicode"

[ui]
disable_hint = false
disable_context_highlighting = false
disable_signs = false
disable_line_numbers = false
show_head_commit_hash = false
recent_commit_count = 10
HEAD_padding = 10
HEAD_folded = false
mode_padding = 3
notification_icon = "󰐗"
console_timeout = 5000
auto_show_console = true
auto_close_console = true

[commit_editor]
show_staged_diff = true
staged_diff_split_kind = "split"
spell_check = false
disable_insert_on_commit = false
generate_commit_message_command = ""

[commit_view]
verify_commit = true

[filewatcher]
enabled = true

[sections.sequencer]
folded = false
hidden = false

[sections.untracked]
folded = false
hidden = false

[sections.unstaged]
folded = false
hidden = false

[sections.staged]
folded = false
hidden = false

[sections.stashes]
folded = true
hidden = false

[sections.unpulled_upstream]
folded = true
hidden = false

[sections.unmerged_upstream]
folded = false
hidden = false

[sections.unpulled_pushremote]
folded = true
hidden = false

[sections.unmerged_pushremote]
folded = false
hidden = false

[sections.recent]
folded = false
hidden = false

[sections.rebase]
folded = false
hidden = false

[sections.bisect]
folded = false
hidden = false

[log]
max_size = "10MB"
keep = 3
```
