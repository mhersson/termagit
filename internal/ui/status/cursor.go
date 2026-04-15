package status

import (
	tea "github.com/charmbracelet/bubbletea"
)

// handleNextSection moves to the next section header.
func handleNextSection(m Model) (tea.Model, tea.Cmd) {
	visible := visibleSections(m.sections)
	if len(visible) == 0 {
		return m, nil
	}

	// Find current section in visible list
	currentIdx := -1
	for i, v := range visible {
		if v == m.cursor.Section {
			currentIdx = i
			break
		}
	}

	// Move to next visible section
	nextIdx := (currentIdx + 1) % len(visible)
	m.cursor = Cursor{Section: visible[nextIdx], Item: -1, Hunk: -1, Line: -1}

	return m, nil
}

// handlePreviousSection moves to the previous section header.
func handlePreviousSection(m Model) (tea.Model, tea.Cmd) {
	visible := visibleSections(m.sections)
	if len(visible) == 0 {
		return m, nil
	}

	// Find current section in visible list
	currentIdx := 0
	for i, v := range visible {
		if v == m.cursor.Section {
			currentIdx = i
			break
		}
	}

	// Move to previous visible section
	prevIdx := currentIdx - 1
	if prevIdx < 0 {
		prevIdx = len(visible) - 1
	}
	m.cursor = Cursor{Section: visible[prevIdx], Item: -1, Hunk: -1, Line: -1}

	return m, nil
}

// handleNextHunk moves to the next hunk header.
func handleNextHunk(m Model) (tea.Model, tea.Cmd) {
	item, _ := getCurrentItem(m)
	if item == nil || item.Entry == nil || len(item.Hunks) == 0 {
		// No hunks in current item, try next item
		return m, nil
	}

	// If not in hunks yet, go to first hunk
	if m.cursor.Hunk < 0 {
		m.cursor.Hunk = 0
		m.cursor.Line = -1
		return m, nil
	}

	// Move to next hunk
	if m.cursor.Hunk < len(item.Hunks)-1 {
		m.cursor.Hunk++
		m.cursor.Line = -1
	}

	return m, nil
}

// handlePrevHunk moves to the previous hunk header.
func handlePrevHunk(m Model) (tea.Model, tea.Cmd) {
	item, _ := getCurrentItem(m)
	if item == nil || item.Entry == nil || len(item.Hunks) == 0 {
		return m, nil
	}

	// If on first hunk or before, go to item
	if m.cursor.Hunk <= 0 {
		m.cursor.Hunk = -1
		m.cursor.Line = -1
		return m, nil
	}

	// Move to previous hunk
	m.cursor.Hunk--
	m.cursor.Line = -1

	return m, nil
}

