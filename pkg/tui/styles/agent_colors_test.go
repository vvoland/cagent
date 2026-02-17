package styles

import (
	"fmt"
	"testing"

	"charm.land/lipgloss/v2"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// --- Agent index and registry tests ---

func TestAgentIndex_UsesRegisteredOrder(t *testing.T) {
	SetAgentOrder([]string{"root", "git-agent", "docs-writer"})
	defer SetAgentOrder(nil)

	assert.Equal(t, 0, agentIndex("root"))
	assert.Equal(t, 1, agentIndex("git-agent"))
	assert.Equal(t, 2, agentIndex("docs-writer"))
}

func TestAgentIndex_UnknownAgentReturnsFallback(t *testing.T) {
	SetAgentOrder([]string{"root", "git-agent"})
	defer SetAgentOrder(nil)

	assert.Equal(t, 0, agentIndex("unknown-agent"))
}

func TestAgentIndex_WrapsAroundPaletteSize(t *testing.T) {
	size := paletteSize()
	require.Positive(t, size)

	agents := make([]string, size+3)
	for i := range agents {
		agents[i] = fmt.Sprintf("agent-%d", i)
	}
	SetAgentOrder(agents)
	defer SetAgentOrder(nil)

	last := agents[len(agents)-1]
	idx := agentIndex(last)
	assert.Less(t, idx, size)
	assert.Equal(t, (size+2)%size, idx)
}

func TestAgentIndex_EmptyRegistryReturnsFallback(t *testing.T) {
	SetAgentOrder(nil)
	defer SetAgentOrder(nil)

	assert.Equal(t, 0, agentIndex("anything"))
}

func TestSetAgentOrder_UpdatesRegistry(t *testing.T) {
	SetAgentOrder([]string{"a", "b", "c"})
	defer SetAgentOrder(nil)

	assert.Equal(t, 0, agentIndex("a"))
	assert.Equal(t, 2, agentIndex("c"))

	SetAgentOrder([]string{"c", "b", "a"})
	assert.Equal(t, 2, agentIndex("a"))
	assert.Equal(t, 0, agentIndex("c"))
}

// --- Style rendering tests ---

func TestAgentBadgeStyleFor_ProducesDifferentStylesPerIndex(t *testing.T) {
	SetAgentOrder([]string{"root", "docs-writer"})
	defer SetAgentOrder(nil)

	rendered1 := AgentBadgeStyleFor("root").Render("root")
	rendered2 := AgentBadgeStyleFor("docs-writer").Render("docs-writer")

	require.NotEmpty(t, rendered1)
	require.NotEmpty(t, rendered2)
	assert.NotEqual(t, rendered1, rendered2)
}

func TestAgentBadgeStyleFor_Deterministic(t *testing.T) {
	SetAgentOrder([]string{"root"})
	defer SetAgentOrder(nil)

	s1 := AgentBadgeStyleFor("root").Render("root")
	s2 := AgentBadgeStyleFor("root").Render("root")
	assert.Equal(t, s1, s2)
}

func TestAgentAccentStyleFor_Deterministic(t *testing.T) {
	SetAgentOrder([]string{"root"})
	defer SetAgentOrder(nil)

	s1 := AgentAccentStyleFor("root").Render("root")
	s2 := AgentAccentStyleFor("root").Render("root")
	assert.Equal(t, s1, s2)
}

func TestAgentBadgeColorsFor_HasFgAndBg(t *testing.T) {
	SetAgentOrder([]string{"root"})
	defer SetAgentOrder(nil)

	colors := AgentBadgeColorsFor("root")
	assert.NotNil(t, colors.Fg)
	assert.NotNil(t, colors.Bg)
}

// --- Cache tests ---

func TestSetAgentOrder_PopulatesCache(t *testing.T) {
	SetAgentOrder([]string{"root", "docs-writer"})
	defer SetAgentOrder(nil)

	agentRegistry.RLock()
	defer agentRegistry.RUnlock()

	assert.NotEmpty(t, agentRegistry.badgeStyles)
	assert.NotEmpty(t, agentRegistry.accentStyles)
	assert.Len(t, agentRegistry.accentStyles, len(agentRegistry.badgeStyles))
}

func TestInvalidateAgentColorCache_RebuildsCachedStyles(t *testing.T) {
	SetAgentOrder([]string{"root"})
	defer SetAgentOrder(nil)

	before := AgentBadgeStyleFor("root").Render("root")
	InvalidateAgentColorCache()
	after := AgentBadgeStyleFor("root").Render("root")

	assert.Equal(t, before, after, "cache rebuild with same theme should produce identical styles")
}

func TestAgentBadgeStyleFor_UsesCachedStyle(t *testing.T) {
	SetAgentOrder([]string{"a", "b"})
	defer SetAgentOrder(nil)

	for range 100 {
		s := AgentBadgeStyleFor("b").Render("b")
		require.NotEmpty(t, s)
	}
}

// --- Layer 1: WCAG contrast validation across all themes ---

const (
	// minBadgeContrast is the WCAG AA minimum for normal text.
	minBadgeContrast = 4.5
	// minAccentContrast is the WCAG AA minimum for large/bold text.
	minAccentContrast = 3.0
)

func TestAllBuiltinThemes_AgentBadgeContrast(t *testing.T) {
	t.Parallel()

	refs, err := listBuiltinThemeRefs()
	require.NoError(t, err)
	require.NotEmpty(t, refs)

	for _, ref := range refs {
		t.Run(ref, func(t *testing.T) {
			t.Parallel()

			theme, err := LoadTheme(ref)
			require.NoError(t, err)

			hues := theme.Colors.AgentHues
			if len(hues) == 0 {
				hues = DefaultAgentHues
			}

			bg := lipgloss.Color(theme.Colors.Background)
			badgeColors := GenerateBadgePalette(hues, bg)

			for i, badgeBg := range badgeColors {
				r, g, b := ColorToRGB(badgeBg)
				bgHex := RGBToHex(r, g, b)
				fgHex := BestForegroundHex(
					bgHex,
					theme.Colors.TextBright,
					theme.Colors.Background,
					"#000000",
					"#ffffff",
				)
				fg := lipgloss.Color(fgHex)

				ratio := ContrastRatio(fg, badgeBg)
				assert.GreaterOrEqual(t, ratio, minBadgeContrast,
					"badge %d (bg=%s, fg=%s) contrast %.2f < %.1f in theme %s",
					i, bgHex, fgHex, ratio, minBadgeContrast, ref)
			}
		})
	}
}

func TestAllBuiltinThemes_AgentAccentContrast(t *testing.T) {
	t.Parallel()

	refs, err := listBuiltinThemeRefs()
	require.NoError(t, err)
	require.NotEmpty(t, refs)

	for _, ref := range refs {
		t.Run(ref, func(t *testing.T) {
			t.Parallel()

			theme, err := LoadTheme(ref)
			require.NoError(t, err)

			hues := theme.Colors.AgentHues
			if len(hues) == 0 {
				hues = DefaultAgentHues
			}

			bg := lipgloss.Color(theme.Colors.Background)
			accentColors := GenerateAccentPalette(hues, bg)

			for i, accent := range accentColors {
				ratio := ContrastRatio(accent, bg)
				r, g, b := ColorToRGB(accent)
				hex := RGBToHex(r, g, b)
				assert.GreaterOrEqual(t, ratio, minAccentContrast,
					"accent %d (%s) contrast %.2f < %.1f against bg %s in theme %s",
					i, hex, ratio, minAccentContrast, theme.Colors.Background, ref)
			}
		})
	}
}

// --- Layer 1: Pairwise color distinctness across all themes ---

const (
	// minColorDistance is the minimum CIE76 ΔE between adjacent palette entries.
	// Below ~15 colors become hard to distinguish at a glance.
	minColorDistance = 10.0
)

func TestAllBuiltinThemes_AgentBadgeDistinctness(t *testing.T) {
	t.Parallel()

	refs, err := listBuiltinThemeRefs()
	require.NoError(t, err)
	require.NotEmpty(t, refs)

	for _, ref := range refs {
		t.Run(ref, func(t *testing.T) {
			t.Parallel()

			theme, err := LoadTheme(ref)
			require.NoError(t, err)

			hues := theme.Colors.AgentHues
			if len(hues) == 0 {
				hues = DefaultAgentHues
			}

			bg := lipgloss.Color(theme.Colors.Background)
			palette := GenerateBadgePalette(hues, bg)

			for i := range palette {
				for j := i + 1; j < len(palette); j++ {
					dist := ColorDistanceCIE76(palette[i], palette[j])
					assert.GreaterOrEqual(t, dist, minColorDistance,
						"badge colors %d and %d are too similar (ΔE=%.1f) in theme %s",
						i, j, dist, ref)
				}
			}
		})
	}
}

func TestAllBuiltinThemes_AgentAccentDistinctness(t *testing.T) {
	t.Parallel()

	refs, err := listBuiltinThemeRefs()
	require.NoError(t, err)
	require.NotEmpty(t, refs)

	for _, ref := range refs {
		t.Run(ref, func(t *testing.T) {
			t.Parallel()

			theme, err := LoadTheme(ref)
			require.NoError(t, err)

			hues := theme.Colors.AgentHues
			if len(hues) == 0 {
				hues = DefaultAgentHues
			}

			bg := lipgloss.Color(theme.Colors.Background)
			palette := GenerateAccentPalette(hues, bg)

			for i := range palette {
				for j := i + 1; j < len(palette); j++ {
					dist := ColorDistanceCIE76(palette[i], palette[j])
					assert.GreaterOrEqual(t, dist, minColorDistance,
						"accent colors %d and %d are too similar (ΔE=%.1f) in theme %s",
						i, j, dist, ref)
				}
			}
		})
	}
}

// --- Layer 2: Color audit report (run with -v) ---

func TestAgentColorAuditReport(t *testing.T) {
	t.Parallel()

	refs, err := listBuiltinThemeRefs()
	require.NoError(t, err)

	for _, ref := range refs {
		t.Run(ref, func(t *testing.T) {
			t.Parallel()

			theme, err := LoadTheme(ref)
			require.NoError(t, err)

			hues := theme.Colors.AgentHues
			if len(hues) == 0 {
				hues = DefaultAgentHues
			}

			bg := lipgloss.Color(theme.Colors.Background)
			badges := GenerateBadgePalette(hues, bg)
			accents := GenerateAccentPalette(hues, bg)

			t.Logf("\n=== Agent Color Audit: %s (bg: %s) ===", theme.Name, theme.Colors.Background)
			t.Logf("%-5s %-10s %-10s %-10s %-12s %-10s %-10s",
				"Idx", "Hue", "Badge", "Badge FG", "Badge CR", "Accent", "Accent CR")
			t.Logf("%-5s %-10s %-10s %-10s %-12s %-10s %-10s",
				"---", "---", "---", "---", "---", "---", "---")

			for i := range hues {
				br, bg2, bb := ColorToRGB(badges[i])
				badgeHex := RGBToHex(br, bg2, bb)

				fgHex := BestForegroundHex(badgeHex,
					theme.Colors.TextBright, theme.Colors.Background,
					"#000000", "#ffffff")
				fg := lipgloss.Color(fgHex)
				badgeCR := ContrastRatio(fg, badges[i])

				ar, ag, ab := ColorToRGB(accents[i])
				accentHex := RGBToHex(ar, ag, ab)
				accentCR := ContrastRatio(accents[i], bg)

				badgeStatus := "✓"
				if badgeCR < minBadgeContrast {
					badgeStatus = "✗"
				}
				accentStatus := "✓"
				if accentCR < minAccentContrast {
					accentStatus = "✗"
				}

				t.Logf("%-5d %-10.0f %-10s %-10s %s %-9.2f %-10s %s %.2f",
					i, hues[i], badgeHex, fgHex, badgeStatus, badgeCR,
					accentHex, accentStatus, accentCR)
			}

			// Log minimum pairwise distances
			minBadgeDist := 999.0
			minAccentDist := 999.0
			for i := range badges {
				for j := i + 1; j < len(badges); j++ {
					if d := ColorDistanceCIE76(badges[i], badges[j]); d < minBadgeDist {
						minBadgeDist = d
					}
					if d := ColorDistanceCIE76(accents[i], accents[j]); d < minAccentDist {
						minAccentDist = d
					}
				}
			}
			t.Logf("\nMin badge pairwise ΔE:  %.1f (threshold: %.1f)", minBadgeDist, minColorDistance)
			t.Logf("Min accent pairwise ΔE: %.1f (threshold: %.1f)", minAccentDist, minColorDistance)
		})
	}
}
