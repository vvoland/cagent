package styles

import (
	"image/color"
	"sync"

	"charm.land/lipgloss/v2"
)

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
// precomputed styles for each palette entry.
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

// rebuildAgentColorCache precomputes badge and accent styles from the current theme's hues.
// Must be called with agentRegistry.Lock held.
func rebuildAgentColorCache() {
	theme := CurrentTheme()

	hues := theme.Colors.AgentHues
	if len(hues) == 0 {
		hues = DefaultAgentHues
	}

	bg := lipgloss.Color(theme.Colors.Background)
	badgeColors := GenerateBadgePalette(hues, bg)
	accentColors := GenerateAccentPalette(hues, bg)

	agentRegistry.badgeStyles = make([]cachedBadgeStyle, len(badgeColors))
	for i, bgColor := range badgeColors {
		r, g, b := ColorToRGB(bgColor)
		bgHex := RGBToHex(r, g, b)
		fgHex := BestForegroundHex(
			bgHex,
			theme.Colors.TextBright,
			theme.Colors.Background,
			"#000000",
			"#ffffff",
		)
		colors := AgentBadgeColors{
			Fg: lipgloss.Color(fgHex),
			Bg: bgColor,
		}
		agentRegistry.badgeStyles[i] = cachedBadgeStyle{
			colors: colors,
			style: BaseStyle.
				Foreground(colors.Fg).
				Background(colors.Bg).
				Padding(0, 1),
		}
	}

	agentRegistry.accentStyles = make([]lipgloss.Style, len(accentColors))
	for i, c := range accentColors {
		agentRegistry.accentStyles[i] = BaseStyle.Foreground(c)
	}
}

// InvalidateAgentColorCache rebuilds the cached agent styles.
// Call this after a theme change so colors are recalculated against the new background.
func InvalidateAgentColorCache() {
	agentRegistry.Lock()
	defer agentRegistry.Unlock()

	rebuildAgentColorCache()
}

// paletteSize returns the current number of cached palette entries.
func paletteSize() int {
	agentRegistry.RLock()
	defer agentRegistry.RUnlock()

	return len(agentRegistry.badgeStyles)
}

// agentIndex returns the palette index for an agent name.
// Uses the registered position if available, wrapping around the palette size.
// Falls back to 0 for unknown agents.
func agentIndex(agentName string) int {
	agentRegistry.RLock()
	idx, ok := agentRegistry.indices[agentName]
	size := len(agentRegistry.badgeStyles)
	agentRegistry.RUnlock()

	if !ok {
		return 0
	}
	if size > 0 {
		return idx % size
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

	return AgentBadgeColors{
		Fg: lipgloss.Color("#ffffff"),
		Bg: lipgloss.Color("#1D63ED"),
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

	return BaseStyle.
		Foreground(lipgloss.Color("#ffffff")).
		Background(lipgloss.Color("#1D63ED")).
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

	return BaseStyle.Foreground(lipgloss.Color("#98C379"))
}
