package git

import (
	"context"
	"fmt"
	"strings"

	gogit "github.com/go-git/go-git/v5"
	"github.com/go-git/go-git/v5/plumbing/object"
	"github.com/go-git/go-git/v5/plumbing/storer"
)

// CommitHistory returns the last n commit message subjects, newest first.
func (r *Repository) CommitHistory(ctx context.Context, n int) ([]string, error) {
	var subjects []string

	_, _, err := r.logOp(ctx, fmt.Sprintf("git log -%d --format=%%s", n), func() (string, string, error) {
		head, err := r.raw.Head()
		if err != nil {
			return "", "", fmt.Errorf("get HEAD: %w", err)
		}

		iter, err := r.raw.Log(&gogit.LogOptions{From: head.Hash()})
		if err != nil {
			return "", "", fmt.Errorf("log: %w", err)
		}

		count := 0
		err = iter.ForEach(func(c *object.Commit) error {
			if count >= n {
				return storer.ErrStop
			}
			subject := strings.Split(c.Message, "\n")[0]
			subjects = append(subjects, subject)
			count++
			return nil
		})
		if err != nil {
			return "", "", fmt.Errorf("iterate commits: %w", err)
		}

		return strings.Join(subjects, "\n"), "", nil
	})

	return subjects, err
}

// CommitHistoryCycler navigates through previous commit messages.
type CommitHistoryCycler struct {
	messages []string
	idx      int    // -1 = at current (not yet navigated)
	current  string // saved user input before first navigation
}

// NewCycler creates a cycler from a list of commit message subjects.
func NewCycler(messages []string) *CommitHistoryCycler {
	return &CommitHistoryCycler{
		messages: messages,
		idx:      -1,
	}
}

// Prev moves to the previous (older) message.
// On first call, saves current as the user's draft.
func (c *CommitHistoryCycler) Prev(current string) string {
	if len(c.messages) == 0 {
		return current
	}

	if c.idx == -1 {
		// First navigation: save current text
		c.current = current
		c.idx = 0
		return c.messages[0]
	}

	if c.idx < len(c.messages)-1 {
		c.idx++
	}
	return c.messages[c.idx]
}

// Next moves to the next (newer) message.
// Returns the saved draft when reaching the beginning.
func (c *CommitHistoryCycler) Next() string {
	if c.idx <= 0 {
		c.idx = -1
		return c.current
	}

	c.idx--
	return c.messages[c.idx]
}

// Reset returns to the saved draft, resetting navigation state.
func (c *CommitHistoryCycler) Reset() string {
	c.idx = -1
	return c.current
}

// CommitMessagesForCycling returns full commit messages (not just subjects).
// Skips merge commits to match Neogit behavior. Returns newest first.
func (r *Repository) CommitMessagesForCycling(ctx context.Context, n int) ([]string, error) {
	var messages []string

	_, _, err := r.logOp(ctx, fmt.Sprintf("git log -%d --format=%%B --no-merges", n), func() (string, string, error) {
		head, err := r.raw.Head()
		if err != nil {
			return "", "", fmt.Errorf("get HEAD: %w", err)
		}

		iter, err := r.raw.Log(&gogit.LogOptions{From: head.Hash()})
		if err != nil {
			return "", "", fmt.Errorf("log: %w", err)
		}

		count := 0
		err = iter.ForEach(func(c *object.Commit) error {
			if count >= n {
				return storer.ErrStop
			}
			// Skip merge commits (more than 1 parent)
			if c.NumParents() > 1 {
				return nil
			}
			// Use full message (trimmed of trailing whitespace)
			messages = append(messages, strings.TrimSpace(c.Message))
			count++
			return nil
		})
		if err != nil {
			return "", "", fmt.Errorf("iterate commits: %w", err)
		}

		return strings.Join(messages, "\x00"), "", nil
	})

	return messages, err
}