func moveCursor(sections []Section, cursor Cursor, dir int) Cursor {
	if len(sections) == 0 {
		return cursor
	}

	// Get list of visible sections
	visible := visibleSections(sections)
	if len(visible) == 0 {
		return cursor
	}

	// Find current position in visible sections
	visIdx := -1
	for i, v := range visible {
		if v == cursor.Section {
			visIdx = i
			break
		}
	}

	// If current section not visible, go to first visible
	if visIdx == -1 {
		return Cursor{Section: visible[0], Item: -1, Hunk: -1, Line: -1}
	}

	s := &sections[cursor.Section]

	if dir > 0 {
		// Moving down
		if cursor.Item == -1 {
			// On section header
			if !s.Folded && len(s.Items) > 0 {
				// Enter section
				return Cursor{Section: cursor.Section, Item: 0, Hunk: -1, Line: -1}
			}
			// Go to next section
			nextVisIdx := visIdx + 1
			if nextVisIdx >= len(visible) {
				return cursor // Stay at boundary
			}
			return Cursor{Section: visible[nextVisIdx], Item: -1, Hunk: -1, Line: -1}
		}

		// On an item
		item := &s.Items[cursor.Item]

		// If on item line (not in hunks) and item is expanded with hunks, enter hunks
		if cursor.Hunk == -1 && item.Expanded && len(item.Hunks) > 0 {
			return Cursor{Section: cursor.Section, Item: cursor.Item, Hunk: 0, Line: -1}
		}

		// If on hunk header, enter diff lines (unless hunk is folded)
		if cursor.Hunk >= 0 && cursor.Line == -1 {
			hunk := &item.Hunks[cursor.Hunk]
			isFolded := len(item.HunksFolded) > cursor.Hunk && item.HunksFolded[cursor.Hunk]
			if !isFolded && len(hunk.Lines) > 0 {
				return Cursor{Section: cursor.Section, Item: cursor.Item, Hunk: cursor.Hunk, Line: 0}
			}
			// Hunk is folded or has no lines, go to next hunk
			if cursor.Hunk < len(item.Hunks)-1 {
				return Cursor{Section: cursor.Section, Item: cursor.Item, Hunk: cursor.Hunk + 1, Line: -1}
			}
			// Last hunk, go to next item
			if cursor.Item < len(s.Items)-1 {
				return Cursor{Section: cursor.Section, Item: cursor.Item + 1, Hunk: -1, Line: -1}
			}
			// Go to next section
			nextVisIdx := visIdx + 1
			if nextVisIdx >= len(visible) {
				return cursor // Stay at boundary
			}
			return Cursor{Section: visible[nextVisIdx], Item: -1, Hunk: -1, Line: -1}
		}

		// If on a diff line, move to next line or next hunk
		if cursor.Line >= 0 {
			hunk := &item.Hunks[cursor.Hunk]
			if cursor.Line < len(hunk.Lines)-1 {
				// Next line in same hunk
				return Cursor{Section: cursor.Section, Item: cursor.Item, Hunk: cursor.Hunk, Line: cursor.Line + 1}
			}
			// Last line, go to next hunk header
			if cursor.Hunk < len(item.Hunks)-1 {
				return Cursor{Section: cursor.Section, Item: cursor.Item, Hunk: cursor.Hunk + 1, Line: -1}
			}
			// Last hunk, go to next item
			if cursor.Item < len(s.Items)-1 {
				return Cursor{Section: cursor.Section, Item: cursor.Item + 1, Hunk: -1, Line: -1}
			}
			// Go to next section
			nextVisIdx := visIdx + 1
			if nextVisIdx >= len(visible) {
				return cursor // Stay at boundary
			}
			return Cursor{Section: visible[nextVisIdx], Item: -1, Hunk: -1, Line: -1}
		}

		// Exit hunks / go to next item (shouldn't reach here with line-level navigation)
		if cursor.Item < len(s.Items)-1 {
			return Cursor{Section: cursor.Section, Item: cursor.Item + 1, Hunk: -1, Line: -1}
		}

		// Go to next section
		nextVisIdx := visIdx + 1
		if nextVisIdx >= len(visible) {
			return cursor // Stay at boundary
		}
		return Cursor{Section: visible[nextVisIdx], Item: -1, Hunk: -1, Line: -1}
	}

	// Moving up (dir < 0)

	// If on a diff line, go to previous line or hunk header
	if cursor.Line > 0 {
		return Cursor{Section: cursor.Section, Item: cursor.Item, Hunk: cursor.Hunk, Line: cursor.Line - 1}
	}

	if cursor.Line == 0 {
		// First line, go to hunk header
		return Cursor{Section: cursor.Section, Item: cursor.Item, Hunk: cursor.Hunk, Line: -1}
	}

	// On hunk header (Line == -1)
	if cursor.Hunk > 0 {
		// Go to last line of previous hunk (unless folded)
		item := &s.Items[cursor.Item]
		prevHunk := &item.Hunks[cursor.Hunk-1]
		isFolded := len(item.HunksFolded) > cursor.Hunk-1 && item.HunksFolded[cursor.Hunk-1]
		if !isFolded && len(prevHunk.Lines) > 0 {
			return Cursor{Section: cursor.Section, Item: cursor.Item, Hunk: cursor.Hunk - 1, Line: len(prevHunk.Lines) - 1}
		}
		// Previous hunk is folded or has no lines, go to its header
		return Cursor{Section: cursor.Section, Item: cursor.Item, Hunk: cursor.Hunk - 1, Line: -1}
	}

	if cursor.Hunk == 0 {
		// First hunk, go to item
		return Cursor{Section: cursor.Section, Item: cursor.Item, Hunk: -1, Line: -1}
	}

	// On item (Hunk == -1)
	if cursor.Item > 0 {
		// Previous item
		prevItem := &s.Items[cursor.Item-1]
		if prevItem.Expanded && len(prevItem.Hunks) > 0 {
			// Go to last line of last hunk of previous item (unless folded)
			lastHunkIdx := len(prevItem.Hunks) - 1
			lastHunk := &prevItem.Hunks[lastHunkIdx]
			isFolded := len(prevItem.HunksFolded) > lastHunkIdx && prevItem.HunksFolded[lastHunkIdx]
			if !isFolded && len(lastHunk.Lines) > 0 {
				return Cursor{Section: cursor.Section, Item: cursor.Item - 1, Hunk: lastHunkIdx, Line: len(lastHunk.Lines) - 1}
			}
			return Cursor{Section: cursor.Section, Item: cursor.Item - 1, Hunk: lastHunkIdx, Line: -1}
		}
		return Cursor{Section: cursor.Section, Item: cursor.Item - 1, Hunk: -1, Line: -1}
	}

	if cursor.Item == 0 {
		// Go to section header
		return Cursor{Section: cursor.Section, Item: -1, Hunk: -1, Line: -1}
	}

	// On section header, go to previous section
	prevVisIdx := visIdx - 1
	if prevVisIdx < 0 {
		return cursor // Stay at boundary
	}
	prevSection := &sections[visible[prevVisIdx]]
	if !prevSection.Folded && len(prevSection.Items) > 0 {
		// Go to last item of previous section
		lastIdx := len(prevSection.Items) - 1
		lastItem := &prevSection.Items[lastIdx]
		if lastItem.Expanded && len(lastItem.Hunks) > 0 {
			// Go to last line of last hunk (unless folded)
			lastHunkIdx := len(lastItem.Hunks) - 1
			lastHunk := &lastItem.Hunks[lastHunkIdx]
			isFolded := len(lastItem.HunksFolded) > lastHunkIdx && lastItem.HunksFolded[lastHunkIdx]
			if !isFolded && len(lastHunk.Lines) > 0 {
				return Cursor{Section: visible[prevVisIdx], Item: lastIdx, Hunk: lastHunkIdx, Line: len(lastHunk.Lines) - 1}
			}
			return Cursor{Section: visible[prevVisIdx], Item: lastIdx, Hunk: lastHunkIdx, Line: -1}
		}
		return Cursor{Section: visible[prevVisIdx], Item: lastIdx, Hunk: -1, Line: -1}
	}
	return Cursor{Section: visible[prevVisIdx], Item: -1, Hunk: -1, Line: -1}
}

