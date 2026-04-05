package main

import (
	"errors"
	"flag"
	"fmt"
	"io"
	"os"
	"os/signal"
	"syscall"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"

	"github.com/mhersson/termagit/internal/app"
	"github.com/mhersson/termagit/internal/cmdlog"
	"github.com/mhersson/termagit/internal/config"
	"github.com/mhersson/termagit/internal/git"
	"github.com/mhersson/termagit/internal/theme"
)

var version = "dev"

func main() {
	if err := run(); err != nil {
		fmt.Fprintf(os.Stderr, "termagit: %v\n", err)
		os.Exit(1)
	}
}

func run() error {
	// Parse flags
	var (
		flagPath    = flag.String("path", "", "path to git repository")
		flagTheme   = flag.String("theme", "", "color theme (overrides config)")
		flagVersion = flag.Bool("version", false, "print version and exit")
	)
	flag.Parse()

	if *flagVersion {
		fmt.Println("termagit", version)
		return nil
	}

	// Load config
	cfg, err := config.Load()
	if err != nil {
		return fmt.Errorf("load config: %w", err)
	}

	// Load external themes
	themesDir, err := config.ThemesDir()
	if err == nil {
		_ = theme.LoadDir(themesDir)
	}

	// Open /dev/tty so the TUI renders to the actual terminal even when
	// stdin/stdout are captured by a parent process (e.g. Helix editor).
	// This must happen before theme compilation so lipgloss detects
	// color capabilities from the real terminal, not a captured stdout.
	tty, err := openTTY()
	if err != nil {
		return fmt.Errorf("open terminal: %w", err)
	}
	defer tty.Close() //nolint:errcheck // best-effort cleanup

	// Disable any mouse tracking that a parent process (e.g. Helix) may have
	// enabled. Without this, residual SGR mouse events in the TTY read buffer
	// can be misidentified by Bubble Tea as the start of an escape sequence,
	// causing ESC key disambiguation failures.
	if err := prepareTerminal(tty); err != nil {
		return fmt.Errorf("prepare terminal: %w", err)
	}

	lipgloss.SetDefaultRenderer(lipgloss.NewRenderer(tty))

	// Resolve theme: flag > config > fallback
	themeName := cfg.Theme
	if *flagTheme != "" {
		themeName = *flagTheme
	}

	activeTheme, ok := theme.Get(themeName)
	if !ok {
		activeTheme = theme.Fallback()
	}

	tokens := theme.Compile(activeTheme.Raw())

	// Initialize command logger
	logPath, err := config.LogFile()
	if err != nil {
		return fmt.Errorf("get log path: %w", err)
	}

	maxSize, err := config.ParseMaxSize(cfg.Log.MaxSize)
	if err != nil {
		maxSize = 10 * 1024 * 1024 // 10MB default
	}

	logger, err := cmdlog.New(logPath, maxSize, cfg.Log.Keep)
	if err != nil {
		// Non-fatal: continue without logging
		logger = nil
	}
	defer func() {
		if logger != nil {
			_ = logger.Close()
		}
	}()

	// Discover git repository
	repoPath := *flagPath
	if repoPath == "" {
		repoPath, err = os.Getwd()
		if err != nil {
			return fmt.Errorf("get current directory: %w", err)
		}
	}

	repo, err := git.Open(repoPath, logger)
	if err != nil {
		if errors.Is(err, git.ErrNotARepo) {
			return fmt.Errorf("not a git repository: %s", repoPath)
		}
		return fmt.Errorf("open repository: %w", err)
	}

	// Run the TUI
	model := app.New(repo, cfg, tokens, logger)
	p := tea.NewProgram(model,
		tea.WithAltScreen(),
		tea.WithInput(tty),
		tea.WithOutput(tty),
		tea.WithoutSignalHandler(),
	)

	// Relay OS signals into the Bubble Tea event loop so that Ctrl-C
	// (SIGINT) is handled as a key event rather than terminating the
	// process directly. This is required when termagit runs as a
	// subprocess inside an editor such as Helix on Linux, where the
	// editor's controlling TTY causes Ctrl-C to generate SIGINT instead
	// of a raw key byte.
	sigCh := make(chan os.Signal, 1)
	signal.Notify(sigCh, os.Interrupt, syscall.SIGTERM)
	relayDone := startSignalRelay(sigCh, p.Send, p.Kill)
	defer func() {
		signal.Stop(sigCh)
		close(sigCh)
		<-relayDone
	}()

	// Start file watcher with program.Send as the callback
	model.StartWatcher(p.Send)

	_, err = p.Run()

	// Defensive: reset terminal modes that might linger after abnormal exit.
	// These are idempotent and safe even when the terminal is already clean.
	_, _ = tty.WriteString("\033[?1l\033[?25h\033[?2004l")

	return err
}

// openTTY opens the controlling terminal for direct read-write access.
func openTTY() (*os.File, error) {
	return os.OpenFile("/dev/tty", os.O_RDWR, 0)
}

// prepareTerminal writes mouse-disable escape sequences to w so that any
// mouse tracking enabled by a parent process (e.g. Helix) is turned off
// before Bubble Tea starts reading input. This prevents residual SGR mouse
// events from being delivered in the same Read() buffer as the user's ESC
// keypress, which would cause Bubble Tea's detectOneMsg to misidentify ESC
// as the start of an escape sequence.
func prepareTerminal(w io.Writer) error {
	// Each sequence disables a different mouse-tracking mode:
	//   ?1000l — basic mouse tracking (X10 compatible)
	//   ?1002l — button event tracking
	//   ?1003l — all-motion tracking
	//   ?1006l — SGR extended mouse mode
	//   ?1015l — urxvt extended mouse mode
	_, err := io.WriteString(w, "\x1b[?1000l\x1b[?1002l\x1b[?1003l\x1b[?1006l\x1b[?1015l")
	return err
}

// startSignalRelay starts a goroutine that reads from sigCh and relays each
// signal to the Bubble Tea program via the provided callbacks:
//   - os.Interrupt (SIGINT) is relayed as tea.KeyMsg{Type: tea.KeyCtrlC} via sendFn,
//     so the two-Ctrl-C commit sequence works when running inside an editor.
//   - syscall.SIGTERM is relayed as killFn() for a clean shutdown.
//
// The goroutine exits when sigCh is closed. The returned channel is closed
// when the goroutine has exited, allowing callers to wait for it.
func startSignalRelay(sigCh <-chan os.Signal, sendFn func(tea.Msg), killFn func()) <-chan struct{} {
	done := make(chan struct{})
	go func() {
		defer close(done)
		for sig := range sigCh {
			switch sig {
			case os.Interrupt:
				sendFn(tea.KeyMsg{Type: tea.KeyCtrlC})
			case syscall.SIGTERM:
				killFn()
			}
		}
	}()
	return done
}
