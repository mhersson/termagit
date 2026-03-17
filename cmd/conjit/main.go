package main

import (
	"errors"
	"flag"
	"fmt"
	"os"

	tea "github.com/charmbracelet/bubbletea"

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
	p := tea.NewProgram(model, tea.WithAltScreen())
	_, err = p.Run()
	return err
}