// visibleSections returns indices of non-hidden sections.
func visibleSections(sections []Section) []int {
	var visible []int
	for i, s := range sections {
		if !s.Hidden {
			visible = append(visible, i)
		}
	}
	return visible
}

// findFirstValidCursor finds the first non-hidden, non-empty section.
func findFirstValidCursor(sections []Section) Cursor {
	for i, s := range sections {
		if !s.Hidden {
			return Cursor{Section: i, Item: -1, Hunk: -1, Line: -1}
		}
	}
	return Cursor{Section: 0, Item: -1, Hunk: -1, Line: -1}
}

// restoreCursor tries to place the cursor near where the user was before a
// status reload. It searches for the file path in the original section first,
// then falls back to staying at the same index in that section, then falls
// back to findFirstValidCursor.
func restoreCursor(sections []Section, restore cursorRestore) Cursor {
	// Find the section matching the original kind.
	sectionIdx := -1
	for i, s := range sections {
		if s.Kind == restore.sectionKind {
			sectionIdx = i
			break
		}
	}

	// If original section found, look for the file in it.
	if sectionIdx >= 0 {
		s := sections[sectionIdx]
		for itemIdx, item := range s.Items {
			if item.Entry != nil && item.Entry.Path == restore.path {
				return Cursor{Section: sectionIdx, Item: itemIdx, Hunk: -1, Line: -1}
			}
		}
		// File not in original section (it moved). Clamp to same item index.
		if len(s.Items) > 0 {
			idx := restore.itemIndex
			if idx >= len(s.Items) {
				idx = len(s.Items) - 1
			}
			if idx < 0 {
				idx = 0
			}
			return Cursor{Section: sectionIdx, Item: idx, Hunk: -1, Line: -1}
		}
		// Section is empty — try next visible section with items.
		for i := sectionIdx + 1; i < len(sections); i++ {
			if !sections[i].Hidden {
				return Cursor{Section: i, Item: -1, Hunk: -1, Line: -1}
			}
		}
		// Fall through to global fallback.
	}

	return findFirstValidCursor(sections)
}

