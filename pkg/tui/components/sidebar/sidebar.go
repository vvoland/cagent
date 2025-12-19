package sidebar

import (
	"fmt"
	"log/slog"
	"os"
	"sort"
	"strings"

	tea "charm.land/bubbletea/v2"
	"charm.land/lipgloss/v2"

	"github.com/docker/cagent/pkg/paths"
	"github.com/docker/cagent/pkg/runtime"
	"github.com/docker/cagent/pkg/tools"
	"github.com/docker/cagent/pkg/tui/components/spinner"
	"github.com/docker/cagent/pkg/tui/components/tab"
	"github.com/docker/cagent/pkg/tui/components/tool/todotool"
	"github.com/docker/cagent/pkg/tui/core/layout"
	"github.com/docker/cagent/pkg/tui/styles"
)

type Mode int

const (
	ModeVertical Mode = iota
	ModeHorizontal
)

// Model represents a sidebar component
type Model interface {
	layout.Model
	layout.Sizeable

	SetTokenUsage(event *runtime.TokenUsageEvent)
	SetTodos(result *tools.ToolCallResult) error
	SetMode(mode Mode)
	SetAgentInfo(agentName, model, description string)
	SetTeamInfo(availableAgents []string)
	SetAgentSwitching(switching bool)
	SetToolsetInfo(availableTools int)
	SetYolo(yolo bool)
	GetSize() (width, height int)
}

// ragIndexingState tracks per-strategy indexing progress
type ragIndexingState struct {
	current int
	total   int
	spinner spinner.Spinner
}

// model implements Model
type model struct {
	width            int
	height           int
	sessionUsage     map[string]*runtime.Usage // sessionID -> latest usage snapshot
	sessionAgent     map[string]string         // sessionID -> agent name
	todoComp         *todotool.SidebarComponent
	mcpInit          bool
	yolo             bool
	ragIndexing      map[string]*ragIndexingState // strategy name -> indexing state
	spinner          spinner.Spinner
	mode             Mode
	sessionTitle     string
	currentAgent     string
	agentModel       string
	agentDescription string
	availableAgents  []string
	agentSwitching   bool
	availableTools   int
}

func New() Model {
	return &model{
		width:        20,
		height:       24,
		sessionUsage: make(map[string]*runtime.Usage),
		sessionAgent: make(map[string]string),
		todoComp:     todotool.NewSidebarComponent(),
		spinner:      spinner.New(spinner.ModeSpinnerOnly),
		sessionTitle: "New session",
		ragIndexing:  make(map[string]*ragIndexingState),
	}
}

func (m *model) Init() tea.Cmd {
	return nil
}

func (m *model) SetTokenUsage(event *runtime.TokenUsageEvent) {
	if event == nil || event.Usage == nil || event.SessionID == "" || event.AgentName == "" {
		return
	}

	// Store/replace by session ID (each event has cumulative totals for that session)
	usage := *event.Usage
	m.sessionUsage[event.SessionID] = &usage
	m.sessionAgent[event.SessionID] = event.AgentName
}

func (m *model) SetTodos(result *tools.ToolCallResult) error {
	return m.todoComp.SetTodos(result)
}

// SetAgentInfo sets the current agent information
func (m *model) SetAgentInfo(agentName, model, description string) {
	m.currentAgent = agentName
	m.agentModel = model
	m.agentDescription = description
}

// SetTeamInfo sets the available agents in the team
func (m *model) SetTeamInfo(availableAgents []string) {
	m.availableAgents = availableAgents
}

// SetAgentSwitching sets whether an agent switch is in progress
func (m *model) SetAgentSwitching(switching bool) {
	m.agentSwitching = switching
}

// SetToolsetInfo sets the number of available tools
func (m *model) SetToolsetInfo(availableTools int) {
	m.availableTools = availableTools
}

func (m *model) SetYolo(yolo bool) {
	m.yolo = yolo
}

// formatTokenCount formats a token count with K/M suffixes for readability
func formatTokenCount(count int64) string {
	if count >= 1000000 {
		return fmt.Sprintf("%.1fM", float64(count)/1000000)
	} else if count >= 1000 {
		return fmt.Sprintf("%.1fK", float64(count)/1000)
	}
	return fmt.Sprintf("%d", count)
}

func formatCost(cost float64) string {
	return fmt.Sprintf("%.2f", cost)
}

// contextPercent returns a context usage percentage string when a single session has a limit.
func (m *model) contextPercent() (string, bool) {
	if len(m.sessionUsage) != 1 {
		return "", false
	}
	for _, usage := range m.sessionUsage {
		if usage.ContextLimit > 0 {
			percent := (float64(usage.ContextLength) / float64(usage.ContextLimit)) * 100
			return fmt.Sprintf("%.0f%%", percent), true
		}
	}
	return "", false
}

