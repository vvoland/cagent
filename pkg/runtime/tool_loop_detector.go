package runtime

import (
	"strings"

	"github.com/docker/docker-agent/pkg/tools"
)

// toolLoopDetector detects consecutive identical tool call batches.
// When the model issues the same tool call(s) N times in a row without
// making progress, the detector signals that the agent should be terminated.
type toolLoopDetector struct {
	lastSignature string
	consecutive   int
	threshold     int
}

// newToolLoopDetector creates a detector that triggers after threshold
// consecutive identical call batches.
func newToolLoopDetector(threshold int) *toolLoopDetector {
	return &toolLoopDetector{threshold: threshold}
}

// record updates the detector with the latest tool call batch and returns
// true if the consecutive-duplicate threshold has been reached.
func (d *toolLoopDetector) record(calls []tools.ToolCall) bool {
	if len(calls) == 0 {
		return false
	}

	sig := callSignature(calls)
	if sig == d.lastSignature {
		d.consecutive++
	} else {
		d.lastSignature = sig
		d.consecutive = 1
	}

	return d.consecutive >= d.threshold
}

// callSignature builds a composite key from the name and arguments of every
// tool call in the batch. Null-byte separators prevent ambiguity between
// different call structures that could otherwise produce the same string.
func callSignature(calls []tools.ToolCall) string {
	var b strings.Builder
	for i, c := range calls {
		if i > 0 {
			b.WriteByte(0)
		}
		b.WriteString(c.Function.Name)
		b.WriteByte(0)
		b.WriteString(c.Function.Arguments)
	}
	return b.String()
}
