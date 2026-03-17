package git

// StashEntry represents a stash in the stash list.
type StashEntry struct {
	Index   int    // Stash index (0, 1, 2, ...)
	Name    string // Full stash name (e.g., "stash@{0}")
	Message string // Stash message (e.g., "WIP on main: abc123 commit message")
	Branch  string // Branch the stash was created on
}
