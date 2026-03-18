package main

import (
	"errors"
	"flag"
	"fmt"
	"os"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"

	"github.com/mhersson/conjit/internal/app"
	"github.com/mhersson/conjit/internal/cmdlog"
	"github.com/mhersson/conjit/internal/config"
	"github.com/mhersson/conjit/internal/git"
	"github.com/mhersson/conjit/internal/theme"
)

var version = "dev"

func main() {
	if err := run(); err != nil {
		fmt.Fprintf(os.Stderr, "conjit: %v\n", err)
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
		fmt.Println("conjit", version)
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
	defer tty.Close()

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
	)
	_, err = p.Run()
	return err
}

// openTTY opens the controlling terminal for direct read-write access.
func openTTY() (*os.File, error) {
	return os.OpenFile("/dev/tty", os.O_RDWR, 0)
}
