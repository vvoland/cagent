package completions

import "github.com/docker/cagent/pkg/tui/components/completion"

type commandCompletion struct {
	trigger string
	items   []completion.Item
}

func NewCommandCompletion() Completion {
	return &commandCompletion{
		trigger: "/",
		items:   []completion.Item{},
	}
}

func (c *commandCompletion) AutoSubmit() bool {
	return true
}

func (c *commandCompletion) Trigger() string {
	return c.trigger
}

func (c *commandCompletion) Items() []completion.Item {
	return []completion.Item{
		{
			Label:       "New",
			Description: "Start a new conversation",
			Value:       "/new",
		},
		{
			Label:       "Compact",
			Description: "Summarize the current conversation",
			Value:       "/compact",
		},
	}
}
