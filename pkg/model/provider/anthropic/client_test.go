package anthropic

import (
	"encoding/json"
	"strings"
	"testing"

	"github.com/docker/cagent/pkg/chat"
	"github.com/docker/cagent/pkg/tools"
)

func TestConvertMessages_SkipEmptySystemText(t *testing.T) {
	msgs := []chat.Message{{
		Role:    chat.MessageRoleSystem,
		Content: "   \n\t  ",
	}}

	out := convertMessages(msgs)
	if len(out) != 0 {
		t.Fatalf("expected 0 messages, got %d", len(out))
	}
}

func TestConvertMessages_SkipEmptyUserText_NoMultiContent(t *testing.T) {
	msgs := []chat.Message{{
		Role:    chat.MessageRoleUser,
		Content: "   \n\t  ",
	}}

	out := convertMessages(msgs)
	if len(out) != 0 {
		t.Fatalf("expected 0 messages, got %d", len(out))
	}
}

func TestConvertMessages_UserMultiContent_SkipEmptyText_KeepImage(t *testing.T) {
	msgs := []chat.Message{{
		Role: chat.MessageRoleUser,
		MultiContent: []chat.MessagePart{
			{Type: chat.MessagePartTypeText, Text: "   "},
			{Type: chat.MessagePartTypeImageURL, ImageURL: &chat.MessageImageURL{URL: "data:image/png;base64,AAAA"}},
		},
	}}

	out := convertMessages(msgs)
	if len(out) != 1 {
		t.Fatalf("expected 1 message, got %d", len(out))
	}

	b, err := json.Marshal(out[0])
	if err != nil {
		t.Fatalf("marshal error: %v", err)
	}
	// Basic JSON structure checks
	var m map[string]any
	if err := json.Unmarshal(b, &m); err != nil {
		t.Fatalf("unmarshal error: %v", err)
	}
	// role should be user
	if role, _ := m["role"].(string); role != "user" {
		t.Fatalf("expected role 'user', got %v", m["role"])
	}
	// content should contain exactly one block (the image)
	if content, _ := m["content"].([]any); len(content) != 1 {
		t.Fatalf("expected 1 content block, got %d", len(content))
	}
	// and it should be an image block
	if content, _ := m["content"].([]any); len(content) == 1 {
		cb, _ := content[0].(map[string]any)
		if typ, _ := cb["type"].(string); typ != "image" {
			t.Fatalf("expected content block type 'image', got %v", typ)
		}
	}
}

func TestConvertMessages_SkipEmptyAssistantText_NoToolCalls(t *testing.T) {
	msgs := []chat.Message{{
		Role:    chat.MessageRoleAssistant,
		Content: "  \t\n  ",
	}}

	out := convertMessages(msgs)
	if len(out) != 0 {
		t.Fatalf("expected 0 messages, got %d", len(out))
	}
}

func TestConvertMessages_AssistantToolCalls_NoText_IncludesToolUse(t *testing.T) {
	msgs := []chat.Message{{
		Role:    chat.MessageRoleAssistant,
		Content: "   ",
		ToolCalls: []tools.ToolCall{
			{ID: "tool-1", Function: tools.FunctionCall{Name: "do_thing", Arguments: "{\"x\":1}"}},
		},
	}}

	out := convertMessages(msgs)
	if len(out) != 1 {
		t.Fatalf("expected 1 message, got %d", len(out))
	}

	b, err := json.Marshal(out[0])
	if err != nil {
		t.Fatalf("marshal error: %v", err)
	}
	var m map[string]any
	if err := json.Unmarshal(b, &m); err != nil {
		t.Fatalf("unmarshal error: %v", err)
	}
	if role, _ := m["role"].(string); role != "assistant" {
		t.Fatalf("expected role 'assistant', got %v", m["role"])
	}
	content, _ := m["content"].([]any)
	if len(content) != 1 {
		t.Fatalf("expected 1 content block, got %d", len(content))
	}
	cb, _ := content[0].(map[string]any)
	if typ, _ := cb["type"].(string); typ != "tool_use" {
		t.Fatalf("expected content block type 'tool_use', got %v", typ)
	}
}

func TestSystemMessages_AreExtractedAndNotInMessageList(t *testing.T) {
	msgs := []chat.Message{
		{Role: chat.MessageRoleSystem, Content: "  system rules here  "},
		{Role: chat.MessageRoleUser, Content: "hi"},
	}

	// System blocks should be extracted
	sys := extractSystemBlocks(msgs)
	if len(sys) != 1 {
		t.Fatalf("expected 1 system block, got %d", len(sys))
	}
	if strings.TrimSpace(sys[0].Text) != "system rules here" {
		t.Fatalf("unexpected system text: %q", sys[0].Text)
	}

	// System role messages must not appear in the anthropic messages list
	out := convertMessages(msgs)
	if len(out) != 1 {
		t.Fatalf("expected 1 non-system message, got %d", len(out))
	}
}

func TestSystemMessages_MultipleExtractedAndExcludedFromMessageList(t *testing.T) {
	msgs := []chat.Message{
		{Role: chat.MessageRoleSystem, Content: " sys A "},
		{Role: chat.MessageRoleSystem, Content: "\n sys B \t"},
		{Role: chat.MessageRoleUser, Content: "hello"},
	}

	sys := extractSystemBlocks(msgs)
	if len(sys) != 2 {
		t.Fatalf("expected 2 system blocks, got %d", len(sys))
	}
	if strings.TrimSpace(sys[0].Text) != "sys A" {
		t.Fatalf("unexpected first system text: %q", sys[0].Text)
	}
	if strings.TrimSpace(sys[1].Text) != "sys B" {
		t.Fatalf("unexpected second system text: %q", sys[1].Text)
	}

	out := convertMessages(msgs)
	if len(out) != 1 {
		t.Fatalf("expected 1 non-system message, got %d", len(out))
	}
}

func TestSystemMessages_InterspersedExtractedAndExcluded(t *testing.T) {
	msgs := []chat.Message{
		{Role: chat.MessageRoleSystem, Content: " S1 "},
		{Role: chat.MessageRoleUser, Content: "U1"},
		{Role: chat.MessageRoleAssistant, Content: "A1"},
		{Role: chat.MessageRoleSystem, Content: "S2"},
		{Role: chat.MessageRoleUser, Content: " U2 "},
	}

	// All system messages should be extracted in order of appearance
	sys := extractSystemBlocks(msgs)
	if len(sys) != 2 {
		t.Fatalf("expected 2 system blocks, got %d", len(sys))
	}
	if strings.TrimSpace(sys[0].Text) != "S1" {
		t.Fatalf("unexpected first system text: %q", sys[0].Text)
	}
	if strings.TrimSpace(sys[1].Text) != "S2" {
		t.Fatalf("unexpected second system text: %q", sys[1].Text)
	}

	// Converted messages must exclude system roles and preserve order of others
	out := convertMessages(msgs)
	if len(out) != 3 {
		t.Fatalf("expected 3 non-system messages, got %d", len(out))
	}
	// Check roles: user, assistant, user
	for i, expected := range []string{"user", "assistant", "user"} {
		b, err := json.Marshal(out[i])
		if err != nil {
			t.Fatalf("marshal error: %v", err)
		}
		var m map[string]any
		if err := json.Unmarshal(b, &m); err != nil {
			t.Fatalf("unmarshal error: %v", err)
		}
		if role, _ := m["role"].(string); role != expected {
			t.Fatalf("unexpected role at %d: got %q want %q", i, role, expected)
		}
	}
}
