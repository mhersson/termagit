package git

import (
	"context"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestListTags_Empty(t *testing.T) {
	r := newMemRepo(t)
	ctx := context.Background()

	tags, err := r.ListTags(ctx)
	require.NoError(t, err)
	require.Empty(t, tags)
}

func TestCreateTag_Lightweight(t *testing.T) {
	skipInShort(t)

	r := newTempRepo(t)
	ctx := context.Background()

	head, err := r.HeadOID(ctx)
	require.NoError(t, err)

	err = r.CreateTag(ctx, "v1.0.0", head, TagOpts{})
	require.NoError(t, err)

	tags, err := r.ListTags(ctx)
	require.NoError(t, err)
	require.Contains(t, tags, "v1.0.0")
}

func TestCreateTag_Annotated(t *testing.T) {
	skipInShort(t)

	r := newTempRepo(t)
	ctx := context.Background()

	head, err := r.HeadOID(ctx)
	require.NoError(t, err)

	err = r.CreateTag(ctx, "v2.0.0", head, TagOpts{
		Annotate: true,
		Message:  "Release v2.0.0",
	})
	require.NoError(t, err)

	tags, err := r.ListTags(ctx)
	require.NoError(t, err)
	require.Contains(t, tags, "v2.0.0")
}

func TestDeleteTag_RemovesTag(t *testing.T) {
	skipInShort(t)

	r := newTempRepo(t)
	ctx := context.Background()

	head, err := r.HeadOID(ctx)
	require.NoError(t, err)

	err = r.CreateTag(ctx, "v1.0.0", head, TagOpts{})
	require.NoError(t, err)

	err = r.DeleteTag(ctx, "v1.0.0")
	require.NoError(t, err)

	tags, err := r.ListTags(ctx)
	require.NoError(t, err)
	require.NotContains(t, tags, "v1.0.0")
}

func TestCreateTag_Force_OverwritesExisting(t *testing.T) {
	skipInShort(t)

	r := newTempRepo(t)
	ctx := context.Background()

	head, err := r.HeadOID(ctx)
	require.NoError(t, err)

	err = r.CreateTag(ctx, "v1.0.0", head, TagOpts{})
	require.NoError(t, err)

	// Create a second commit
	addAndCommit(t, r, "new.txt", "content", "New commit")
	head2, err := r.HeadOID(ctx)
	require.NoError(t, err)

	// Force overwrite the tag
	err = r.CreateTag(ctx, "v1.0.0", head2, TagOpts{Force: true})
	require.NoError(t, err)

	// Tag should still exist
	tags, err := r.ListTags(ctx)
	require.NoError(t, err)
	require.Contains(t, tags, "v1.0.0")
}
