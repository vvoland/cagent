package gemini

import (
	"testing"

	"github.com/docker/cagent/pkg/chat"
	"google.golang.org/genai"
)

func TestStreamAdapter_FunctionCalls(t *testing.T) {
	// Test that function calls are properly handled in the final message
	t.Run("function calls in final message", func(t *testing.T) {
		// Create a mock response with function calls
		mockResp := &genai.GenerateContentResponse{
			Candidates: []*genai.Candidate{
				{
					Content: &genai.Content{
						Parts: []*genai.Part{
							{
								FunctionCall: &genai.FunctionCall{
									Name: "test_function",
									Args: map[string]any{"param": "value"},
								},
							},
						},
					},
				},
			},
		}

		// Simulate the iterator behavior
		iter := func(fn func(*genai.GenerateContentResponse, error) bool) {
			// Send the response with function call
			fn(mockResp, nil)
		}

		adapter := NewStreamAdapter(iter, "test-model")

		// Read the response
		resp, err := adapter.Recv()
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}

		// Should have tool calls
		if len(resp.Choices[0].Delta.ToolCalls) == 0 {
			t.Error("expected tool calls in response")
		}

		// Read the final message
		finalResp, err := adapter.Recv()
		if err != nil {
			t.Fatalf("unexpected error reading final message: %v", err)
		}

		// Should have finish reason tool_calls
		if finalResp.Choices[0].FinishReason != chat.FinishReasonToolCalls {
			t.Errorf("expected finish reason tool_calls, got %v", finalResp.Choices[0].FinishReason)
		}

		// Should NOT include tool calls in final message (to avoid duplication)
		if len(finalResp.Choices[0].Delta.ToolCalls) != 0 {
			t.Error("expected no tool calls in final message to avoid duplication")
		}
	})
}
