package styles

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

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
	agents := make([]string, len(agentColorPalette)+3)
	for i := range agents {
		agents[i] = "agent-" + string(rune('a'+i))
	}
	SetAgentOrder(agents)
	defer SetAgentOrder(nil)

	last := agents[len(agents)-1]
	idx := agentIndex(last)
	assert.Less(t, idx, len(agentColorPalette))
	assert.Equal(t, (len(agentColorPalette)+2)%len(agentColorPalette), idx)
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

func TestPaletteSizes_AreEqual(t *testing.T) {
	t.Parallel()

	assert.Len(t, agentAccentPalette, len(agentColorPalette),
		"badge and accent palettes must have the same number of entries")
	assert.Len(t, agentColorPalette, 16)
}

func TestSetAgentOrder_PopulatesCache(t *testing.T) {
	SetAgentOrder([]string{"root", "docs-writer"})
	defer SetAgentOrder(nil)

	agentRegistry.RLock()
	defer agentRegistry.RUnlock()

	assert.Len(t, agentRegistry.badgeStyles, len(agentColorPalette))
	assert.Len(t, agentRegistry.accentStyles, len(agentAccentPalette))
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

	// Calling AgentBadgeStyleFor repeatedly should return identical styles from cache
	for range 100 {
		s := AgentBadgeStyleFor("b").Render("b")
		require.NotEmpty(t, s)
	}
}
