package gemini

import (
	"encoding/json"
	"fmt"
	"io"
	"log/slog"
	"sync"

	"github.com/docker/cagent/pkg/chat"
	"github.com/docker/cagent/pkg/tools"
	"google.golang.org/genai"
)

// StreamAdapter adapts the Gemini streaming iterator to chat.MessageStream
type StreamAdapter struct {
	ch           chan result
	model        string
	mu           sync.Mutex
	lastResponse *genai.GenerateContentResponse // Store last response for final message
	logger       *slog.Logger
}

type result struct {
	resp *genai.GenerateContentResponse
	err  error
	done bool
}

// NewStreamAdapter constructs a StreamAdapter from Gemini's iterator
func NewStreamAdapter(iter func(func(*genai.GenerateContentResponse, error) bool), model string, logger *slog.Logger) *StreamAdapter {
	adapter := &StreamAdapter{
		ch:     make(chan result),
		model:  model,
		logger: logger,
	}

	go func() {
		defer close(adapter.ch)

		hasContent := false
		hasToolCalls := false

		// Consume the iterator
		iter(func(resp *genai.GenerateContentResponse, err error) bool {
			// Skip noisy http2 errors
			if err != nil && err.Error() == "http2: response body closed" {
				return true
			}

			if err != nil {
				adapter.ch <- result{err: err}
				return false
			}

			if resp != nil {
				// Check for text content without using Text() to avoid warnings
				hasText := false
				for _, candidate := range resp.Candidates {
					if candidate.Content != nil {
						for _, part := range candidate.Content.Parts {
							if part.Text != "" {
								hasText = true
								break
							}
						}
					}
					if hasText {
						break
					}
				}

				// Check for function calls
				hasFuncs := len(resp.FunctionCalls()) > 0

				// Send response if it has content or function calls
				if hasText || hasFuncs {
					if hasText {
						hasContent = true
					}
					if hasFuncs {
						hasToolCalls = true
					}
					adapter.mu.Lock()
					adapter.lastResponse = resp // Store for final message
					adapter.mu.Unlock()
					adapter.ch <- result{resp: resp}
				}
			}

			return true
		})

		// Send final message with appropriate stop reason
		if hasContent || hasToolCalls {
			// Use the last response if available to preserve function calls
			adapter.mu.Lock()
			finalResp := adapter.lastResponse
			adapter.mu.Unlock()
			if finalResp == nil {
				finalResp = &genai.GenerateContentResponse{}
			}
			adapter.ch <- result{done: true, resp: finalResp}
		}
	}()

	return adapter
}

// Recv gets the next Gemini content chunk
func (g *StreamAdapter) Recv() (chat.MessageStreamResponse, error) {
	res, ok := <-g.ch
	if !ok {
		return chat.MessageStreamResponse{}, io.EOF
	}

	if res.err != nil {
		return chat.MessageStreamResponse{}, res.err
	}

	// Build response
	resp := chat.MessageStreamResponse{
		Model:   g.model,
		Choices: []chat.MessageStreamChoice{{}},
	}

	if res.done {
		// Set finish reason and role
		resp.Choices[0].Delta.Role = string(chat.MessageRoleAssistant)

		// Check if we have function calls in the final response
		if res.resp != nil && len(res.resp.FunctionCalls()) > 0 {
			resp.Choices[0].FinishReason = chat.FinishReasonToolCalls
			// Don't include function calls in the final message - they were already sent
			g.logger.Debug("Gemini: Final message with tool calls finish reason")
		} else {
			resp.Choices[0].FinishReason = chat.FinishReasonStop
		}
	} else if res.resp != nil {
		resp.ID = res.resp.ResponseID

		// Handle text content without using Text() to avoid warnings
		var textContent string
		for _, candidate := range res.resp.Candidates {
			if candidate.Content != nil {
				for _, part := range candidate.Content.Parts {
					if part.Text != "" {
						textContent += part.Text
					}
				}
			}
		}
		if textContent != "" {
			resp.Choices[0].Delta.Content = textContent
		}

		// Handle function calls
		if funcs := res.resp.FunctionCalls(); len(funcs) > 0 {
			resp.Choices[0].Delta.ToolCalls = make([]tools.ToolCall, len(funcs))
			for i, fc := range funcs {
				// Convert args to JSON string
				argsJSON, _ := json.Marshal(fc.Args)
				idx := i
				// Generate ID if not provided
				toolID := fc.ID
				if toolID == "" {
					toolID = fmt.Sprintf("call_%d", i)
				}
				resp.Choices[0].Delta.ToolCalls[i] = tools.ToolCall{
					Index: &idx,
					ID:    toolID,
					Type:  "function",
					Function: tools.FunctionCall{
						Name:      fc.Name,
						Arguments: string(argsJSON),
					},
				}
				g.logger.Debug("Gemini: Sending tool call", "name", fc.Name, "args", string(argsJSON), "id", toolID)
			}
		}
	}

	return resp, nil
}

// Close closes the stream
func (g *StreamAdapter) Close() {
	// Drain channel to let goroutine exit
	go func() {
		for range g.ch {
		}
	}()
}
