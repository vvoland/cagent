package completions

import (
	"github.com/docker/cagent/pkg/fsx"
	"github.com/docker/cagent/pkg/tui/components/completion"
)

type fileCompletion struct {
	trigger string
}

func NewFileCompletion() Completion {
	return &fileCompletion{
		trigger: "@",
	}
}

func (c *fileCompletion) Trigger() string {
	return c.trigger
}

func (c *fileCompletion) Items() []completion.Item {
	files, err := fsx.ListDirectory(".", 0)
	if err != nil {
		return nil
	}
	items := make([]completion.Item, len(files))
	for i, f := range files {
		items[i] = completion.Item{
			Label: f,
			Value: f,
		}
	}

	return items
}

func (c *fileCompletion) AutoSubmit() bool {
	return false
}
