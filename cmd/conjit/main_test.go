package main

import (
	"os"
	"testing"
)

func TestOpenTTY(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping TTY test in short mode")
	}

	tty, err := openTTY()
	if err != nil {
		t.Fatalf("openTTY() returned error: %v", err)
	}
	defer tty.Close() //nolint:errcheck // best-effort cleanup
	// Verify the file descriptor is valid and read-write
	info, err := tty.Stat()
	if err != nil {
		t.Fatalf("stat on tty fd failed: %v", err)
	}

	if info.Mode()&os.ModeCharDevice == 0 {
		t.Error("expected tty to be a character device")
	}
}
