# Custom Themes

termagit supports custom color themes via TOML files placed in
`~/.config/termagit/themes/` (or `$XDG_CONFIG_HOME/termagit/themes/`).

There are two ways to define a theme: **palette-based** (recommended) and
**token-based** (advanced). You can also combine both approaches.

## Palette-Based Themes (Recommended)

A palette defines ~21 named colors. termagit maps these to all the UI elements
automatically. This is the simplest way to create a theme.

Create a file like `~/.config/termagit/themes/my-theme.toml`:

```toml
[palette]
bg           = "#1e1e2e"   # base background
bg1          = "#313244"   # surface - cursor highlight, float header bg
bg2          = "#45475a"   # surface - selection bg, diff header bg
bg3          = "#585b70"   # surface - popup border
diff_add_bg  = "#1e3a2f"   # diff added line background
diff_del_bg  = "#3b1f29"   # diff deleted line background
fg           = "#cdd6f4"   # bright foreground - bold text, popup title, cursor
fg1          = "#cdd6f4"   # normal foreground - body text, popup actions
fg2          = "#bac2de"   # secondary foreground - commit date, diff context
dim          = "#6c7086"   # dimmed - comments, subtle text, graph gray
dim1         = "#7f849c"   # slightly brighter dim - hash, untracked label
blue         = "#89b4fa"   # branches, modified, popup key, info notifications
green        = "#a6e3a1"   # remote, staged, added, diff add, success
red          = "#f38ba8"   # deleted, conflict, diff delete, error
yellow       = "#f9e2af"   # tag, unstaged, option, confirm key
purple       = "#cba6f7"   # section header, renamed, popup section, stashes
teal         = "#94e2d5"   # hunk header, copied, rebasing
cyan         = "#89dceb"   # popup switch, commit view header, float header fg
orange       = "#fab387"   # warning, confirm border, numbers
pink         = "#f5c2e7"   # commit author, merging
lavender     = "#b4befe"   # current commit hash
```

Then set it in your config:

```toml
theme = "my-theme"
```

### Palette Fields

| Field         | Used For                                                                                                   |
| ------------- | ---------------------------------------------------------------------------------------------------------- |
| `bg`          | Base background, commit view header text                                                                   |
| `bg1`         | Cursor highlight, editor bar bg, float header bg                                                           |
| `bg2`         | Selection bg, diff header bg                                                                               |
| `bg3`         | Popup border                                                                                               |
| `diff_add_bg` | Background for added diff lines                                                                            |
| `diff_del_bg` | Background for deleted diff lines                                                                          |
| `fg`          | Bold text, popup title, cursor, graph white                                                                |
| `fg1`         | Normal text, popup action labels, confirm text                                                             |
| `fg2`         | Commit date, diff context lines                                                                            |
| `dim`         | Comments, subtle text, rebase done items, graph gray                                                       |
| `dim1`        | Commit hash, untracked file label                                                                          |
| `blue`        | Branch names, modified indicator, popup key, info notifications, file paths, diff header fg                |
| `green`       | Remote names, staged indicator, added indicator, diff add lines, success notifications, cherry-pick header |
| `red`         | Deleted indicator, conflict indicator, diff delete lines, error notifications, revert header               |
| `yellow`      | Tag names, unstaged indicator, popup option labels, confirm key, bisect header                             |
| `purple`      | Section headers, renamed indicator, popup section headers, stash indicator                                 |
| `teal`        | Hunk headers, copied indicator, rebase header                                                              |
| `cyan`        | Popup switch labels, commit view header bg, float header fg, graph cyan                                    |
| `orange`      | Warning notifications, confirm border, numbers                                                             |
| `pink`        | Commit author, merge header                                                                                |
| `lavender`    | Current commit hash (HEAD)                                                                                 |

### Tips

- **`fg` vs `fg1`**: In most themes these are the same color. Set `fg1` to a
  slightly dimmer shade if you want normal body text to be subtler than
  bold/title text (like Tokyo Night does).
- **Reusing colors**: If your color scheme doesn't distinguish teal from cyan,
  or pink from purple, just set them to the same value. The built-in
  everforest-dark and tokyo-night themes do this.
- **Diff backgrounds**: These should be very dark, desaturated versions of your
  green and red. They're used as background colors for added/deleted diff lines.

## Token-Based Themes (Advanced)

For fine-grained control, you can override individual tokens directly. Token
fields are set at the top level of the TOML file (outside any section).

```toml
normal         = "#d4d4d4"
branch         = "#569cd6"
section_header = "#c586c0"
diff_add       = "#4ec9b0"
# ... any of the 67 token fields
```

Missing fields are filled from the default theme (catppuccin-mocha).

