package dialog

import (
	"fmt"
	"sort"
	"strings"

	"charm.land/bubbles/v2/key"
	"charm.land/bubbles/v2/textinput"
	tea "charm.land/bubbletea/v2"
	"charm.land/lipgloss/v2"

	"github.com/docker/cagent/pkg/runtime"
	"github.com/docker/cagent/pkg/tui/core"
	"github.com/docker/cagent/pkg/tui/core/layout"
	"github.com/docker/cagent/pkg/tui/messages"
	"github.com/docker/cagent/pkg/tui/styles"
)

// SupportedProviders lists the valid provider names that can be used in custom model specs.
// This includes both core providers and aliases.
var SupportedProviders = []string{
	// Core providers
	"openai", "anthropic", "google", "dmr",
	// Aliases (these map to core providers with different defaults)
	"requesty", "azure", "xai", "nebius", "mistral", "ollama",
}

// modelPickerDialog is a dialog for selecting a model for the current agent.
type modelPickerDialog struct {
	BaseDialog
	textInput textinput.Model
	models    []runtime.ModelChoice
	filtered  []runtime.ModelChoice
	selected  int
	offset    int
	keyMap    commandPaletteKeyMap
	errMsg    string // validation error message
}

// NewModelPickerDialog creates a new model picker dialog.
func NewModelPickerDialog(models []runtime.ModelChoice) Dialog {
	ti := textinput.New()
	ti.Placeholder = "Type to search or enter custom model (provider/model)…"
	ti.Focus()
	ti.CharLimit = 100
	ti.SetWidth(50)

	// Sort models: default first, then config models alphabetically, then custom models
	sortedModels := make([]runtime.ModelChoice, len(models))
	copy(sortedModels, models)
	sort.Slice(sortedModels, func(i, j int) bool {
		// Custom models always come last
		if sortedModels[i].IsCustom != sortedModels[j].IsCustom {
			return !sortedModels[i].IsCustom
		}
		// Within each group: default first, then alphabetically
		if sortedModels[i].IsDefault {
			return true
		}
		if sortedModels[j].IsDefault {
			return false
		}
		return sortedModels[i].Name < sortedModels[j].Name
	})

	d := &modelPickerDialog{
		textInput: ti,
		models:    sortedModels,
		keyMap:    defaultCommandPaletteKeyMap(),
	}
	d.filterModels()
	return d
}

func (d *modelPickerDialog) Init() tea.Cmd {
	return textinput.Blink
}

func (d *modelPickerDialog) Update(msg tea.Msg) (layout.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		cmd := d.SetSize(msg.Width, msg.Height)
		return d, cmd

	case tea.KeyPressMsg:
		if cmd := HandleQuit(msg); cmd != nil {
			return d, cmd
		}

		switch {
		case key.Matches(msg, d.keyMap.Escape):
			return d, core.CmdHandler(CloseDialogMsg{})

		case key.Matches(msg, d.keyMap.Up):
			if d.selected > 0 {
				d.selected--
			}
			return d, nil

		case key.Matches(msg, d.keyMap.Down):
			if d.selected < len(d.filtered)-1 {
				d.selected++
			}
			return d, nil

		case key.Matches(msg, d.keyMap.PageUp):
			d.selected -= d.pageSize()
			if d.selected < 0 {
				d.selected = 0
			}
			return d, nil

		case key.Matches(msg, d.keyMap.PageDown):
			d.selected += d.pageSize()
			if d.selected >= len(d.filtered) {
				d.selected = max(0, len(d.filtered)-1)
			}
			return d, nil

		case key.Matches(msg, d.keyMap.Enter):
			cmd := d.handleSelection()
			return d, cmd

		default:
			var cmd tea.Cmd
			d.textInput, cmd = d.textInput.Update(msg)
			d.filterModels()
			d.errMsg = "" // Clear error when user types
			return d, cmd
		}
	}

	return d, nil
}

