package searchfiles

import (
	"testing"

	"github.com/docker/cagent/pkg/tui/types"
)

func TestExtractResult(t *testing.T) {
	tests := []struct {
		name    string
		content string
		want    string
	}{
		{
			name:    "single file found",
			content: "1 file found:\ntest.txt",
			want:    "one file found: test.txt",
		},
		{
			name:    "single file found with path",
			content: "1 file found:\n/path/to/file.go",
			want:    "one file found: /path/to/file.go",
		},
		{
			name:    "two files found",
			content: "2 files found:\nfile1.txt\nfile2.txt",
			want:    "2 files found",
		},
		{
			name:    "multiple files found",
			content: "7 files found:\ngordon.yaml\ngordon_dev.yaml\ngordon_workspace/gordon_dev_modular.yaml\nold/gordon-handoff.yaml\nold/gordon_dev_bu.yaml\nold/gordon_dev_test.yaml\nold/v2/gordon_dev_modular.yaml",
			want:    "7 files found",
		},
		{
			name:    "no files found",
			content: "No files found",
			want:    "no file found",
		},
		{
			name:    "empty content",
			content: "",
			want:    "no result",
		},
		{
			name:    "error case",
			content: "Error: permission denied",
			want:    "Error: permission denied",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			msg := &types.Message{Content: tt.content}
			got := extractResult(msg)
			if got != tt.want {
				t.Errorf("extractResult() = %q, want %q", got, tt.want)
			}
		})
	}
}
