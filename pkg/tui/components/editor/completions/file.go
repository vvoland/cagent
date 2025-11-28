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
	// Try to create VCS matcher for current directory
	vcsMatcher, err := fsx.NewVCSMatcher(".")

	// Prepare shouldIgnore function
	var shouldIgnore func(string) bool
	if err == nil && vcsMatcher != nil {
		shouldIgnore = vcsMatcher.ShouldIgnore
	}
	// If vcsMatcher is nil (not in git repo), shouldIgnore stays nil = show all files

	// Get files with optional VCS filtering
	files, err := fsx.ListDirectory(".", shouldIgnore)
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
