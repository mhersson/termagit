package popup

import (
	"strings"
	"unicode/utf8"

	"github.com/charmbracelet/bubbles/textinput"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/mhersson/conjit/internal/theme"
)

// Switch represents a toggleable boolean flag in a popup.
type Switch struct {
	Key          string
	Label        string // CLI flag name (e.g., "all", "verbose")
	Description  string
	Enabled      bool
	Persisted    bool     // whether to persist across sessions (default true)
	Incompatible []string // labels of switches that are mutually exclusive
}

// Option represents a key=value setting in a popup.
type Option struct {
	Key         string
	Label       string // CLI flag name
	Description string
	Value       string
	Persisted   bool // whether to persist across sessions (default true)
}

// Config represents a git config item displayed in a popup.
type Config struct {
	Key         string
	Label       string // config key (e.g., "branch.main.description")
	Description string
	Value       string
}

// Action represents an executable action in a popup.
type Action struct {
	Key      string
	Label    string
	Disabled bool // greyed out, not executable
	Spacer   bool // if true, this is just a visual spacer
}

// ActionGroup is a named group of actions.
type ActionGroup struct {
	Title   string
	Actions []Action
}

// Result holds the outcome when a popup closes.
type Result struct {
	Action   string            // empty if closed without action
	Switches map[string]bool   // label -> enabled
	Options  map[string]string // label -> value
	Config   map[string]string // label -> value
}

// Popup is the base model for all popups.
type Popup struct {
	title   string
	tokens  theme.Tokens
	width   int
	height  int

	config   []Config
	switches []Switch
	options  []Option
	groups   []ActionGroup

	// incompatible tracks mutually exclusive switches by label
	incompatible map[string][]string

	// pending key for two-key sequences (e.g., "-a")
	pendingKey string

	// option value editing
	editingOption int // index into options, -1 when not editing
	optionInput   textinput.Model

	done   bool
	result Result
}

// New creates a new popup with the given title.
func New(title string, tokens theme.Tokens) Popup {
	return Popup{
		title:         title,
		tokens:        tokens,
		incompatible:  make(map[string][]string),
		editingOption: -1,
		result: Result{
			Switches: make(map[string]bool),
			Options:  make(map[string]string),
			Config:   make(map[string]string),
		},
	}
}

// AddSwitch adds a switch to the popup.
func (p *Popup) AddSwitch(key, label, description string, enabled bool) {
	p.switches = append(p.switches, Switch{
		Key:         key,
		Label:       label,
		Description: description,
		Enabled:     enabled,
		Persisted:   true,
	})
}

// AddSwitchNonPersisted adds a switch that won't be persisted across sessions.
func (p *Popup) AddSwitchNonPersisted(key, label, description string, enabled bool) {
	p.switches = append(p.switches, Switch{
		Key:         key,
		Label:       label,
		Description: description,
		Enabled:     enabled,
		Persisted:   false,
	})
}

// SetIncompatible marks two switches as mutually exclusive.
func (p *Popup) SetIncompatible(key1, key2 string) {
	// Find labels by key
	var label1, label2 string
	for _, sw := range p.switches {
		if sw.Key == key1 {
			label1 = sw.Label
		}
		if sw.Key == key2 {
			label2 = sw.Label
		}
	}
	if label1 != "" && label2 != "" {
		p.incompatible[label1] = append(p.incompatible[label1], label2)
		p.incompatible[label2] = append(p.incompatible[label2], label1)
	}
}

// AddOption adds an option to the popup.
func (p *Popup) AddOption(key, label, description, value string) {
	p.options = append(p.options, Option{
		Key:         key,
		Label:       label,
		Description: description,
		Value:       value,
		Persisted:   true,
	})
}

// AddConfig adds a config item to the popup.
func (p *Popup) AddConfig(key, label, description, value string) {
	p.config = append(p.config, Config{
		Key:         key,
		Label:       label,
		Description: description,
		Value:       value,
	})
}

// AddActionGroup adds a group of actions to the popup.
func (p *Popup) AddActionGroup(title string, actions []Action) {
	p.groups = append(p.groups, ActionGroup{
		Title:   title,
		Actions: actions,
	})
}

// SetSize sets the popup dimensions.
func (p *Popup) SetSize(width, height int) {
	p.width = width
	p.height = height
}

// Done returns true if the popup should close.
func (p Popup) Done() bool {
	return p.done
}

// Result returns the popup outcome.
func (p Popup) Result() Result {
	return p.result
}

// Init implements tea.Model.
func (p Popup) Init() tea.Cmd {
	return nil
}

// Update implements tea.Model.
func (p Popup) Update(msg tea.Msg) (Popup, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.KeyMsg:
		return p.handleKey(msg)
	}
	return p, nil
}

