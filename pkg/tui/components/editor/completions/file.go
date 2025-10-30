package completions

import (
	"github.com/docker/cagent/pkg/fsx"
	"github.com/docker/cagent/pkg/tui/components/completion"
)

type fileCompletion struct{}

func NewFileCompletion() Completion {
	return &fileCompletion{}
}

func (c *fileCompletion) AutoSubmit() bool {
	return false
}

func (c *fileCompletion) RequiresEmptyEditor() bool {
	return false
}

func (c *fileCompletion) Trigger() string {
	return "@"
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
