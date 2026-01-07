package completions

import (
	"github.com/docker/cagent/pkg/app"
	"github.com/docker/cagent/pkg/tui/components/completion"
)

type Completion interface {
	Trigger() string
	Items() []completion.Item
	AutoSubmit() bool
	RequiresEmptyEditor() bool
	// MatchMode returns how items should be filtered (fuzzy or prefix)
	MatchMode() completion.MatchMode
}

func Completions(a *app.App) []Completion {
	return []Completion{
		NewCommandCompletion(a),
		NewFileCompletion(),
	}
}