func (d *modelPickerDialog) handleSelection() tea.Cmd {
	query := strings.TrimSpace(d.textInput.Value())

	// If user typed something that looks like a custom model (contains /), validate and use it
	if strings.Contains(query, "/") {
		if err := validateCustomModelSpec(query); err != nil {
			d.errMsg = err.Error()
			return nil
		}
		return tea.Sequence(
			core.CmdHandler(CloseDialogMsg{}),
			core.CmdHandler(messages.ChangeModelMsg{ModelRef: query}),
		)
	}

	// Otherwise, use the selected item from the filtered list
	if d.selected >= 0 && d.selected < len(d.filtered) {
		selected := d.filtered[d.selected]
		// If selecting the default model, send empty ref to clear the override
		modelRef := selected.Ref
		if selected.IsDefault {
			modelRef = ""
		}
		return tea.Sequence(
			core.CmdHandler(CloseDialogMsg{}),
			core.CmdHandler(messages.ChangeModelMsg{ModelRef: modelRef}),
		)
	}

	return nil
}

// validateCustomModelSpec validates a custom model specification entered by the user.
// It checks that each provider/model pair is properly formatted and uses a supported provider.
func validateCustomModelSpec(spec string) error {
	spec = strings.TrimSpace(spec)
	if spec == "" {
		return nil
	}

	// Handle alloy specs (comma-separated)
	parts := strings.Split(spec, ",")
	for _, part := range parts {
		part = strings.TrimSpace(part)
		if part == "" {
			continue
		}

		providerName, modelName, ok := strings.Cut(part, "/")
		if !ok {
			return fmt.Errorf("invalid format: expected 'provider/model'")
		}

		providerName = strings.TrimSpace(providerName)
		modelName = strings.TrimSpace(modelName)

		if providerName == "" {
			return fmt.Errorf("provider name cannot be empty (got '/%s')", modelName)
		}
		if modelName == "" {
			return fmt.Errorf("model name cannot be empty (got '%s/')", providerName)
		}

		if !isValidProvider(providerName) {
			return fmt.Errorf("unknown provider '%s'. Supported: %s",
				providerName, strings.Join(SupportedProviders, ", "))
		}
	}

	return nil
}

// isValidProvider checks if the provider name is in the list of supported providers.
func isValidProvider(name string) bool {
	for _, p := range SupportedProviders {
		if strings.EqualFold(p, name) {
			return true
		}
	}
	return false
}

func (d *modelPickerDialog) filterModels() {
	query := strings.ToLower(strings.TrimSpace(d.textInput.Value()))

	// If query contains "/", show "Custom" option as well as matches
	isCustomQuery := strings.Contains(query, "/")

	d.filtered = nil
	for _, model := range d.models {
		if query == "" {
			d.filtered = append(d.filtered, model)
			continue
		}

		// Match against name, provider, and model
		searchText := strings.ToLower(model.Name + " " + model.Provider + " " + model.Model)
		if strings.Contains(searchText, query) {
			d.filtered = append(d.filtered, model)
		}
	}

	// If query looks like a custom model spec and we have no exact match, show it as an option
	if isCustomQuery && len(d.filtered) == 0 {
		d.filtered = append(d.filtered, runtime.ModelChoice{
			Name: "Custom: " + query,
			Ref:  query,
		})
	}

	if d.selected >= len(d.filtered) {
		d.selected = max(0, len(d.filtered)-1)
	}
	d.offset = 0
}

func (d *modelPickerDialog) dialogSize() (dialogWidth, maxHeight, contentWidth int) {
	dialogWidth = max(min(d.Width()*80/100, 70), 50)
	maxHeight = min(d.Height()*70/100, 25)
	contentWidth = dialogWidth - 6
	return dialogWidth, maxHeight, contentWidth
}