### All Token Fields

| Token                   | Description                           |
| ----------------------- | ------------------------------------- |
| `normal`                | Base text                             |
| `bold`                  | Bold text                             |
| `dim`                   | Dimmed/subtle text                    |
| `comment`               | Comment text (italic)                 |
| `branch`                | Branch names (bold)                   |
| `branch_head`           | HEAD branch (bold + underline)        |
| `remote`                | Remote names (bold)                   |
| `tag`                   | Tag names (bold)                      |
| `hash`                  | Commit hashes                         |
| `hash_current`          | Current/HEAD commit hash (bold)       |
| `commit_author`         | Author names                          |
| `commit_date`           | Date/time                             |
| `section_header`        | Status section titles (bold)          |
| `diff_add`              | Added line foreground                 |
| `diff_add_bg`           | Added line background                 |
| `diff_delete`           | Deleted line foreground               |
| `diff_delete_bg`        | Deleted line background               |
| `diff_context`          | Context line text                     |
| `diff_hunk_header`      | Hunk header (@@ lines, bold)          |
| `change_modified`       | "modified" label (bold)               |
| `change_added`          | "new file" label (bold)               |
| `change_deleted`        | "deleted" label (bold)                |
| `change_renamed`        | "renamed" label (bold + italic)       |
| `change_copied`         | "copied" label (bold + italic)        |
| `change_untracked`      | Untracked file label (bold)           |
| `staged`                | Staged indicator                      |
| `unstaged`              | Unstaged indicator                    |
| `conflict`              | Conflict indicator (bold)             |
| `popup_border`          | Popup frame                           |
| `popup_title`           | Popup title (bold)                    |
| `popup_key`             | Popup keybinding (bold)               |
| `popup_key_bg`          | Unused (reserved)                     |
| `popup_switch`          | Popup toggle switch                   |
| `popup_option`          | Popup option label                    |
| `popup_action`          | Popup action label                    |
| `popup_section`         | Popup section heading (bold)          |
| `notification_info`     | Info notification                     |
| `notification_success`  | Success notification                  |
| `notification_warn`     | Warning notification                  |
| `notification_error`    | Error notification (bold)             |
| `confirm_border`        | Confirm dialog frame                  |
| `confirm_text`          | Confirm dialog text                   |
| `confirm_key`           | Confirm dialog keybinding (bold)      |
| `cursor`                | Cursor foreground                     |
| `cursor_bg`             | Cursor background highlight           |
| `selection`             | Selection foreground                  |
| `select_bg`             | Selection background                  |
| `background`            | Terminal background                   |
| `graph_orange`          | Graph line color                      |
| `graph_green`           | Graph line color                      |
| `graph_red`             | Graph line color                      |
| `graph_blue`            | Graph line color                      |
| `graph_yellow`          | Graph line color                      |
| `graph_cyan`            | Graph line color                      |
| `graph_purple`          | Graph line color                      |
| `graph_gray`            | Graph line color                      |
| `graph_white`           | Graph line color                      |
| `merging`               | Merge in-progress header (bold)       |
| `rebasing`              | Rebase in-progress header (bold)      |
| `picking`               | Cherry-pick in-progress header (bold) |
| `reverting`             | Revert in-progress header (bold)      |
| `bisecting`             | Bisect in-progress header (bold)      |
| `rebase_done`           | Completed rebase items                |
| `subtle_text`           | Secondary/subtle text                 |
| `stashes`               | Stash indicator (bold)                |
| `commit_view_header`    | Commit view header background         |
| `commit_view_header_fg` | Commit view header foreground         |
| `file_path`             | File paths (italic)                   |
| `number`                | Numbers in stats                      |
| `diff_header`           | Diff view header background           |
| `diff_header_fg`        | Diff view header foreground           |
| `float_header`          | Float header background               |
| `float_header_fg`       | Float header foreground               |

## Mixed: Palette + Token Overrides

You can define a palette for the base colors and override specific tokens on
top. Token overrides take precedence over the palette-generated values.

```toml
# Use palette for the base
[palette]
bg   = "#1e1e2e"
fg   = "#cdd6f4"
fg1  = "#cdd6f4"
# ... rest of palette

# Override specific tokens
graph_gray = "#585b70"
diff_context = "#a6adc8"
```

This is useful when a palette gets you 95% of the way there, but a few tokens
need specific colors that don't fit the standard mapping.

## Built-in Themes

termagit ships with three built-in themes:

- **catppuccin-mocha** (default) - warm, pastel colors on a dark base
- **everforest-dark** - muted, earthy tones inspired by nature
- **tokyo-night** - cool blues and purples with vibrant accents
