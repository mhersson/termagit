package logview

import (
	"testing"

	"github.com/mhersson/termagit/internal/git"
	"github.com/mhersson/termagit/internal/theme"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestPluralizeTime_Singular(t *testing.T) {
	assert.Equal(t, "1 second", pluralizeTime(1, "second"))
	assert.Equal(t, "1 minute", pluralizeTime(1, "minute"))
	assert.Equal(t, "1 hour", pluralizeTime(1, "hour"))
	assert.Equal(t, "1 day", pluralizeTime(1, "day"))
	assert.Equal(t, "1 year", pluralizeTime(1, "year"))
}

func TestPluralizeTime_Plural(t *testing.T) {
	assert.Equal(t, "0 seconds", pluralizeTime(0, "second"))
	assert.Equal(t, "2 minutes", pluralizeTime(2, "minute"))
	assert.Equal(t, "10 hours", pluralizeTime(10, "hour"))
	assert.Equal(t, "30 days", pluralizeTime(30, "day"))
	assert.Equal(t, "5 years", pluralizeTime(5, "year"))
}

func TestApplyFilter_UsesSearchCache(t *testing.T) {
	commits := []git.LogEntry{
		{AbbreviatedHash: "abc123d", Subject: "Add feature", AuthorName: "Alice"},
		{AbbreviatedHash: "def456a", Subject: "Fix BUG in parser", AuthorName: "Bob"},
		{AbbreviatedHash: "789abcd", Subject: "Update docs", AuthorName: "Charlie"},
	}

	m := New(commits, nil, theme.Compile(theme.RawTokens{}), nil, false, "main")

	// Verify search cache was populated
	require.Len(t, m.searchCache, 3)
	assert.Equal(t, "add feature", m.searchCache[0].subject)
	assert.Equal(t, "abc123d", m.searchCache[0].hash)
	assert.Equal(t, "alice", m.searchCache[0].author)

	// Filter by subject (case-insensitive)
	m.filterText = "bug"
	m.applyFilter()
	require.Len(t, m.filtered, 1)
	assert.Equal(t, 1, m.filtered[0]) // index of "Fix BUG in parser"

	// Filter by author
	m.filterText = "charlie"
	m.applyFilter()
	require.Len(t, m.filtered, 1)
	assert.Equal(t, 2, m.filtered[0])

	// Empty filter clears results
	m.filterText = ""
	m.applyFilter()
	assert.Nil(t, m.filtered)
}