// getCurrentWorkingDirectory returns the current working directory with home directory replaced by ~/
func getCurrentWorkingDirectory() string {
	pwd, err := os.Getwd()
	if err != nil {
		return ""
	}

	// Replace home directory with ~/
	if homeDir := paths.GetHomeDir(); homeDir != "" && strings.HasPrefix(pwd, homeDir) {
		pwd = "~" + pwd[len(homeDir):]
	}

	return pwd
}

// Update handles messages and updates the component state.
func (m *model) Update(msg tea.Msg) (layout.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		cmd := m.SetSize(msg.Width, msg.Height)
		return m, cmd
	case *runtime.TokenUsageEvent:
		m.SetTokenUsage(msg)
		return m, nil
	case *runtime.MCPInitStartedEvent:
		m.mcpInit = true
		return m, m.spinner.Init()
	case *runtime.MCPInitFinishedEvent:
		m.mcpInit = false
		return m, nil
	case *runtime.RAGIndexingStartedEvent:
		// Use composite key: "ragName/strategyName" to differentiate strategies within same RAG manager
		key := msg.RAGName + "/" + msg.StrategyName
		slog.Debug("Sidebar received RAG indexing started event", "rag", msg.RAGName, "strategy", msg.StrategyName, "key", key)
		state := &ragIndexingState{
			spinner: spinner.New(spinner.ModeSpinnerOnly),
		}
		m.ragIndexing[key] = state
		return m, state.spinner.Init()
	case *runtime.RAGIndexingProgressEvent:
		key := msg.RAGName + "/" + msg.StrategyName
		slog.Debug("Sidebar received RAG indexing progress event", "rag", msg.RAGName, "strategy", msg.StrategyName, "current", msg.Current, "total", msg.Total)
		if state, exists := m.ragIndexing[key]; exists {
			state.current = msg.Current
			state.total = msg.Total
		}
		return m, nil
	case *runtime.RAGIndexingCompletedEvent:
		key := msg.RAGName + "/" + msg.StrategyName
		slog.Debug("Sidebar received RAG indexing completed event", "rag", msg.RAGName, "strategy", msg.StrategyName)
		delete(m.ragIndexing, key)
		return m, nil
	case *runtime.SessionTitleEvent:
		m.sessionTitle = msg.Title
		return m, nil
	case *runtime.AgentInfoEvent:
		m.SetAgentInfo(msg.AgentName, msg.Model, msg.Description)
		return m, nil
	case *runtime.TeamInfoEvent:
		m.SetTeamInfo(msg.AvailableAgents)
		return m, nil
	case *runtime.AgentSwitchingEvent:
		m.SetAgentSwitching(msg.Switching)
		return m, nil
	case *runtime.ToolsetInfoEvent:
		m.SetToolsetInfo(msg.AvailableTools)
		return m, nil
	default:
		var cmds []tea.Cmd

		// Update main spinner
		if m.mcpInit {
			model, cmd := m.spinner.Update(msg)
			m.spinner = model.(spinner.Spinner)
			cmds = append(cmds, cmd)
		}

		// Update each RAG indexing spinner
		for _, state := range m.ragIndexing {
			model, cmd := state.spinner.Update(msg)
			state.spinner = model.(spinner.Spinner)
			cmds = append(cmds, cmd)
		}

		if len(cmds) > 0 {
			return m, tea.Batch(cmds...)
		}
		return m, nil
	}
}

// View renders the component
func (m *model) View() string {
	if m.mode == ModeVertical {
		return m.verticalView()
	}

	return m.horizontalView()
}

func (m *model) horizontalView() string {
	pwd := getCurrentWorkingDirectory()
	usageSummary := m.tokenUsageSummary()

	wi := m.workingIndicatorHorizontal()
	titleGapWidth := m.width - lipgloss.Width(m.sessionTitle) - lipgloss.Width(wi) - 2
	title := fmt.Sprintf("%s%*s%s", m.sessionTitle, titleGapWidth, "", wi)

	gapWidth := m.width - lipgloss.Width(pwd) - lipgloss.Width(usageSummary) - 2
	return lipgloss.JoinVertical(lipgloss.Top, title, fmt.Sprintf("%s%*s%s", styles.MutedStyle.Render(pwd), gapWidth, "", usageSummary))
}

func (m *model) verticalView() string {
	var main []string
	if sessionInfo := m.sessionInfo(); sessionInfo != "" {
		main = append(main, sessionInfo)
	}
	if agentInfo := m.agentInfo(); agentInfo != "" {
		main = append(main, agentInfo)
	}
	if toolsetInfo := m.toolsetInfo(); toolsetInfo != "" {
		main = append(main, toolsetInfo)
	}
	if usage := m.tokenUsage(); usage != "" {
		main = append(main, usage)
	}

	m.todoComp.SetSize(m.width)
	todoContent := strings.TrimSuffix(m.todoComp.Render(), "\n")
	if todoContent != "" {
		main = append(main, todoContent)
	}

	return strings.Join(main, "\n")
}

