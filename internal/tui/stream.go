package tui

import (
	"context"
	"fmt"
	"time"

	tea "github.com/charmbracelet/bubbletea/v2"

	"github.com/docker/cagent/pkg/runtime"
	"github.com/docker/cagent/pkg/session"
)

func processStream(rt *runtime.Runtime, sess *session.Session, ch chan<- string, toolCh chan<- any) tea.Cmd {
	return func() tea.Msg {
		ctx := context.Background()
		first := true

		// Signal that work has started
		toolCh <- workStartMsg{}

		for event := range rt.RunStream(ctx, sess) {
			switch e := event.(type) {
			case *runtime.AgentChoiceEvent:
				if first {
					ch <- fmt.Sprintf("\n**%s**: ", rt.CurrentAgent().Name())
					first = false
				}
				ch <- e.Choice.Delta.Content
			case *runtime.ToolCallEvent:
				// Send tool start event
				toolCall := ToolCall{
					ID:        e.ToolCall.ID,
					Name:      e.ToolCall.Function.Name,
					Arguments: e.ToolCall.Function.Arguments,
					IsActive:  true,
					StartTime: time.Now(),
				}
				toolCh <- toolCallMsg{toolCall: toolCall}

				// Add to chat content
				ch <- fmt.Sprintf("\n\n> 🔧 **Tool Call**: `%s(%s)`\n\n",
					e.ToolCall.Function.Name,
					truncateWithEllipsis(e.ToolCall.Function.Arguments, 60))

			case *runtime.ToolCallResponseEvent:
				// Send tool completion event
				toolCh <- toolCompleteMsg{id: e.ToolCall.ID, response: e.Response}

				// Add completion to chat content
				ch <- fmt.Sprintf("> ✅ **Completed**: `%s`\n\n",
					truncateWithEllipsis(e.Response, 60))

			case *runtime.ErrorEvent:
				close(ch)
				close(toolCh)
				return errorMsg(e.Error)
			}
		}

		// Signal that work has ended
		toolCh <- workEndMsg{}
		close(ch)
		close(toolCh)
		return nil
	}
}

func readResponse(ch <-chan string) tea.Cmd {
	return func() tea.Msg {
		if msg, ok := <-ch; ok {
			return responseMsg{content: msg}
		}
		return nil
	}
}

func readToolEvents(ch <-chan any) tea.Cmd {
	return func() tea.Msg {
		if msg, ok := <-ch; ok {
			return msg
		}
		return nil
	}
}
