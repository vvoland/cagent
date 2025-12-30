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
	"github.com/docker/cagent/pkg/tui/components/toolcommon"
	"github.com/docker/cagent/pkg/tui/core/layout"
	"github.com/docker/cagent/pkg/tui/service"
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
	SetTeamInfo(availableAgents []runtime.AgentDetails)
	SetAgentSwitching(switching bool)
	SetToolsetInfo(availableTools int, loading bool)
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
	ragIndexing      map[string]*ragIndexingState // strategy name -> indexing state
	spinner          spinner.Spinner
	mode             Mode
	sessionTitle     string
	currentAgent     string
	agentModel       string
	agentDescription string
	availableAgents  []runtime.AgentDetails
	agentSwitching   bool
	availableTools   int
	toolsLoading     bool // true when more tools may still be loading
	sessionState     *service.SessionState
	workingAgent     string // Name of the agent currently working (empty if none)
}

func New(sessionState *service.SessionState) Model {
	return &model{
		width:        20,
		height:       24,
		sessionUsage: make(map[string]*runtime.Usage),
		sessionAgent: make(map[string]string),
		todoComp:     todotool.NewSidebarComponent(),
		spinner:      spinner.New(spinner.ModeSpinnerOnly),
		sessionTitle: "New session",
		ragIndexing:  make(map[string]*ragIndexingState),
		sessionState: sessionState,
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
func (m *model) SetTeamInfo(availableAgents []runtime.AgentDetails) {
	m.availableAgents = availableAgents
}

// SetAgentSwitching sets whether an agent switch is in progress
func (m *model) SetAgentSwitching(switching bool) {
	m.agentSwitching = switching
}

// SetToolsetInfo sets the number of available tools and loading state
func (m *model) SetToolsetInfo(availableTools int, loading bool) {
	m.availableTools = availableTools
	m.toolsLoading = loading
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
func (m *model) contextPercent() string {
	if len(m.sessionUsage) == 1 {
		for _, usage := range m.sessionUsage {
			if usage.ContextLimit > 0 {
				percent := (float64(usage.ContextLength) / float64(usage.ContextLimit)) * 100
				return fmt.Sprintf("%.0f%%", percent)
			}
		}
	}
	return "0%"
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
	case *runtime.StreamStartedEvent:
		m.workingAgent = msg.AgentName
		return m, m.spinner.Init()
	case *runtime.StreamStoppedEvent:
		m.workingAgent = ""
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
		m.SetToolsetInfo(msg.AvailableTools, msg.Loading)
		if msg.Loading {
			return m, m.spinner.Init()
		}
		return m, nil
	default:
		var cmds []tea.Cmd

		// Update main spinner when MCP is initializing, tools are loading, or an agent is working
		if m.mcpInit || m.toolsLoading || m.workingAgent != "" {
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
	if usage := m.tokenUsage(); usage != "" {
		main = append(main, usage)
	}
	if agentInfo := m.agentInfo(); agentInfo != "" {
		main = append(main, agentInfo)
	}
	if toolsetInfo := m.toolsetInfo(); toolsetInfo != "" {
		main = append(main, toolsetInfo)
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
	var totalTokens int64
	var totalCost float64
	for _, usage := range m.sessionUsage {
		totalTokens += usage.InputTokens + usage.OutputTokens
		totalCost += usage.Cost
	}

	var tokenUsage strings.Builder
	fmt.Fprintf(&tokenUsage, "%s", formatTokenCount(totalTokens))
	if ctxText := m.contextPercent(); ctxText != "" {
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

	if ctxText := m.contextPercent(); ctxText != "" {
		return fmt.Sprintf("Tokens: %s | Cost: $%s | Context: %s", formatTokenCount(totalTokens), formatCost(totalCost), ctxText)
	}

	return fmt.Sprintf("Tokens: %s | Cost: $%s", formatTokenCount(totalTokens), formatCost(totalCost))
}

func (m *model) sessionInfo() string {
	lines := []string{
		m.sessionTitle,
		"",
	}

	if pwd := getCurrentWorkingDirectory(); pwd != "" {
		lines = append(lines, styles.TabAccentStyle.Render("█")+styles.TabPrimaryStyle.Render(" "+pwd))
	}

	return m.renderTab("Session", strings.Join(lines, "\n"))
}

// agentInfo renders the current agent information
func (m *model) agentInfo() string {
	// Read current agent from session state so sidebar updates when agent is switched
	currentAgent := m.sessionState.CurrentAgent
	if currentAgent == "" {
		return ""
	}

	agentTitle := "Agent"
	if len(m.availableAgents) > 1 {
		agentTitle = "Agents"
	}
	if m.agentSwitching {
		agentTitle += " ↔"
	}

	var content strings.Builder
	for i, agent := range m.availableAgents {
		if content.Len() > 0 {
			content.WriteString("\n\n")
		}
		isCurrent := agent.Name == currentAgent
		m.renderAgentEntry(&content, agent, isCurrent, i)
	}

	return m.renderTab(agentTitle, content.String())
}

func (m *model) renderAgentEntry(content *strings.Builder, agent runtime.AgentDetails, isCurrent bool, index int) {
	var prefix string
	if isCurrent {
		if m.workingAgent == agent.Name {
			// Style the spinner with the same green as the agent name
			prefix = styles.TabAccentStyle.Render(m.spinner.View()) + " "
		} else {
			prefix = styles.TabAccentStyle.Render("▶") + " "
		}
	}
	// Agent name
	agentNameText := prefix + styles.TabAccentStyle.Render(agent.Name)
	// Shortcut hint (^1, ^2, etc.) - show for agents 1-9
	var shortcutHint string
	if index >= 0 && index < 9 {
		shortcutHint = styles.MutedStyle.Render(fmt.Sprintf("^%d", index+1))
	}
	// Calculate space needed to right-align the shortcut
	nameWidth := lipgloss.Width(agentNameText)
	hintWidth := lipgloss.Width(shortcutHint)
	spaceWidth := max(m.width-nameWidth-hintWidth-2, 1)
	if shortcutHint != "" {
		content.WriteString(agentNameText + strings.Repeat(" ", spaceWidth) + shortcutHint)
	} else {
		content.WriteString(agentNameText)
	}

	maxWidth := m.width - 4

	if desc := agent.Description; desc != "" {
		content.WriteString("\n")
		content.WriteString(styles.MutedStyle.Render("├ "))
		content.WriteString(toolcommon.TruncateText(desc, maxWidth))
	}

	content.WriteString("\n")
	content.WriteString(styles.MutedStyle.Render("├ "))
	content.WriteString(toolcommon.TruncateText("Provider: "+agent.Provider, maxWidth))
	content.WriteString("\n")
	content.WriteString(styles.MutedStyle.Render("└ "))
	content.WriteString(toolcommon.TruncateText("Model: "+agent.Model, maxWidth))
}

// toolsetInfo renders the current toolset status information
func (m *model) toolsetInfo() string {
	var lines []string
	if m.toolsLoading {
		// Show spinner with current count while loading
		if m.availableTools > 0 {
			lines = append(lines, m.spinner.View()+styles.TabPrimaryStyle.Render(fmt.Sprintf(" %d tools available…", m.availableTools)))
		} else {
			lines = append(lines, m.spinner.View()+styles.TabPrimaryStyle.Render(" Loading tools…"))
		}
	} else if m.availableTools > 0 {
		lines = append(lines, styles.TabAccentStyle.Render("█")+styles.TabPrimaryStyle.Render(fmt.Sprintf(" %d tools available", m.availableTools)))
	}

	if m.sessionState.YoloMode {
		indicator := styles.TabAccentStyle.Render("✓") + styles.TabPrimaryStyle.Render(" YOLO mode enabled")
		shortcut := lipgloss.PlaceHorizontal(m.width-lipgloss.Width(indicator)-2, lipgloss.Right, styles.MutedStyle.Render("^y"))
		lines = append(lines, indicator+shortcut)
	}

	if m.sessionState.HideToolResults {
		indicator := styles.TabAccentStyle.Render("✓") + styles.TabPrimaryStyle.Render(" Tool output hidden")
		shortcut := lipgloss.PlaceHorizontal(m.width-lipgloss.Width(indicator)-2, lipgloss.Right, styles.MutedStyle.Render("^o"))
		lines = append(lines, indicator+shortcut)
	}

	if m.sessionState.SplitDiffView {
		indicator := styles.TabAccentStyle.Render("✓") + styles.TabPrimaryStyle.Render(" Split Diff View enabled")
		shortcut := lipgloss.PlaceHorizontal(m.width-lipgloss.Width(indicator)-2, lipgloss.Right, styles.MutedStyle.Render("^t"))
		lines = append(lines, indicator+shortcut)
	}

	if working := m.workingIndicator(); working != "" {
		lines = append(lines, working)
	}

	return m.renderTab("Tools", lipgloss.JoinVertical(lipgloss.Top, lines...))
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