func (m *model) workingIndicator() string {
	var indicators []string

	// Add MCP init indicator if initializing
	if m.mcpInit {
		indicators = append(indicators, styles.ActiveStyle.Render(m.spinner.View()+" "+"Initializing MCP servers…"))
	}

	// Add RAG indexing indicators for each active indexing operation
	// Group strategies by RAG source to avoid repeating the source name
	ragGroups := make(map[string][]struct {
		strategyName string
		state        *ragIndexingState
	})

	for key, state := range m.ragIndexing {
		parts := strings.Split(key, "/")
		if len(parts) == 2 {
			ragName := parts[0]
			if ragGroups[ragName] == nil {
				ragGroups[ragName] = []struct {
					strategyName string
					state        *ragIndexingState
				}{}
			}
			ragGroups[ragName] = append(ragGroups[ragName], struct {
				strategyName string
				state        *ragIndexingState
			}{parts[1], state})
		}
	}

	// Display each RAG source with its strategies (sorted for stable display)
	ragNames := make([]string, 0, len(ragGroups))
	for ragName := range ragGroups {
		ragNames = append(ragNames, ragName)
	}
	sort.Strings(ragNames)

	for _, ragName := range ragNames {
		strategies := ragGroups[ragName]
		displayRagName := strings.ReplaceAll(ragName, "_", " ")

		// Sort strategies by name for stable display
		sort.Slice(strategies, func(i, j int) bool {
			return strategies[i].strategyName < strategies[j].strategyName
		})

		// Display RAG source name as header
		ragNameStyled := styles.BoldStyle.Render(displayRagName)
		header := fmt.Sprintf("Indexing %s", ragNameStyled)
		indicators = append(indicators, styles.ActiveStyle.Render(header))

		// Following lines: each strategy with its own spinner and progress
		for _, strategy := range strategies {
			displayStratName := strings.ReplaceAll(strategy.strategyName, "-", " ")
			progress := ""
			if strategy.state.total > 0 {
				progress = fmt.Sprintf(" [%d/%d]", strategy.state.current, strategy.state.total)
			}
			stratNameStyled := styles.BoldStyle.Render(displayStratName)
			// Show spinner for each strategy
			spinnerView := strategy.state.spinner.View()
			line := fmt.Sprintf("  %s %s%s", spinnerView, stratNameStyled, progress)
			indicators = append(indicators, line)
		}
	}

	if len(indicators) == 0 {
		return ""
	}

	return strings.Join(indicators, "\n")
}

// workingIndicatorHorizontal returns a single-line version of the working indicator for horizontal mode
func (m *model) workingIndicatorHorizontal() string {
	var labels []string

	// Add MCP init indicator if initializing
	if m.mcpInit {
		labels = append(labels, "Initializing MCP servers…")
	}

	// Add RAG indexing labels for each active indexing operation
	// Group strategies by RAG source to avoid repeating the source name
	ragGroups := make(map[string][]struct {
		strategyName string
		state        *ragIndexingState
	})

	for key, state := range m.ragIndexing {
		parts := strings.Split(key, "/")
		if len(parts) == 2 {
			ragName := parts[0]
			if ragGroups[ragName] == nil {
				ragGroups[ragName] = []struct {
					strategyName string
					state        *ragIndexingState
				}{}
			}
			ragGroups[ragName] = append(ragGroups[ragName], struct {
				strategyName string
				state        *ragIndexingState
			}{parts[1], state})
		}
	}

	// Display each RAG source with its strategies (sorted for stable display)
	ragNames := make([]string, 0, len(ragGroups))
	for ragName := range ragGroups {
		ragNames = append(ragNames, ragName)
	}
	sort.Strings(ragNames)

	for _, ragName := range ragNames {
		strategies := ragGroups[ragName]
		displayRagName := strings.ReplaceAll(ragName, "_", " ")

		// Sort strategies by name for stable display
		sort.Slice(strategies, func(i, j int) bool {
			return strategies[i].strategyName < strategies[j].strategyName
		})

		// First line: RAG source name (bold for emphasis)
		ragNameStyled := styles.BoldStyle.Render(displayRagName)
		labels = append(labels, fmt.Sprintf("Indexing %s", ragNameStyled))

		// Following lines: each strategy with progress, indented with bullet
		for _, strategy := range strategies {
			displayStratName := strings.ReplaceAll(strategy.strategyName, "-", " ")
			progress := ""
			if strategy.state.total > 0 {
				progress = fmt.Sprintf(" [%d/%d]", strategy.state.current, strategy.state.total)
			}
			stratNameStyled := styles.BoldStyle.Render(displayStratName)
			labels = append(labels, fmt.Sprintf("  • %s%s", stratNameStyled, progress))
		}
	}

	if len(labels) == 0 {
		return ""
	}

	// For horizontal mode, show all labels separated by " | "
	label := strings.Join(labels, " | ")
	return styles.ActiveStyle.Render(m.spinner.View() + " " + label)
}

