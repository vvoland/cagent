package tool

import (
	"os"
	"strings"

	"github.com/aymanbagabas/go-udiff"
)

func computeDiff(path, oldText, newText string) []*udiff.Hunk {
	currentContent, err := os.ReadFile(path)
	if err != nil {
		return []*udiff.Hunk{}
	}

	// Generate the old contents by applying inverse diff, the current file has
	// newText applied, so we need to reverse it
	oldContent := strings.Replace(string(currentContent), newText, oldText, 1)

	// Now compute diff between old (reconstructed) and new (complete file)
	edits := udiff.Strings(oldContent, string(currentContent))

	diff, err := udiff.ToUnifiedDiff("old", "new", oldContent, edits, 3)
	if err != nil {
		return []*udiff.Hunk{}
	}

	return diff.Hunks
}
