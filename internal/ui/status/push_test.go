package status

import (
	"testing"

	"github.com/mhersson/termagit/internal/git"
	"github.com/mhersson/termagit/internal/ui/popup"
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
	remote, branch, setUpstream := resolvePushTarget("p", head, nil)
	assert.Equal(t, "origin", remote)
	assert.Equal(t, "main", branch)
	assert.False(t, setUpstream)
}

func TestResolvePushTarget_Upstream(t *testing.T) {
	head := HeadState{
		Branch:         "feature",
		UpstreamRemote: "upstream",
		UpstreamBranch: "upstream/feature",
	}
	remote, branch, setUpstream := resolvePushTarget("u", head, nil)
	assert.Equal(t, "upstream", remote)
	assert.Equal(t, "feature", branch)
	assert.False(t, setUpstream)
}

func TestResolvePushTarget_AllTags(t *testing.T) {
	head := HeadState{
		PushRemote: "origin",
	}
	remote, branch, _ := resolvePushTarget("t", head, nil)
	assert.Equal(t, "origin", remote)
	assert.Empty(t, branch)
}

func TestResolvePushTarget_Matching(t *testing.T) {
	head := HeadState{
		PushRemote: "origin",
	}
	remote, _, _ := resolvePushTarget("m", head, nil)
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

func TestApplyPushActionOverrides_AllTags(t *testing.T) {
	// Action "t" (all tags) should set Tags=true regardless of switch state
	result := popup.Result{
		Action:   "t",
		Switches: map[string]bool{}, // Tags switch is NOT toggled
		Options:  map[string]string{},
	}
	opts := buildPushOpts(result)
	assert.False(t, opts.Tags, "Tags should be false from switches before override")

	applyPushActionOverrides(result.Action, &opts)
	assert.True(t, opts.Tags, "Tags should be true after applying action 't' override")
}

func TestApplyPushActionOverrides_OtherActions(t *testing.T) {
	// Other actions should not modify Tags
	for _, action := range []string{"p", "u", "e", "o"} {
		result := popup.Result{
			Action:   action,
			Switches: map[string]bool{},
			Options:  map[string]string{},
		}
		opts := buildPushOpts(result)
		applyPushActionOverrides(result.Action, &opts)
		assert.False(t, opts.Tags, "Action %q should not set Tags", action)
	}
}
