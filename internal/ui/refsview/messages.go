package refsview

// CloseRefsViewMsg signals that the refs view should close.
type CloseRefsViewMsg struct{}

// DeleteBranchMsg carries the result of a branch deletion attempt.
type DeleteBranchMsg struct {
	Err error
}

// RefsRefreshedMsg carries refreshed refs data after a mutation.
type RefsRefreshedMsg struct {
	Err error
}
