package completions

import (
	"github.com/docker/cagent/pkg/tui/commands"
	"github.com/docker/cagent/pkg/tui/components/completion"
)

type commandCompletion struct {
	items []completion.Item
}

func NewCommandCompletion() Completion {
	return &commandCompletion{
		items: []completion.Item{},
	}
}

func (c *commandCompletion) AutoSubmit() bool {
	return true
}

func (c *commandCompletion) Trigger() string {
	return "/"
}

func (c *commandCompletion) Items() []completion.Item {
	items := make([]completion.Item, len(commands.BuiltInSessionCommands()))
	for i, command := range commands.BuiltInSessionCommands() {
		items[i] = completion.Item{
			Label:       command.Label,
			Description: command.Description,
			Value:       command.SlashCommand,
			Execute:     command.Execute,
		}
	}
	return items
}
