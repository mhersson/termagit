package status

import (
	tea "github.com/charmbracelet/bubbletea"

	"github.com/mhersson/termagit/internal/git"
)

func handleToggle(m Model) (tea.Model, tea.Cmd) {
	if m.cursor.Section >= len(m.sections) {
		return m, nil
	}

	// Save current screen position before toggling
	m.ensureContent()
	oldCursorLine := computeCursorLine(m)
	screenRow := oldCursorLine - m.viewport.YOffset

	s := &m.sections[m.cursor.Section]

	if m.cursor.Item == -1 {
		// Toggle section fold
		s.Folded = !s.Folded
	} else if m.cursor.Item < len(s.Items) {
		item := &s.Items[m.cursor.Item]

		// If on a hunk (header or line), toggle hunk fold
		if m.cursor.Hunk >= 0 && m.cursor.Hunk < len(item.Hunks) {
			// Ensure HunksFolded slice is initialized
			if item.HunksFolded == nil {
				item.HunksFolded = make([]bool, len(item.Hunks))
			}
			// Toggle hunk fold
			item.HunksFolded[m.cursor.Hunk] = !item.HunksFolded[m.cursor.Hunk]
			// If we were on a line, move to hunk header
			if m.cursor.Line >= 0 {
				m.cursor.Line = -1
			}

			// Update viewport with preserved screen position
			m.invalidateContent()
			if m.viewport.Width > 0 {
				m.applyViewportWithCursor()
				preserveScreenPosition(&m, computeCursorLine(m), screenRow)
			}
			return m, nil
		}

		// Toggle item expansion
		item.Expanded = !item.Expanded

		// Load hunks if expanding and not loaded
		if item.Expanded && item.Hunks == nil && item.Entry != nil && !item.HunksLoading {
			item.HunksLoading = true
			kind := diffKindForSection(s.Kind)

			// Update viewport with preserved screen position
			m.invalidateContent()
			if m.viewport.Width > 0 {
				m.applyViewportWithCursor()
				preserveScreenPosition(&m, computeCursorLine(m), screenRow)
			}
			return m, loadHunksCmd(m.repo, m.cursor.Section, m.cursor.Item, item.Entry, kind)
		}
	}

	// Update viewport with preserved screen position
	m.invalidateContent()
	if m.viewport.Width > 0 {
		m.applyViewportWithCursor()
		preserveScreenPosition(&m, computeCursorLine(m), screenRow)
	}

	return m, nil
}

// handleOpenFold opens the current fold.
func handleOpenFold(m Model) (tea.Model, tea.Cmd) {
	if m.cursor.Section >= len(m.sections) {
		return m, nil
	}

	s := &m.sections[m.cursor.Section]

	if m.cursor.Item == -1 {
		s.Folded = false
	} else if m.cursor.Item < len(s.Items) {
		item := &s.Items[m.cursor.Item]
		if !item.Expanded {
			item.Expanded = true
			if item.Hunks == nil && item.Entry != nil && !item.HunksLoading {
				item.HunksLoading = true
				kind := diffKindForSection(s.Kind)
				m.invalidateContent()
				if m.viewport.Width > 0 {
					m.applyViewportWithCursor()
				}
				return m, loadHunksCmd(m.repo, m.cursor.Section, m.cursor.Item, item.Entry, kind)
			}
		}
	}

	m.invalidateContent()
	if m.viewport.Width > 0 {
		m.applyViewportWithCursor()
	}
	return m, nil
}

// handleCloseFold closes the current fold.
func handleCloseFold(m Model) (tea.Model, tea.Cmd) {
	if m.cursor.Section >= len(m.sections) {
		return m, nil
	}

	s := &m.sections[m.cursor.Section]

	if m.cursor.Item == -1 {
		s.Folded = true
	} else if m.cursor.Item < len(s.Items) {
		s.Items[m.cursor.Item].Expanded = false
	}

	m.invalidateContent()
	if m.viewport.Width > 0 {
		m.applyViewportWithCursor()
	}
	return m, nil
}

// handleDepth sets fold depth: 1=headers only, 2=items, 3=hunks, 4=all.
func handleDepth(m Model, depth int) (tea.Model, tea.Cmd) {
	for i := range m.sections {
		s := &m.sections[i]
		switch depth {
		case 1:
			s.Folded = true
			for j := range s.Items {
				s.Items[j].Expanded = false
			}
		case 2:
			s.Folded = false
			for j := range s.Items {
				s.Items[j].Expanded = false
			}
		case 3, 4:
			s.Folded = false
			for j := range s.Items {
				s.Items[j].Expanded = true
			}
		}
	}
	m.invalidateContent()
	if m.viewport.Width > 0 {
		m.applyViewportWithCursor()
	}
	return m, nil
}

func diffKindForSection(kind SectionKind) git.DiffKind {
	switch kind {
	case SectionStaged:
		return git.DiffStaged
	default:
		return git.DiffUnstaged
	}
}
