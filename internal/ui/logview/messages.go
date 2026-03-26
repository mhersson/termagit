package logview

// CloseLogViewMsg signals that the log view should close.
type CloseLogViewMsg struct{}

// LoadMoreMsg requests loading more commits.
type LoadMoreMsg struct{}

// CommitsLoadedMsg delivers newly loaded commits.
type CommitsLoadedMsg struct {
	Commits []logCommit
	HasMore bool
	Err     error
}

// logCommit is an internal type for loaded commits.
type logCommit struct {
	hash, abbrevHash, subject, authorName, parentHashes string
}
