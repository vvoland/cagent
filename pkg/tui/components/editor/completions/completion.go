package completions

import "github.com/docker/cagent/pkg/tui/components/completion"

type Completion interface {
	Trigger() string
	Items() []completion.Item
	AutoSubmit() bool
}

func Completions() []Completion {
	return []Completion{
		NewCommandCompletion(),
		NewFileCompletion(),
	}
}
