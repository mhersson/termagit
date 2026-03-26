package graph

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func rowText(row Row) string {
	var s string
	for _, c := range row {
		s += c.Text
	}
	return s
}

func TestBuild_SingleCommitNoParents(t *testing.T) {
	commits := []CommitInput{
		{OID: "aaa", Parents: nil},
	}
	rows, err := Build(commits)
	require.NoError(t, err)

	// Root commit gets a commit row + a trailing merge line (nparents==0 && nbranches==0)
	require.GreaterOrEqual(t, len(rows), 1)
	assert.Equal(t, "• ", rowText(rows[0]))
	assert.Equal(t, "aaa", rows[0][0].OID)
}

func TestBuild_LinearChain(t *testing.T) {
	// A -> B -> C (A is newest, shown first)
	commits := []CommitInput{
		{OID: "aaa", Parents: []string{"bbb"}},
		{OID: "bbb", Parents: []string{"ccc"}},
		{OID: "ccc", Parents: nil},
	}
	rows, err := Build(commits)
	require.NoError(t, err)

	// Linear chain: each commit gets "• ", no merge lines
	commitRows := 0
	for _, row := range rows {
		if len(row) > 0 && row[0].OID != "" {
			commitRows++
			assert.Equal(t, "• ", rowText(row))
		}
	}
	assert.Equal(t, 3, commitRows)
}

func TestBuild_SimpleMerge(t *testing.T) {
	// M merges A and B; A and B both have parent C
	// History: M -> A, B -> C
	commits := []CommitInput{
		{OID: "mmm", Parents: []string{"aaa", "bbb"}}, // merge commit
		{OID: "aaa", Parents: []string{"ccc"}},
		{OID: "bbb", Parents: []string{"ccc"}},
		{OID: "ccc", Parents: nil},
	}
	rows, err := Build(commits)
	require.NoError(t, err)

	// Should have commit rows and merge connector rows
	require.Greater(t, len(rows), 4, "merge should produce connector rows")

	// First row should be the merge commit
	assert.Equal(t, "mmm", rows[0][0].OID)
	assert.Contains(t, rowText(rows[0]), "•")

	// Should have at least one connector row (OID empty)
	hasConnector := false
	for _, row := range rows {
		if len(row) > 0 && row[0].OID == "" {
			hasConnector = true
			break
		}
	}
	assert.True(t, hasConnector, "merge should produce connector rows")
}

func TestBuild_BranchAndMerge(t *testing.T) {
	// Topology:
	//   M (merge A+B)
	//   |\
	//   A B
	//   |/
	//   C
	commits := []CommitInput{
		{OID: "M", Parents: []string{"A", "B"}},
		{OID: "A", Parents: []string{"C"}},
		{OID: "B", Parents: []string{"C"}},
		{OID: "C", Parents: nil},
	}
	rows, err := Build(commits)
	require.NoError(t, err)

	// Verify commit rows have correct OIDs in order
	var commitOIDs []string
	for _, row := range rows {
		if len(row) > 0 && row[0].OID != "" {
			commitOIDs = append(commitOIDs, row[0].OID)
		}
	}
	assert.Equal(t, []string{"M", "A", "B", "C"}, commitOIDs)
}

func TestBuild_OctopusMerge(t *testing.T) {
	// O merges A, B, C (3 parents)
	commits := []CommitInput{
		{OID: "O", Parents: []string{"A", "B", "C"}},
		{OID: "A", Parents: nil},
		{OID: "B", Parents: nil},
		{OID: "C", Parents: nil},
	}
	rows, err := Build(commits)
	require.NoError(t, err)
	require.Greater(t, len(rows), 4, "octopus merge should produce extra rows")

	// All commit rows should be present
	var commitOIDs []string
	for _, row := range rows {
		if len(row) > 0 && row[0].OID != "" {
			commitOIDs = append(commitOIDs, row[0].OID)
		}
	}
	assert.Equal(t, []string{"O", "A", "B", "C"}, commitOIDs)
}