func (p Popup) handleKey(msg tea.KeyMsg) (Popup, tea.Cmd) {
	keyStr := msg.String()

	// Handle option editing mode
	if p.editingOption >= 0 {
		return p.handleOptionInput(msg)
	}

	// Handle escape
	if msg.Type == tea.KeyEscape {
		p.done = true
		p.buildResult()
		return p, nil
	}

	// Handle pending key sequences
	if p.pendingKey == "-" {
		p.pendingKey = ""
		// Toggle switch with this key
		for i := range p.switches {
			if p.switches[i].Key == keyStr {
				p.switches[i].Enabled = !p.switches[i].Enabled
				// Handle incompatible switches
				if p.switches[i].Enabled {
					p.disableIncompatible(p.switches[i].Label)
				}
				return p, nil
			}
		}
		return p, nil
	}

	if p.pendingKey == "=" {
		p.pendingKey = ""
		for i := range p.options {
			if p.options[i].Key == keyStr {
				if p.options[i].Value != "" {
					// Toggle off: clear the value
					p.options[i].Value = ""
				} else {
					// Start editing
					p.editingOption = i
					ti := textinput.New()
					ti.Prompt = p.options[i].Label + "="
					ti.Focus()
					p.optionInput = ti
				}
				return p, nil
			}
		}
		return p, nil
	}

	// Handle special keys
	switch keyStr {
	case "q":
		p.done = true
		p.buildResult()
		return p, nil

	case "-":
		p.pendingKey = "-"
		return p, nil

	case "=":
		p.pendingKey = "="
		return p, nil
	}

	// Check for action keys
	for _, group := range p.groups {
		for _, action := range group.Actions {
			if action.Spacer || action.Disabled {
				continue
			}
			if action.Key == keyStr {
				p.done = true
				p.result.Action = action.Key
				p.buildResult()
				return p, nil
			}
		}
	}

	return p, nil
}

func (p Popup) handleOptionInput(msg tea.KeyMsg) (Popup, tea.Cmd) {
	switch msg.Type {
	case tea.KeyEnter:
		// Confirm: save value and stop editing
		p.options[p.editingOption].Value = p.optionInput.Value()
		p.editingOption = -1
		return p, nil
	case tea.KeyEscape:
		// Cancel: discard and stop editing
		p.editingOption = -1
		return p, nil
	default:
		var cmd tea.Cmd
		p.optionInput, cmd = p.optionInput.Update(msg)
		return p, cmd
	}
}

func (p *Popup) disableIncompatible(label string) {
	incompatibles := p.incompatible[label]
	for i := range p.switches {
		for _, inc := range incompatibles {
			if p.switches[i].Label == inc {
				p.switches[i].Enabled = false
			}
		}
	}
}

func (p *Popup) buildResult() {
	for _, sw := range p.switches {
		p.result.Switches[sw.Label] = sw.Enabled
	}
	for _, opt := range p.options {
		if opt.Value != "" {
			p.result.Options[opt.Label] = opt.Value
		}
	}
	for _, cfg := range p.config {
		if cfg.Value != "" {
			p.result.Config[cfg.Label] = cfg.Value
		}
	}
}

// View implements tea.Model.
func (p Popup) View() string {
	if p.width == 0 || p.height == 0 {
		return ""
	}

	var b strings.Builder

	// Top border (separates popup from main window)
	border := strings.Repeat("─", p.width)
	b.WriteString(p.tokens.PopupBorder.Render(border))
	b.WriteString("\n")

	// Config items (if any)
	if len(p.config) > 0 {
		for _, cfg := range p.config {
			line := p.renderConfigItem(cfg)
			b.WriteString(line)
			b.WriteString("\n")
		}
		b.WriteString("\n")
	}

	// Arguments section (switches + options together)
	if len(p.switches) > 0 || len(p.options) > 0 {
		b.WriteString(p.tokens.PopupSection.Render("Arguments"))
		b.WriteString("\n")

		// Render switches
		for _, sw := range p.switches {
			line := p.renderSwitch(sw)
			b.WriteString(line)
			b.WriteString("\n")
		}

		// Render options
		for _, opt := range p.options {
			line := p.renderOption(opt)
			b.WriteString(line)
			b.WriteString("\n")
		}
		b.WriteString("\n")
	}

	// Actions as grid (columns side by side)
	b.WriteString(p.renderActionsGrid())

	return b.String()
}

func (p Popup) renderSwitch(sw Switch) string {
	// Format: -key description (--flag)
	keyStyle := p.tokens.PopupKey
	key := keyStyle.Render("-" + sw.Key)

	desc := sw.Description

	var flagStyle lipgloss.Style
	if sw.Enabled {
		flagStyle = p.tokens.PopupSwitch
	} else {
		flagStyle = p.tokens.Dim
	}
	flag := flagStyle.Render("--" + sw.Label)

	return key + " " + desc + " (" + flag + ")"
}

