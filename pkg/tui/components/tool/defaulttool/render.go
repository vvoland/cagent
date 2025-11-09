package defaulttool

import (
	"encoding/json"
	"fmt"
	"strings"

	"github.com/docker/cagent/pkg/tools"
	"github.com/docker/cagent/pkg/tui/styles"
)

// renderToolArgs renders tool arguments
func renderToolArgs(toolCall tools.ToolCall, width int) string {
	decoder := json.NewDecoder(strings.NewReader(toolCall.Function.Arguments))

	tok, err := decoder.Token()
	if err != nil {
		return ""
	}
	if delim, ok := tok.(json.Delim); !ok || delim != '{' {
		return ""
	}

	type kv struct {
		Key   string
		Value any
	}
	var kvs []kv

	for decoder.More() {
		tok, err := decoder.Token()
		if err != nil {
			return ""
		}
		key, ok := tok.(string)
		if !ok {
			return ""
		}

		var val any
		if err := decoder.Decode(&val); err != nil {
			return ""
		}

		kvs = append(kvs, kv{Key: key, Value: val})
	}
	_, _ = decoder.Token()

	style := styles.ToolCallArgs.Width(width)

	var md strings.Builder
	for i, kv := range kvs {
		if i > 0 {
			md.WriteString("\n")
		}

		var content string
		if v, ok := kv.Value.(string); ok {
			content = v
		} else {
			buf, err := json.MarshalIndent(kv.Value, "", "  ")
			if err != nil {
				content = fmt.Sprintf("%v", kv.Value)
			} else {
				content = string(buf)
			}
		}

		fmt.Fprintf(&md, "%s:\n%s", styles.ToolCallArgKey.Render(kv.Key), content)
		if !strings.HasSuffix(content, "\n") {
			md.WriteString("\n")
		}
	}

	return "\n" + style.Render(strings.TrimSuffix(md.String(), "\n"))
}
