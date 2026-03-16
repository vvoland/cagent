package runtime

import (
	"testing"

	"github.com/docker/docker-agent/pkg/tools"
)

func TestToolLoopDetector(t *testing.T) {
	makeCalls := func(pairs ...string) []tools.ToolCall {
		var calls []tools.ToolCall
		for i := 0; i < len(pairs); i += 2 {
			calls = append(calls, tools.ToolCall{
				Function: tools.FunctionCall{
					Name:      pairs[i],
					Arguments: pairs[i+1],
				},
			})
		}
		return calls
	}

	tests := []struct {
		name      string
		threshold int
		batches   [][]tools.ToolCall
		wantTrip  bool // whether any record call returns true
	}{
		{
			name:      "no loop with varied calls",
			threshold: 3,
			batches: [][]tools.ToolCall{
				makeCalls("read_file", `{"path":"a.txt"}`),
				makeCalls("read_file", `{"path":"b.txt"}`),
				makeCalls("write_file", `{"path":"c.txt"}`),
			},
			wantTrip: false,
		},
		{
			name:      "loop detected at exact threshold",
			threshold: 3,
			batches: [][]tools.ToolCall{
				makeCalls("read_file", `{"path":"a.txt"}`),
				makeCalls("read_file", `{"path":"a.txt"}`),
				makeCalls("read_file", `{"path":"a.txt"}`),
			},
			wantTrip: true,
		},
		{
			name:      "counter resets when calls change",
			threshold: 3,
			batches: [][]tools.ToolCall{
				makeCalls("read_file", `{"path":"a.txt"}`),
				makeCalls("read_file", `{"path":"a.txt"}`),
				makeCalls("read_file", `{"path":"b.txt"}`), // reset
				makeCalls("read_file", `{"path":"b.txt"}`),
			},
			wantTrip: false,
		},
		{
			name:      "empty calls never trigger",
			threshold: 2,
			batches: [][]tools.ToolCall{
				{},
				{},
				{},
			},
			wantTrip: false,
		},
		{
			name:      "multi-tool batches compared correctly",
			threshold: 2,
			batches: [][]tools.ToolCall{
				makeCalls("read_file", `{"path":"a"}`, "write_file", `{"path":"b"}`),
				makeCalls("read_file", `{"path":"a"}`, "write_file", `{"path":"b"}`),
			},
			wantTrip: true,
		},
		{
			name:      "multi-tool batches differ by one argument",
			threshold: 2,
			batches: [][]tools.ToolCall{
				makeCalls("read_file", `{"path":"a"}`, "write_file", `{"path":"b"}`),
				makeCalls("read_file", `{"path":"a"}`, "write_file", `{"path":"c"}`),
			},
			wantTrip: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			d := newToolLoopDetector(tt.threshold)
			var tripped bool
			for _, batch := range tt.batches {
				if d.record(batch) {
					tripped = true
				}
			}
			if tripped != tt.wantTrip {
				t.Errorf("tripped = %v, want %v", tripped, tt.wantTrip)
			}
		})
	}
}
