package git

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestListSubmodules_EmptyWhenNone(t *testing.T) {
	skipInShort(t)
	r := newTempRepo(t)
	ctx := context.Background()

	subs, err := r.ListSubmodules(ctx)
	require.NoError(t, err)
	assert.Empty(t, subs)
}

func TestParentRepo_EmptyWhenNotSubmodule(t *testing.T) {
	skipInShort(t)
	r := newTempRepo(t)
	ctx := context.Background()

	parent, err := r.ParentRepo(ctx)
	require.NoError(t, err)
	assert.Empty(t, parent)
}
