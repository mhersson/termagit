package status

import (
	"testing"

	"github.com/mhersson/conjit/internal/git"
	"github.com/mhersson/conjit/internal/ui/popup"
	"github.com/stretchr/testify/assert"
)

func TestBuildPushOpts_DefaultSwitches(t *testing.T) {
	result := popup.Result{
		Switches: map[string]bool{},
		Options:  map[string]string{},
	}

	opts := buildPushOpts(result)
	assert.False(t, opts.Force)
	assert.False(t, opts.ForceWithLease)
	assert.False(t, opts.DryRun)
	assert.False(t, opts.SetUpstream)
	assert.False(t, opts.NoVerify)
	assert.False(t, opts.Tags)
	assert.False(t, opts.FollowTags)
}

func TestBuildPushOpts_AllSwitchesEnabled(t *testing.T) {
	result := popup.Result{
		Switches: map[string]bool{
			"force-with-lease": true,
			"force":            true,
			"no-verify":        true,
			"dry-run":          true,
			"set-upstream":     true,
			"tags":             true,
			"follow-tags":      true,
		},
		Options: map[string]string{},
	}

	opts := buildPushOpts(result)
	assert.True(t, opts.ForceWithLease)
	assert.True(t, opts.Force)
	assert.True(t, opts.DryRun)
	assert.True(t, opts.SetUpstream)
	assert.True(t, opts.NoVerify)
	assert.True(t, opts.Tags)
	assert.True(t, opts.FollowTags)
}

func TestBuildPushOpts_PartialSwitches(t *testing.T) {
	result := popup.Result{
		Switches: map[string]bool{
			"force-with-lease": true,
			"dry-run":          true,
		},
		Options: map[string]string{},
	}

	opts := buildPushOpts(result)
	assert.True(t, opts.ForceWithLease)
	assert.False(t, opts.Force)
	assert.True(t, opts.DryRun)
	assert.False(t, opts.SetUpstream)
}

func TestResolvePushTarget_PushRemote(t *testing.T) {
	head := HeadState{
		Branch:     "main",
		PushRemote: "origin",
	}
	remote, branch := resolvePushTarget("p", head)
	assert.Equal(t, "origin", remote)
	assert.Equal(t, "main", branch)
}

func TestResolvePushTarget_Upstream(t *testing.T) {
	head := HeadState{
		Branch:         "feature",
		UpstreamRemote: "upstream",
		UpstreamBranch: "upstream/feature",
	}
	remote, branch := resolvePushTarget("u", head)
	assert.Equal(t, "upstream", remote)
	assert.Equal(t, "feature", branch)
}

func TestResolvePushTarget_AllTags(t *testing.T) {
	head := HeadState{
		PushRemote: "origin",
	}
	remote, branch := resolvePushTarget("t", head)
	assert.Equal(t, "origin", remote)
	assert.Empty(t, branch)
}

func TestResolvePushTarget_Matching(t *testing.T) {
	head := HeadState{
		PushRemote: "origin",
	}
	remote, _ := resolvePushTarget("m", head)
	assert.Equal(t, "origin", remote)
}

func TestPushCmd_BuildsCorrectOpts(t *testing.T) {
	// Verify the pushCmd function returns a non-nil command
	opts := git.PushOpts{
		Remote: "origin",
		Branch: "main",
		DryRun: true,
	}
	cmd := pushCmd(nil, opts) // nil repo - we just test it returns a Cmd
	assert.NotNil(t, cmd)
}