// preserveUIState transfers user-driven UI state (fold/expand, loaded hunks)
// from old sections to new sections after a watcher-triggered reload. Sections
// are matched by Kind; items are matched by Entry.Path.
func preserveUIState(oldSections, newSections []Section) {
	for _, oldS := range oldSections {
		var newS *Section
		for i := range newSections {
			if newSections[i].Kind == oldS.Kind {
				newS = &newSections[i]
				break
			}
		}
		if newS == nil {
			continue
		}

		newS.Folded = oldS.Folded

		for _, oldItem := range oldS.Items {
			if !oldItem.Expanded || oldItem.Entry == nil {
				continue
			}
			for j := range newS.Items {
				if newS.Items[j].Entry != nil && newS.Items[j].Entry.Path == oldItem.Entry.Path {
					newS.Items[j].Expanded = oldItem.Expanded
					newS.Items[j].Hunks = oldItem.Hunks
					newS.Items[j].HunksFolded = oldItem.HunksFolded
					newS.Items[j].HunksLoading = oldItem.HunksLoading
					break
				}
			}
		}
	}
}

// preserveCursorAcrossReload maps a cursor position from old sections to new
// sections after a watcher-triggered reload. It matches by section kind and
// (when on a file item) by file path, falling back to clamped index, then
// section header, then findFirstValidCursor.
// Importantly, it preserves the Hunk and Line indices when inside an expanded diff.
func preserveCursorAcrossReload(oldSections []Section, oldCursor Cursor, newSections []Section) Cursor {
	if oldCursor.Section < 0 || oldCursor.Section >= len(oldSections) {
		return findFirstValidCursor(newSections)
	}

	oldKind := oldSections[oldCursor.Section].Kind

	// Find the same section kind in new sections.
	newSectionIdx := -1
	for i, s := range newSections {
		if s.Kind == oldKind {
			newSectionIdx = i
			break
		}
	}
	if newSectionIdx < 0 || newSections[newSectionIdx].Hidden {
		return findFirstValidCursor(newSections)
	}

	// Cursor was on section header.
	if oldCursor.Item < 0 {
		return Cursor{Section: newSectionIdx, Item: -1, Hunk: -1, Line: -1}
	}

	newSection := newSections[newSectionIdx]

	// Try to find the same file by path.
	if oldCursor.Item < len(oldSections[oldCursor.Section].Items) {
		oldItem := oldSections[oldCursor.Section].Items[oldCursor.Item]
		if oldItem.Entry != nil {
			for i, item := range newSection.Items {
				if item.Entry != nil && item.Entry.Path == oldItem.Entry.Path {
					// Preserve hunk/line position if the file is still expanded with hunks
					hunk, line := oldCursor.Hunk, oldCursor.Line
					if hunk >= 0 && item.Expanded && len(item.Hunks) > 0 {
						// Clamp hunk index to available hunks
						if hunk >= len(item.Hunks) {
							hunk = len(item.Hunks) - 1
							line = -1 // Can't preserve line if hunk changed
						} else if line >= 0 && line >= len(item.Hunks[hunk].Lines) {
							// Clamp line index if it exceeds available lines
							line = len(item.Hunks[hunk].Lines) - 1
							if line < 0 {
								line = -1
							}
						}
					} else {
						// Not on a hunk or item not expanded - stay on file
						hunk = -1
						line = -1
					}
					return Cursor{Section: newSectionIdx, Item: i, Hunk: hunk, Line: line}
				}
			}
		}
	}

	// File not found — clamp to same index (cursor position within diff is lost).
	if len(newSection.Items) > 0 {
		idx := oldCursor.Item
		if idx >= len(newSection.Items) {
			idx = len(newSection.Items) - 1
		}
		return Cursor{Section: newSectionIdx, Item: idx, Hunk: -1, Line: -1}
	}

	// Section is now empty — fall back to header.
	return Cursor{Section: newSectionIdx, Item: -1, Hunk: -1, Line: -1}
}

// saveCursorContext builds a cursorRestore from the current model state.
func saveCursorContext(m Model) cursorRestore {
	item, sectionKind := getCurrentItem(m)
	if item == nil || item.Entry == nil {
		return cursorRestore{}
	}
	return cursorRestore{
		active:      true,
		path:        item.Entry.Path,
		sectionKind: sectionKind,
		itemIndex:   m.cursor.Item,
		hunk:        m.cursor.Hunk,
	}
}

