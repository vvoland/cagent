package styles

import (
	"image/color"
	"sync"

	"charm.land/lipgloss/v2"
)

// agentColorPalette defines distinct background colors for agent badges.
// These are chosen to be visually distinguishable and to provide good
// contrast with white text on dark backgrounds.
var agentColorPalette = []string{
	"#1D63ED", // Blue
	"#9B59B6", // Purple
	"#1ABC9C", // Teal
	"#E67E22", // Orange
	"#E74C8B", // Pink
	"#27AE60", // Green
	"#2980B9", // Steel blue
	"#8E44AD", // Deep purple
	"#D4AC0D", // Gold
	"#C0392B", // Red
	"#16A085", // Dark teal
	"#D35400", // Burnt orange
	"#2C3E99", // Indigo
	"#7D3C98", // Plum
	"#117864", // Forest green
	"#A93226", // Crimson
}

// agentAccentPalette defines foreground accent colors for agent names in the sidebar.
// These are brighter variants designed to be readable on dark backgrounds without
// a background fill.
var agentAccentPalette = []string{
	"#98C379", // Green
	"#C678DD", // Purple
	"#56B6C2", // Cyan
	"#E5C07B", // Yellow
	"#E06C9F", // Pink
	"#61AFEF", // Blue
	"#D19A66", // Orange
	"#BE5046", // Red
	"#73C991", // Mint
	"#CDA0E0", // Lavender
	"#4EC9B0", // Turquoise
	"#DCDCAA", // Khaki
	"#9CDCFE", // Ice blue
	"#CE9178", // Salmon
	"#B5CEA8", // Sage
	"#D7BA7D", // Tan
}

// AgentBadgeColors holds the resolved foreground and background colors for an agent badge.
type AgentBadgeColors struct {
	Fg color.Color
	Bg color.Color
}

// cachedBadgeStyle holds a precomputed badge style for a palette index.
type cachedBadgeStyle struct {
	colors AgentBadgeColors
	style  lipgloss.Style
}

// agentRegistry maps agent names to their index in the team list and holds
// precomputed styles for each palette index.
var agentRegistry struct {
	sync.RWMutex
	indices      map[string]int
	badgeStyles  []cachedBadgeStyle
	accentStyles []lipgloss.Style
}

// SetAgentOrder updates the agent name → index mapping and rebuilds the style cache.
// Call this when the team info changes (e.g., on TeamInfoEvent).
func SetAgentOrder(agentNames []string) {
	agentRegistry.Lock()
	defer agentRegistry.Unlock()

	agentRegistry.indices = make(map[string]int, len(agentNames))
	for i, name := range agentNames {
		agentRegistry.indices[name] = i
	}

	rebuildAgentColorCache()
}

// rebuildAgentColorCache precomputes badge and accent styles for all palette indices.
// Must be called with agentRegistry.Lock held.
func rebuildAgentColorCache() {
	theme := CurrentTheme()

	agentRegistry.badgeStyles = make([]cachedBadgeStyle, len(agentColorPalette))
	for i, bgHex := range agentColorPalette {
		fgHex := BestForegroundHex(
			bgHex,
			theme.Colors.TextBright,
			theme.Colors.Background,
			"#000000",
			"#ffffff",
		)
		colors := AgentBadgeColors{
			Fg: lipgloss.Color(fgHex),
			Bg: lipgloss.Color(bgHex),
		}
		agentRegistry.badgeStyles[i] = cachedBadgeStyle{
			colors: colors,
			style: BaseStyle.
				Foreground(colors.Fg).
				Background(colors.Bg).
				Padding(0, 1),
		}
	}

	agentRegistry.accentStyles = make([]lipgloss.Style, len(agentAccentPalette))
	for i, hex := range agentAccentPalette {
		agentRegistry.accentStyles[i] = BaseStyle.Foreground(lipgloss.Color(hex))
	}
}

// InvalidateAgentColorCache rebuilds the cached agent styles.
// Call this after a theme change so foreground contrast is recalculated.
func InvalidateAgentColorCache() {
	agentRegistry.Lock()
	defer agentRegistry.Unlock()

	rebuildAgentColorCache()
}

// agentIndex returns the palette index for an agent name.
// Uses the registered position if available, wrapping around the palette size.
// Falls back to 0 for unknown agents.
func agentIndex(agentName string) int {
	agentRegistry.RLock()
	idx, ok := agentRegistry.indices[agentName]
	agentRegistry.RUnlock()

	if ok {
		return idx % len(agentColorPalette)
	}
	return 0
}

// AgentBadgeColorsFor returns the badge foreground/background colors for a given agent name.
func AgentBadgeColorsFor(agentName string) AgentBadgeColors {
	idx := agentIndex(agentName)

	agentRegistry.RLock()
	defer agentRegistry.RUnlock()

	if idx < len(agentRegistry.badgeStyles) {
		return agentRegistry.badgeStyles[idx].colors
	}

	// Fallback if cache is not yet initialized
	return AgentBadgeColors{
		Fg: lipgloss.Color("#ffffff"),
		Bg: lipgloss.Color(agentColorPalette[idx]),
	}
}

// AgentBadgeStyleFor returns a lipgloss badge style colored for the given agent.
func AgentBadgeStyleFor(agentName string) lipgloss.Style {
	idx := agentIndex(agentName)

	agentRegistry.RLock()
	defer agentRegistry.RUnlock()

	if idx < len(agentRegistry.badgeStyles) {
		return agentRegistry.badgeStyles[idx].style
	}

	// Fallback if cache is not yet initialized
	return BaseStyle.
		Foreground(lipgloss.Color("#ffffff")).
		Background(lipgloss.Color(agentColorPalette[idx])).
		Padding(0, 1)
}

// AgentAccentStyleFor returns a foreground-only style for agent names (used in sidebar).
func AgentAccentStyleFor(agentName string) lipgloss.Style {
	idx := agentIndex(agentName)

	agentRegistry.RLock()
	defer agentRegistry.RUnlock()

	if idx < len(agentRegistry.accentStyles) {
		return agentRegistry.accentStyles[idx]
	}

	// Fallback if cache is not yet initialized
	return BaseStyle.Foreground(lipgloss.Color(agentAccentPalette[idx]))
}
