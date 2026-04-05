package main

import (
	"os"
	"syscall"
	"testing"
	"time"

	tea "github.com/charmbracelet/bubbletea"
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

// stopRelay closes sigCh and waits for the relay goroutine to finish.
// This helper ensures the close happens before the wait.
func stopRelay(sigCh chan os.Signal, done <-chan struct{}) {
	close(sigCh)
	<-done
}

// TestStartSignalRelay_SIGINTSendsCtrlC verifies that receiving SIGINT
// causes startSignalRelay to invoke sendFn with a KeyCtrlC message.
func TestStartSignalRelay_SIGINTSendsCtrlC(t *testing.T) {
	sentMsgs := make(chan tea.Msg, 10)
	sendFn := func(msg tea.Msg) {
		sentMsgs <- msg
	}
	killCalled := make(chan struct{}, 1)
	killFn := func() {
		killCalled <- struct{}{}
	}

	sigCh := make(chan os.Signal, 1)
	done := startSignalRelay(sigCh, sendFn, killFn)
	defer stopRelay(sigCh, done)

	// Send SIGINT to the relay channel
	sigCh <- syscall.SIGINT

	select {
	case msg := <-sentMsgs:
		keyMsg, ok := msg.(tea.KeyMsg)
		if !ok {
			t.Fatalf("expected tea.KeyMsg, got %T", msg)
		}
		if keyMsg.Type != tea.KeyCtrlC {
			t.Errorf("expected KeyCtrlC (%d), got %d", tea.KeyCtrlC, keyMsg.Type)
		}
	case <-time.After(time.Second):
		t.Fatal("timed out waiting for message from signal relay")
	}

	// killFn should NOT have been called for SIGINT
	select {
	case <-killCalled:
		t.Error("killFn should not be called on SIGINT")
	default:
		// expected: no kill
	}
}

// TestStartSignalRelay_SIGTERMCallsKill verifies that receiving SIGTERM
// causes startSignalRelay to invoke killFn.
func TestStartSignalRelay_SIGTERMCallsKill(t *testing.T) {
	sentMsgs := make(chan tea.Msg, 10)
	sendFn := func(msg tea.Msg) {
		sentMsgs <- msg
	}
	killCalled := make(chan struct{}, 1)
	killFn := func() {
		killCalled <- struct{}{}
	}

	sigCh := make(chan os.Signal, 1)
	done := startSignalRelay(sigCh, sendFn, killFn)
	defer stopRelay(sigCh, done)

	// Send SIGTERM to the relay channel
	sigCh <- syscall.SIGTERM

	select {
	case <-killCalled:
		// expected
	case <-time.After(time.Second):
		t.Fatal("timed out waiting for killFn to be called on SIGTERM")
	}

	// sendFn should NOT have been called for SIGTERM
	select {
	case msg := <-sentMsgs:
		t.Errorf("sendFn should not be called on SIGTERM, got %T", msg)
	default:
		// expected: no send
	}
}

// TestStartSignalRelay_ChannelCloseStopsGoroutine verifies that closing
// the signal channel causes the relay goroutine to exit cleanly.
func TestStartSignalRelay_ChannelCloseStopsGoroutine(t *testing.T) {
	sendFn := func(msg tea.Msg) {}
	killFn := func() {}

	sigCh := make(chan os.Signal, 1)
	done := startSignalRelay(sigCh, sendFn, killFn)

	close(sigCh)

	select {
	case <-done:
		// goroutine exited cleanly
	case <-time.After(time.Second):
		t.Fatal("timed out waiting for relay goroutine to exit after channel close")
	}
}