// handlePageUp scrolls the viewport up by a full page.
func handlePageUp(m Model) (tea.Model, tea.Cmd) {
	m.viewport.YOffset -= m.viewport.Height
	if m.viewport.YOffset < 0 {
		m.viewport.YOffset = 0
	}

	// Move cursor up to stay visible and re-render
	if m.viewport.Width > 0 {
		// Move cursor up by approximately a page worth of items
		for i := 0; i < m.viewport.Height; i++ {
			m.cursor = moveCursor(m.sections, m.cursor, -1)
		}
		m.applyViewportWithCursor()
	}

	return m, nil
}

// handlePageDown scrolls the viewport down by a full page.
func handlePageDown(m Model) (tea.Model, tea.Cmd) {
	maxOffset := m.viewport.TotalLineCount() - m.viewport.Height
	if maxOffset < 0 {
		maxOffset = 0
	}

	m.viewport.YOffset += m.viewport.Height
	if m.viewport.YOffset > maxOffset {
		m.viewport.YOffset = maxOffset
	}

	// Move cursor down to stay visible and re-render
	if m.viewport.Width > 0 {
		// Move cursor down by approximately a page worth of items
		for i := 0; i < m.viewport.Height; i++ {
			m.cursor = moveCursor(m.sections, m.cursor, 1)
		}
		m.applyViewportWithCursor()
	}

	return m, nil
}

// handleHalfPageUp scrolls the viewport up by half a page.
func handleHalfPageUp(m Model) (tea.Model, tea.Cmd) {
	m.viewport.YOffset -= m.viewport.Height / 2
	if m.viewport.YOffset < 0 {
		m.viewport.YOffset = 0
	}

	// Move cursor up to stay visible and re-render
	if m.viewport.Width > 0 {
		// Move cursor up by approximately half a page worth of items
		for i := 0; i < m.viewport.Height/2; i++ {
			m.cursor = moveCursor(m.sections, m.cursor, -1)
		}
		m.applyViewportWithCursor()
	}

	return m, nil
}

// handleHalfPageDown scrolls the viewport down by half a page.
func handleHalfPageDown(m Model) (tea.Model, tea.Cmd) {
	maxOffset := m.viewport.TotalLineCount() - m.viewport.Height
	if maxOffset < 0 {
		maxOffset = 0
	}

	m.viewport.YOffset += m.viewport.Height / 2
	if m.viewport.YOffset > maxOffset {
		m.viewport.YOffset = maxOffset
	}

	// Move cursor down to stay visible and re-render
	if m.viewport.Width > 0 {
		// Move cursor down by approximately half a page worth of items
		for i := 0; i < m.viewport.Height/2; i++ {
			m.cursor = moveCursor(m.sections, m.cursor, 1)
		}
		m.applyViewportWithCursor()
	}

	return m, nil
}

// handleGoToTop moves cursor to the first section header and scrolls to top.
func handleGoToTop(m Model) (tea.Model, tea.Cmd) {
	// Move cursor to first visible section header
	m.cursor = findFirstValidCursor(m.sections)

	// Scroll viewport to top
	m.viewport.YOffset = 0

	// Re-render content to update cursor highlighting
	if m.viewport.Width > 0 {
		m.applyViewportWithCursor()
	}

	return m, nil
}

// handleGoToBottom moves cursor to the last item and scrolls to show it.
func handleGoToBottom(m Model) (tea.Model, tea.Cmd) {
	// Find last visible section with items
	visible := visibleSections(m.sections)
	if len(visible) == 0 {
		return m, nil
	}

	// Start from last visible section and find last item
	for i := len(visible) - 1; i >= 0; i-- {
		s := &m.sections[visible[i]]
		if !s.Folded && len(s.Items) > 0 {
			// Go to last item in this section
			m.cursor = Cursor{
				Section: visible[i],
				Item:    len(s.Items) - 1,
				Hunk:    -1,
				Line:    -1,
			}
			break
		} else if i == 0 {
			// All sections are folded or empty, go to last section header
			m.cursor = Cursor{
				Section: visible[len(visible)-1],
				Item:    -1,
				Hunk:    -1,
				Line:    -1,
			}
		}
	}

	// Re-render to get cursor line, then scroll to show it
	m.applyViewportWithCursor()

	return m, nil
}
