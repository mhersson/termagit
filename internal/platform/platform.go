package platform

import (
	"encoding/base64"
	"fmt"
	"os"
	"os/exec"
	"runtime"
	"strings"
)

// CopyToClipboard copies text to the system clipboard.
func CopyToClipboard(text string) error {
	var cmd *exec.Cmd
	switch runtime.GOOS {
	case "darwin":
		cmd = exec.Command("pbcopy")
	case "linux":
		cmd = linuxClipboardCmd()
	case "windows":
		cmd = exec.Command("clip")
	default:
		return copyViaOSC52(text)
	}
	cmd.Stdin = strings.NewReader(text)
	if err := cmd.Run(); err != nil {
		// Native clipboard failed (e.g. no display server over SSH).
		// Fall back to OSC 52 terminal escape sequence.
		return copyViaOSC52(text)
	}
	return nil
}

func linuxClipboardCmd() *exec.Cmd {
	if os.Getenv("WAYLAND_DISPLAY") != "" || os.Getenv("XDG_SESSION_TYPE") == "wayland" {
		return exec.Command("wl-copy")
	}
	return exec.Command("xclip", "-selection", "clipboard")
}

func copyViaOSC52(text string) error {
	_, err := os.Stderr.Write([]byte(osc52Sequence(text)))
	return err
}

func osc52Sequence(text string) string {
	encoded := base64.StdEncoding.EncodeToString([]byte(text))
	if os.Getenv("TMUX") != "" {
		return "\x1bPtmux;\x1b\x1b]52;c;" + encoded + "\x07\x1b\\"
	}
	if strings.HasPrefix(os.Getenv("TERM"), "screen") {
		return "\x1bP\x1b]52;c;" + encoded + "\x07\x1b\\"
	}
	return "\x1b]52;c;" + encoded + "\x07"
}

// Open opens a file, directory, or URL with the system default handler.
func Open(target string) error {
	var cmd *exec.Cmd
	switch runtime.GOOS {
	case "darwin":
		cmd = exec.Command("open", target)
	case "linux":
		cmd = exec.Command("xdg-open", target)
	case "windows":
		cmd = exec.Command("rundll32", "url.dll,FileProtocolHandler", target)
	default:
		return fmt.Errorf("open not supported on %s", runtime.GOOS)
	}
	if err := cmd.Start(); err != nil {
		return err
	}
	go func() { _ = cmd.Wait() }()
	return nil
}
