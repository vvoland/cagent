package completion

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestCompletionManagerStaysOpenWithNoResults(t *testing.T) {
	t.Parallel()

	t.Run("query with no matches keeps popup visible", func(t *testing.T) {
		t.Parallel()

		m := New().(*manager)

		// Open with some items
		m.Update(OpenMsg{
			Items: []Item{
				{Label: "new", Value: "/new"},
				{Label: "exit", Value: "/exit"},
			},
		})

		assert.True(t, m.visible, "should be visible after open")
		assert.Len(t, m.filteredItems, 2, "should have 2 items")

		// Query that matches nothing
		m.Update(QueryMsg{Query: "xyz"})

		assert.True(t, m.visible, "should stay visible even with no matches")
		assert.Empty(t, m.filteredItems, "should have no filtered items")
	})

	t.Run("backspace from no-match restores items", func(t *testing.T) {
		t.Parallel()

		m := New().(*manager)

		// Open with some items
		m.Update(OpenMsg{
			Items: []Item{
				{Label: "new", Value: "/new"},
				{Label: "exit", Value: "/exit"},
			},
		})

		// Query that matches nothing
		m.Update(QueryMsg{Query: "xyz"})
		assert.True(t, m.visible, "should stay visible with no matches")
		assert.Empty(t, m.filteredItems)

		// Backspace to shorter query that still matches nothing
		m.Update(QueryMsg{Query: "xy"})
		assert.True(t, m.visible, "should stay visible")
		assert.Empty(t, m.filteredItems)

		// Backspace to query that matches "exit"
		m.Update(QueryMsg{Query: "ex"})
		assert.True(t, m.visible, "should stay visible")
		assert.Len(t, m.filteredItems, 1, "should match 'exit'")
		assert.Equal(t, "exit", m.filteredItems[0].Label)

		// Backspace to empty query - should show all items
		m.Update(QueryMsg{Query: ""})
		assert.True(t, m.visible, "should stay visible")
		assert.Len(t, m.filteredItems, 2, "should show all items")
	})

	t.Run("view shows no results message when empty", func(t *testing.T) {
		t.Parallel()

		m := New().(*manager)
		m.width = 80
		m.height = 24

		// Open with some items
		m.Update(OpenMsg{
			Items: []Item{
				{Label: "new", Value: "/new"},
			},
		})

		// Query that matches nothing
		m.Update(QueryMsg{Query: "xyz"})

		view := m.View()
		assert.Contains(t, view, "No command found", "should show no results message")
	})
}