func (p Popup) renderOption(opt Option) string {
	// Format: -key description (--option=value)
	// Key prefix is always "-" (same as switches)
	keyStyle := p.tokens.PopupKey
	key := keyStyle.Render("-" + opt.Key)

	desc := opt.Description

	var optStyle lipgloss.Style
	if opt.Value != "" {
		optStyle = p.tokens.PopupOption
	} else {
		optStyle = p.tokens.Dim
	}

	optText := "--" + opt.Label + "="
	if opt.Value != "" {
		optText += opt.Value
	}
	optFormatted := optStyle.Render(optText)

	return key + " " + desc + " (" + optFormatted + ")"
}

func (p Popup) renderConfigItem(cfg Config) string {
	keyStyle := p.tokens.PopupKey
	key := keyStyle.Render(cfg.Key)

	label := p.tokens.Normal.Render(cfg.Label)

	var value string
	if cfg.Value != "" {
		value = p.tokens.PopupOption.Render(cfg.Value)
	} else {
		value = p.tokens.Dim.Render("(unset)")
	}

	return "  " + key + " " + label + " " + value
}

// renderActionsGrid renders action groups as columns side by side.
func (p Popup) renderActionsGrid() string {
	if len(p.groups) == 0 {
		return ""
	}

	const gap = 3

	// Build columns: each group is a column
	// Column[i] = []string of lines (header + actions)
	type columnLine struct {
		text    string // raw text for width calculation
		styled  string // styled text for rendering
		isSpacer bool
	}

	var columns [][]columnLine
	var colWidths []int
	maxRows := 0

	for _, group := range p.groups {
		var col []columnLine
		maxWidth := 0

		// Header (section title)
		headerText := group.Title
		if len(headerText) > maxWidth {
			maxWidth = len(headerText)
		}
		col = append(col, columnLine{
			text:   headerText,
			styled: p.tokens.PopupSection.Render(headerText),
		})

		// Actions
		for _, action := range group.Actions {
			if action.Spacer {
				col = append(col, columnLine{text: "", styled: "", isSpacer: true})
				continue
			}

			// Calculate raw width for alignment
			rawText := action.Key + " " + action.Label
			if len(rawText) > maxWidth {
				maxWidth = len(rawText)
			}

			// Style key and label separately
			var styledLine string
			if action.Disabled {
				styledLine = p.tokens.Dim.Render(action.Key) + " " + p.tokens.Dim.Render(action.Label)
			} else {
				styledLine = p.tokens.PopupKey.Render(action.Key) + " " + action.Label
			}

			col = append(col, columnLine{text: rawText, styled: styledLine})
		}

		columns = append(columns, col)
		colWidths = append(colWidths, maxWidth)
		if len(col) > maxRows {
			maxRows = len(col)
		}
	}

	// Render grid row by row
	var b strings.Builder
	for row := 0; row < maxRows; row++ {
		var lineContent strings.Builder
		for colIdx, col := range columns {
			var cell columnLine
			if row < len(col) {
				cell = col[row]
			}

			// Pad to column width based on raw text length
			padding := colWidths[colIdx] - len(cell.text)
			if padding < 0 {
				padding = 0
			}

			// For first row (headers), build plain text for cursor rendering
			// For other rows, use styled content
			if row == 0 {
				lineContent.WriteString(cell.text)
			} else {
				lineContent.WriteString(cell.styled)
			}
			lineContent.WriteString(strings.Repeat(" ", padding))

			// Gap between columns (not after last)
			if colIdx < len(columns)-1 {
				lineContent.WriteString(strings.Repeat(" ", gap))
			}
		}

		// First row gets block cursor, others get newline directly
		if row == 0 {
			b.WriteString(renderWithBlockCursor(p.tokens, lineContent.String()))
		} else {
			b.WriteString(lineContent.String())
			b.WriteString("\n")
		}
	}

	return b.String()
}

// renderWithBlockCursor renders a line with a block cursor at position 0.
// Only the first character gets reverse video - no background on the rest.
func renderWithBlockCursor(tokens theme.Tokens, line string) string {
	if len(line) == 0 {
		return tokens.CursorBlock.Render(" ") + "\n"
	}

	// Get first rune (handles multi-byte UTF-8)
	firstRune, size := utf8.DecodeRuneInString(line)
	rest := line[size:]

	// First character: reverse video, rest: no special styling
	return tokens.CursorBlock.Render(string(firstRune)) + rest + "\n"
}