func (m *model) tokenUsage() string {
	if len(m.sessionUsage) == 0 {
		return ""
	}

	var totalTokens int64
	var totalCost float64
	for _, usage := range m.sessionUsage {
		totalTokens += usage.InputTokens + usage.OutputTokens
		totalCost += usage.Cost
	}

	var tokenUsage strings.Builder
	fmt.Fprintf(&tokenUsage, "%s", formatTokenCount(totalTokens))
	if ctxText, ok := m.contextPercent(); ok {
		fmt.Fprintf(&tokenUsage, " (%s)", ctxText)
	}
	fmt.Fprintf(&tokenUsage, " %s", styles.TabAccentStyle.Render("$"+formatCost(totalCost)))

	return m.renderTab("Token Usage", tokenUsage.String())
}

// tokenUsageSummary returns a single-line summary for horizontal layout.
func (m *model) tokenUsageSummary() string {
	if len(m.sessionUsage) == 0 {
		return ""
	}

	var totalTokens int64
	var totalCost float64
	for _, usage := range m.sessionUsage {
		totalTokens += usage.InputTokens + usage.OutputTokens
		totalCost += usage.Cost
	}

	if ctxText, ok := m.contextPercent(); ok {
		return fmt.Sprintf("Tokens: %s | Cost: $%s | Context: %s", formatTokenCount(totalTokens), formatCost(totalCost), ctxText)
	}

	return fmt.Sprintf("Tokens: %s | Cost: $%s", formatTokenCount(totalTokens), formatCost(totalCost))
}

func (m *model) sessionInfo() string {
	var lines []string

	lines = append(lines, m.sessionTitle, "")
	if pwd := getCurrentWorkingDirectory(); pwd != "" {
		lines = append(lines, styles.TabAccentStyle.Render("█")+styles.TabPrimaryStyle.Render(" "+pwd))
	}
	if m.yolo {
		lines = append(lines, styles.TabAccentStyle.Render("✓")+styles.TabPrimaryStyle.Render(" YOLO mode enabled"))
	}
	if working := m.workingIndicator(); working != "" {
		lines = append(lines, working)
	}

	return m.renderTab("Session", strings.Join(lines, "\n"))
}

// agentInfo renders the current agent information
func (m *model) agentInfo() string {
	if m.currentAgent == "" {
		return ""
	}

	// Agent name with highlight and switching indicator
	agentTitle := "Agent"
	if m.agentSwitching {
		agentTitle += " ↔" // switching indicator
	}

	// Current agent name
	agentName := m.currentAgent
	if m.agentSwitching {
		agentName = "⟳ " + agentName // switching icon
	}

	var content strings.Builder
	content.WriteString(styles.TabAccentStyle.Render(agentName))

	// Agent description if available
	if m.agentDescription != "" {
		description := m.agentDescription
		maxDescWidth := m.width - 2
		if len(description) > maxDescWidth {
			description = description[:maxDescWidth-1] + "…"
		}

		fmt.Fprintf(&content, "\n%s", description)
	}

	// Team info if multiple agents available
	if len(m.availableAgents) > 1 {
		fmt.Fprintf(&content, "\nTeam: %d agents", len(m.availableAgents))
	}

	// Model info if available
	if m.agentModel != "" {
		provider, model, _ := strings.Cut(m.agentModel, "/")
		fmt.Fprintf(&content, "\nProvider: %s", provider)
		fmt.Fprintf(&content, "\nModel: %s", model)
	}

	return m.renderTab(agentTitle, content.String())
}

// toolsetInfo renders the current toolset status information
func (m *model) toolsetInfo() string {
	if m.availableTools == 0 {
		return ""
	}

	return m.renderTab("Tools", fmt.Sprintf("%d tools available", m.availableTools))
}

// SetSize sets the dimensions of the component
func (m *model) SetSize(width, height int) tea.Cmd {
	m.width = width
	m.height = height
	m.todoComp.SetSize(width)
	return nil
}

// GetSize returns the current dimensions
func (m *model) GetSize() (width, height int) {
	return m.width, m.height
}

func (m *model) SetMode(mode Mode) {
	m.mode = mode
}

func (m *model) renderTab(title, content string) string {
	return tab.Render(title, content, m.width-2)
}