func (d *modelPickerDialog) View() string {
	dialogWidth, maxHeight, contentWidth := d.dialogSize()

	d.textInput.SetWidth(contentWidth)

	var modelLines []string
	maxItems := maxHeight - 8

	// Adjust offset to keep selected item visible
	if d.selected < d.offset {
		d.offset = d.selected
	} else if d.selected >= d.offset+maxItems {
		d.offset = d.selected - maxItems + 1
	}

	// Track if we've shown the custom models separator
	customSeparatorShown := false

	// Render visible items based on offset
	visibleEnd := min(d.offset+maxItems, len(d.filtered))
	for i := d.offset; i < visibleEnd; i++ {
		model := d.filtered[i]

		// Add separator before first custom model
		if model.IsCustom && !customSeparatorShown {
			// Check if there are any non-custom models before this
			hasConfigModels := false
			for j := range i {
				if !d.filtered[j].IsCustom {
					hasConfigModels = true
					break
				}
			}
			if hasConfigModels || i > d.offset {
				separatorLine := styles.MutedStyle.Render("── Custom models " + strings.Repeat("─", max(0, contentWidth-19)))
				modelLines = append(modelLines, separatorLine)
			}
			customSeparatorShown = true
		}

		modelLines = append(modelLines, d.renderModel(model, i == d.selected, contentWidth))
	}

	// Show indicator if there are more items
	if visibleEnd < len(d.filtered) {
		modelLines = append(modelLines, styles.MutedStyle.Render("  …and more"))
	}

	if len(d.filtered) == 0 {
		modelLines = append(modelLines, "", styles.DialogContentStyle.
			Italic(true).
			Align(lipgloss.Center).
			Width(contentWidth).
			Render("No models found"))
	}

	contentBuilder := NewContent(contentWidth).
		AddTitle("Select Model").
		AddSpace().
		AddContent(d.textInput.View())

	// Show error message if present
	if d.errMsg != "" {
		errorStyle := lipgloss.NewStyle().Foreground(lipgloss.Color("#FF6B6B"))
		contentBuilder.AddContent(errorStyle.Render("⚠ " + d.errMsg))
	}

	content := contentBuilder.
		AddSeparator().
		AddContent(strings.Join(modelLines, "\n")).
		AddSpace().
		AddHelpKeys("↑/↓", "navigate", "enter", "select", "esc", "cancel").
		Build()

	return styles.DialogStyle.Width(dialogWidth).Render(content)
}

func (d *modelPickerDialog) pageSize() int {
	_, maxHeight, _ := d.dialogSize()
	return max(1, maxHeight-8)
}

func (d *modelPickerDialog) renderModel(model runtime.ModelChoice, selected bool, _ int) string {
	nameStyle, descStyle := styles.PaletteUnselectedActionStyle, styles.PaletteUnselectedDescStyle
	alloyBadgeStyle, defaultBadgeStyle, currentBadgeStyle := styles.BadgeAlloyStyle, styles.BadgeDefaultStyle, styles.BadgeCurrentStyle
	if selected {
		nameStyle, descStyle = styles.PaletteSelectedActionStyle, styles.PaletteSelectedDescStyle
		// Keep badge colors visible on selection background
		alloyBadgeStyle = alloyBadgeStyle.Background(styles.MobyBlue)
		defaultBadgeStyle = defaultBadgeStyle.Background(styles.MobyBlue)
		currentBadgeStyle = currentBadgeStyle.Background(styles.MobyBlue)
	}

	// Check if this is an alloy model (no provider but has comma-separated models)
	isAlloy := model.Provider == "" && strings.Contains(model.Model, ",")

	// Build the name with colored badges
	var nameParts []string
	nameParts = append(nameParts, nameStyle.Render(model.Name))
	if isAlloy {
		nameParts = append(nameParts, alloyBadgeStyle.Render(" (alloy)"))
	}
	if model.IsCurrent {
		nameParts = append(nameParts, currentBadgeStyle.Render(" (current)"))
	} else if model.IsDefault {
		nameParts = append(nameParts, defaultBadgeStyle.Render(" (default)"))
	}
	name := strings.Join(nameParts, "")

	// Build description (skip for custom models where name already is provider/model)
	var desc string
	switch {
	case model.IsCustom:
		// Custom models: name already is provider/model, no need to repeat
	case model.Provider != "" && model.Model != "":
		desc = model.Provider + "/" + model.Model
	case isAlloy:
		// Alloy model: show the constituent models
		desc = model.Model
	case model.Ref != "" && !strings.Contains(model.Name, model.Ref):
		desc = model.Ref
	}

	if desc != "" {
		return name + descStyle.Render(" • "+desc)
	}
	return name
}

func (d *modelPickerDialog) Position() (row, col int) {
	dialogWidth, maxHeight, _ := d.dialogSize()
	return CenterPosition(d.Width(), d.Height(), dialogWidth, maxHeight)
}
