package builtin

import (
	"context"
	"encoding/json"

	"github.com/docker/cagent/pkg/tools"
)

func NewHandler[T any](fn func(_ context.Context, params T) (*tools.ToolCallResult, error)) func(ctx context.Context, toolCall tools.ToolCall) (*tools.ToolCallResult, error) {
	return func(ctx context.Context, toolCall tools.ToolCall) (*tools.ToolCallResult, error) {
		var params T
		if err := json.Unmarshal([]byte(toolCall.Function.Arguments), &params); err != nil {
			return nil, err
		}

		return fn(ctx, params)
	}
}
