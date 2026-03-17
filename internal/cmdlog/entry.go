package cmdlog

import "time"

// Entry represents a single command execution record.
type Entry struct {
	Timestamp  time.Time `json:"ts"`
	Command    string    `json:"cmd"`
	Dir        string    `json:"cwd"`
	ExitCode   int       `json:"exit"`
	Stdout     string    `json:"stdout"`
	Stderr     string    `json:"stderr"`
	DurationMs int64     `json:"ms"`
}
