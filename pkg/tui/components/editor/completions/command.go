package completions

import (
	"context"

	"github.com/docker/cagent/pkg/app"
	"github.com/docker/cagent/pkg/tui/commands"
	"github.com/docker/cagent/pkg/tui/components/completion"
)

type commandCompletion struct {
	app *app.App
}

func NewCommandCompletion(a *app.App) Completion {
	return &commandCompletion{
		app: a,
	}
}

func (c *commandCompletion) AutoSubmit() bool {
	return true // Commands auto-submit: selecting inserts command text and sends it
}

func (c *commandCompletion) RequiresEmptyEditor() bool {
	return true
}

func (c *commandCompletion) Trigger() string {
	return "/"
}

func (c *commandCompletion) Items() []completion.Item {
	var items []completion.Item

	for _, cmd := range commands.BuildCommandCategories(context.Background(), c.app) {
		for _, command := range cmd.Commands {
			items = append(items, completion.Item{
				Label:       command.Label,
				Description: command.Description,
				Value:       command.SlashCommand,
			})
		}
	}

	return items
}

func (c *commandCompletion) MatchMode() completion.MatchMode {
	return completion.MatchPrefix
}
