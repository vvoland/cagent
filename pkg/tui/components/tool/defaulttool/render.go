package defaulttool

import (
	"encoding/json"
	"fmt"
	"strings"

	"github.com/docker/cagent/pkg/tools"
	"github.com/docker/cagent/pkg/tui/styles"
)

type kv struct {
	Key   string
	Value any
}

func renderToolArgs(toolCall tools.ToolCall, shortWidth, width int) string {
	args, err := decodeArguments(toolCall.Function.Arguments)
	if err != nil {
		return ""
	}

	var short strings.Builder
	var md strings.Builder
	for i, arg := range args {
		if i > 0 {
			short.WriteString(" ")
			md.WriteString("\n")
		}

		var content string
		if v, ok := arg.Value.(string); ok {
			content = v
		} else {
			buf, err := json.MarshalIndent(arg.Value, "", "  ")
			if err != nil {
				content = fmt.Sprintf("%v", arg.Value)
			} else {
				content = string(buf)
			}
		}

		fmt.Fprintf(&short, "%s=%s", arg.Key, content)
		fmt.Fprintf(&md, "%s:\n%s", arg.Key, content)
		if !strings.HasSuffix(content, "\n") {
			md.WriteString("\n")
		}
	}

	if len(short.String()) <= shortWidth && !strings.Contains(short.String(), "\n") {
		return short.String()
	}

	return "\n" + styles.ToolCallArgs.Width(width).Render(strings.TrimSuffix(md.String(), "\n"))
}

// decodeArguments decodes the JSON-encoded arguments string into an ordered slice of key-value pairs.
func decodeArguments(arguments string) ([]kv, error) {
	decoder := json.NewDecoder(strings.NewReader(arguments))

	tok, err := decoder.Token()
	if err != nil {
		return nil, err
	}
	if delim, ok := tok.(json.Delim); !ok || delim != '{' {
		return nil, err
	}

	var args []kv

	for decoder.More() {
		tok, err := decoder.Token()
		if err != nil {
			return nil, err
		}
		key, ok := tok.(string)
		if !ok {
			return nil, err
		}

		var val any
		if err := decoder.Decode(&val); err != nil {
			return nil, err
		}

		args = append(args, kv{Key: key, Value: val})
	}
	_, _ = decoder.Token()

	return args, nil
}