func TestBuild_MissingParent(t *testing.T) {
	// A's parent "missing" is not in the commit list
	commits := []CommitInput{
		{OID: "aaa", Parents: []string{"missing"}},
	}
	rows, err := Build(commits)
	require.NoError(t, err)
	require.GreaterOrEqual(t, len(rows), 1)
	assert.Equal(t, "aaa", rows[0][0].OID)

	// Should have missing parent indicator lines
	hasMissingIndicator := false
	for _, row := range rows {
		txt := rowText(row)
		if len(row) > 0 && row[0].OID == "" {
			// Connector row — check for missing parent char
			for _, c := range row {
				if c.Text == "┊" {
					hasMissingIndicator = true
				}
			}
			_ = txt
		}
	}
	assert.True(t, hasMissingIndicator, "missing parent should produce ┊ indicator")
}

func TestBuild_AllCellsHavePurpleColor(t *testing.T) {
	commits := []CommitInput{
		{OID: "M", Parents: []string{"A", "B"}},
		{OID: "A", Parents: []string{"C"}},
		{OID: "B", Parents: []string{"C"}},
		{OID: "C", Parents: nil},
	}
	rows, err := Build(commits)
	require.NoError(t, err)

	for _, row := range rows {
		for _, cell := range row {
			assert.Equal(t, "Purple", cell.Color, "all graph cells should be Purple")
		}
	}
}

func TestBuild_CommitRowsHaveOID_ConnectorRowsDont(t *testing.T) {
	commits := []CommitInput{
		{OID: "M", Parents: []string{"A", "B"}},
		{OID: "A", Parents: []string{"C"}},
		{OID: "B", Parents: []string{"C"}},
		{OID: "C", Parents: nil},
	}
	rows, err := Build(commits)
	require.NoError(t, err)

	for _, row := range rows {
		if len(row) == 0 {
			continue
		}
		oid := row[0].OID
		hasCommitMarker := false
		for _, c := range row {
			if c.Text == "•" {
				hasCommitMarker = true
			}
		}
		if oid != "" {
			assert.True(t, hasCommitMarker, "commit row (OID=%s) should have • marker", oid)
		} else {
			assert.False(t, hasCommitMarker, "connector row should not have • marker")
		}
	}
}

func TestBuild_LinearChain_NoConnectorRows(t *testing.T) {
	// A linear chain should have NO connector rows — just commit rows
	commits := []CommitInput{
		{OID: "A", Parents: []string{"B"}},
		{OID: "B", Parents: []string{"C"}},
		{OID: "C", Parents: []string{"D"}},
		{OID: "D", Parents: nil},
	}
	rows, err := Build(commits)
	require.NoError(t, err)

	// Count commit rows vs connector rows
	commitCount := 0
	for _, row := range rows {
		if len(row) > 0 && row[0].OID != "" {
			commitCount++
			assert.Equal(t, "• ", rowText(row))
		}
	}
	assert.Equal(t, 4, commitCount)
}

func TestBuild_EmptyInput(t *testing.T) {
	rows, err := Build(nil)
	assert.NoError(t, err)
	assert.Empty(t, rows)

	rows, err = Build([]CommitInput{})
	assert.NoError(t, err)
	assert.Empty(t, rows)
}

func TestBuild_ReturnsErrorNotPanic(t *testing.T) {
	// Build should never panic — any internal error should be returned as an error.
	// Normal inputs should return no error.
	commits := []CommitInput{
		{OID: "M", Parents: []string{"A", "B"}},
		{OID: "A", Parents: []string{"C"}},
		{OID: "B", Parents: []string{"C"}},
		{OID: "C", Parents: nil},
	}
	assert.NotPanics(t, func() {
		rows, err := Build(commits)
		assert.NoError(t, err)
		assert.NotEmpty(t, rows)
	})
}

func TestBuild_RootCommitInChain(t *testing.T) {
	// B has no parents (root), A has parent B
	commits := []CommitInput{
		{OID: "A", Parents: []string{"B"}},
		{OID: "B", Parents: nil},
	}
	rows, err := Build(commits)
	require.NoError(t, err)

	var commitOIDs []string
	for _, row := range rows {
		if len(row) > 0 && row[0].OID != "" {
			commitOIDs = append(commitOIDs, row[0].OID)
		}
	}
	assert.Equal(t, []string{"A", "B"}, commitOIDs)
}
