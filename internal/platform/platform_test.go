package platform

import (
	"encoding/base64"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestLinuxClipboardCmd_WaylandDisplay_UsesWlCopy(t *testing.T) {
	t.Setenv("WAYLAND_DISPLAY", "wayland-0")
	t.Setenv("XDG_SESSION_TYPE", "")

	cmd := linuxClipboardCmd()

	assert.Equal(t, "wl-copy", cmd.Args[0])
}

func TestLinuxClipboardCmd_XDGSessionTypeWayland_UsesWlCopy(t *testing.T) {
	t.Setenv("WAYLAND_DISPLAY", "")
	t.Setenv("XDG_SESSION_TYPE", "wayland")

	cmd := linuxClipboardCmd()

	assert.Equal(t, "wl-copy", cmd.Args[0])
}

func TestLinuxClipboardCmd_NoWaylandEnv_UsesXclip(t *testing.T) {
	t.Setenv("WAYLAND_DISPLAY", "")
	t.Setenv("XDG_SESSION_TYPE", "")

	cmd := linuxClipboardCmd()

	assert.Equal(t, "xclip", cmd.Args[0])
	assert.Equal(t, []string{"xclip", "-selection", "clipboard"}, cmd.Args)
}

func TestLinuxClipboardCmd_XDGSessionTypeX11_UsesXclip(t *testing.T) {
	t.Setenv("WAYLAND_DISPLAY", "")
	t.Setenv("XDG_SESSION_TYPE", "x11")

	cmd := linuxClipboardCmd()

	assert.Equal(t, "xclip", cmd.Args[0])
}

func TestOSC52Sequence_PlainTerminal(t *testing.T) {
	t.Setenv("TMUX", "")
	t.Setenv("TERM", "xterm-256color")

	seq := osc52Sequence("hello")

	encoded := base64.StdEncoding.EncodeToString([]byte("hello"))
	assert.Equal(t, "\x1b]52;c;"+encoded+"\x07", seq)
}

func TestOSC52Sequence_Tmux(t *testing.T) {
	t.Setenv("TMUX", "/tmp/tmux-1000/default,12345,0")
	t.Setenv("TERM", "screen-256color")

	seq := osc52Sequence("hello")

	encoded := base64.StdEncoding.EncodeToString([]byte("hello"))
	assert.Equal(t, "\x1bPtmux;\x1b\x1b]52;c;"+encoded+"\x07\x1b\\", seq)
}

func TestOSC52Sequence_Screen(t *testing.T) {
	t.Setenv("TMUX", "")
	t.Setenv("TERM", "screen")

	seq := osc52Sequence("hello")

	encoded := base64.StdEncoding.EncodeToString([]byte("hello"))
	assert.Equal(t, "\x1bP\x1b]52;c;"+encoded+"\x07\x1b\\", seq)
}
