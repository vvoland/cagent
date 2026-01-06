package runtime

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"log/slog"
	"strings"
	"time"

	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/codes"
	"go.opentelemetry.io/otel/trace"

	"github.com/docker/cagent/pkg/agent"
	"github.com/docker/cagent/pkg/chat"
	"github.com/docker/cagent/pkg/hooks"
	"github.com/docker/cagent/pkg/permissions"
	"github.com/docker/cagent/pkg/session"
	"github.com/docker/cagent/pkg/telemetry"
	"github.com/docker/cagent/pkg/tools"
)

type toolExecutor struct {
	tracer       trace.Tracer
	sessionStore session.Store
	resumeChan   chan ResumeType
	toolMap      map[string]ToolHandler
	permissions  *permissions.Checker
	workingDir   string
	env          []string
}

type toolExecutorConfig struct {
	tracer       trace.Tracer
	sessionStore session.Store
	resumeChan   chan ResumeType
	toolMap      map[string]ToolHandler
	permissions  *permissions.Checker
	workingDir   string
	env          []string
}

func newToolExecutor(cfg toolExecutorConfig) *toolExecutor {
	return &toolExecutor{
		tracer:       cfg.tracer,
		sessionStore: cfg.sessionStore,
		resumeChan:   cfg.resumeChan,
		toolMap:      cfg.toolMap,
		permissions:  cfg.permissions,
		workingDir:   cfg.workingDir,
		env:          cfg.env,
	}
}

// ProcessToolCalls handles the execution of tool calls for an agent
func (e *toolExecutor) ProcessToolCalls(ctx context.Context, sess *session.Session, calls []tools.ToolCall, agentTools []tools.Tool, a *agent.Agent, events chan Event) {
	slog.Debug("Processing tool calls", "agent", a.Name(), "call_count", len(calls))

	agentToolMap := make(map[string]tools.Tool, len(agentTools))
	for _, t := range agentTools {
		agentToolMap[t.Name] = t
	}

	for i, toolCall := range calls {
		callCtx, callSpan := e.startSpan(ctx, "runtime.tool.call", trace.WithAttributes(
			attribute.String("tool.name", toolCall.Function.Name),
			attribute.String("tool.type", string(toolCall.Type)),
			attribute.String("agent", a.Name()),
			attribute.String("session.id", sess.ID),
			attribute.String("tool.call_id", toolCall.ID),
		))

		slog.Debug("Processing tool call", "agent", a.Name(), "tool", toolCall.Function.Name, "session_id", sess.ID)

		var tool tools.Tool
		var runTool func()

		if def, exists := e.toolMap[toolCall.Function.Name]; exists {
			if _, available := agentToolMap[toolCall.Function.Name]; !available {
				slog.Warn("Tool call rejected: tool not available to agent", "agent", a.Name(), "tool", toolCall.Function.Name, "session_id", sess.ID)
				e.addToolErrorResponse(ctx, sess, toolCall, def.tool, events, a, fmt.Sprintf("Tool '%s' is not available to this agent (%s).", toolCall.Function.Name, a.Name()))
				callSpan.SetStatus(codes.Error, "tool not available to agent")
				callSpan.End()
				continue
			}
			tool = def.tool
			runTool = func() { e.runAgentTool(callCtx, def.handler, sess, toolCall, def.tool, events, a) }
		} else if t, exists := agentToolMap[toolCall.Function.Name]; exists {
			tool = t
			runTool = func() { e.runTool(callCtx, t, toolCall, events, sess, a) }
		} else {
			callSpan.SetStatus(codes.Ok, "tool not found")
			callSpan.End()
			continue
		}

		canceled := e.executeWithApproval(callCtx, sess, toolCall, tool, events, a, runTool, calls[i+1:])
		if canceled {
			callSpan.SetStatus(codes.Ok, "tool call canceled by user")
			callSpan.End()
			return
		}

		callSpan.SetStatus(codes.Ok, "tool call processed")
		callSpan.End()
	}
}

// executeWithApproval handles the tool approval flow and executes the tool.
// Returns true if the operation was canceled and processing should stop.
func (e *toolExecutor) executeWithApproval(
	ctx context.Context,
	sess *session.Session,
	toolCall tools.ToolCall,
	tool tools.Tool,
	events chan Event,
	a *agent.Agent,
	runTool func(),
	remainingCalls []tools.ToolCall,
) (canceled bool) {
	toolName := toolCall.Function.Name

	if e.permissions != nil {
		var toolArgs map[string]any
		if toolCall.Function.Arguments != "" {
			if err := json.Unmarshal([]byte(toolCall.Function.Arguments), &toolArgs); err != nil {
				slog.Debug("Failed to parse tool arguments for permission check", "tool", toolName, "error", err)
			}
		}

		decision := e.permissions.CheckWithArgs(toolName, toolArgs)
		switch decision {
		case permissions.Deny:
			slog.Debug("Tool denied by permissions config", "tool", toolName, "session_id", sess.ID)
			e.addToolErrorResponse(ctx, sess, toolCall, tool, events, a, fmt.Sprintf("Tool '%s' is denied by permissions configuration.", toolName))
			return false
		case permissions.Allow:
			slog.Debug("Tool auto-approved by permissions config", "tool", toolName, "session_id", sess.ID)
			runTool()
			return false
		case permissions.Ask:
			// Fall through to normal approval flow
		}
	}

	if sess.ToolsApproved || tool.Annotations.ReadOnlyHint {
		runTool()
		return false
	}

	slog.Debug("Tools not approved, waiting for resume", "tool", toolCall.Function.Name, "session_id", sess.ID)
	events <- ToolCallConfirmation(toolCall, tool, a.Name())

	select {
	case cType := <-e.resumeChan:
		switch cType {
		case ResumeTypeApprove:
			slog.Debug("Resume signal received, approving tool", "tool", toolCall.Function.Name, "session_id", sess.ID)
			runTool()
		case ResumeTypeApproveSession:
			slog.Debug("Resume signal received, approving session", "tool", toolCall.Function.Name, "session_id", sess.ID)
			sess.ToolsApproved = true
			runTool()
		case ResumeTypeReject:
			slog.Debug("Resume signal received, rejecting tool", "tool", toolCall.Function.Name, "session_id", sess.ID)
			e.addToolErrorResponse(ctx, sess, toolCall, tool, events, a, "The user rejected the tool call.")
		}
		return false
	case <-ctx.Done():
		slog.Debug("Context cancelled while waiting for resume", "tool", toolCall.Function.Name, "session_id", sess.ID)
		e.addToolErrorResponse(ctx, sess, toolCall, tool, events, a, "The tool call was canceled by the user.")
		for _, remainingCall := range remainingCalls {
			e.addToolErrorResponse(ctx, sess, remainingCall, tool, events, a, "The tool call was canceled by the user.")
		}
		return true
	}
}

// executeToolWithHandler handles tool execution, error handling, event emission, and session updates.
func (e *toolExecutor) executeToolWithHandler(
	ctx context.Context,
	toolCall tools.ToolCall,
	tool tools.Tool,
	events chan Event,
	sess *session.Session,
	a *agent.Agent,
	spanName string,
	execute func(ctx context.Context) (*tools.ToolCallResult, time.Duration, error),
) {
	ctx, span := e.startSpan(ctx, spanName, trace.WithAttributes(
		attribute.String("tool.name", toolCall.Function.Name),
		attribute.String("agent", a.Name()),
		attribute.String("session.id", sess.ID),
		attribute.String("tool.call_id", toolCall.ID),
	))
	defer span.End()

	events <- ToolCall(toolCall, tool, a.Name())

	res, duration, err := execute(ctx)

	telemetry.RecordToolCall(ctx, toolCall.Function.Name, sess.ID, a.Name(), duration, err)

	if err != nil {
		if errors.Is(err, context.Canceled) || errors.Is(ctx.Err(), context.Canceled) {
			slog.Debug("Tool handler canceled by context", "tool", toolCall.Function.Name, "agent", a.Name(), "session_id", sess.ID)
			res = tools.ResultError("The tool call was canceled by the user.")
			span.SetStatus(codes.Ok, "tool handler canceled by user")
		} else {
			span.RecordError(err)
			span.SetStatus(codes.Error, "tool handler error")
			slog.Error("Error calling tool", "tool", toolCall.Function.Name, "error", err)
			res = tools.ResultError(fmt.Sprintf("Error calling tool: %v", err))
		}
	} else {
		span.SetStatus(codes.Ok, "tool handler completed")
		slog.Debug("Tool call completed", "tool", toolCall.Function.Name, "output_length", len(res.Output))
	}

	events <- ToolCallResponse(toolCall, tool, res, res.Output, a.Name())

	content := res.Output
	if strings.TrimSpace(content) == "" {
		content = "(no output)"
	}

	toolResponseMsg := chat.Message{
		Role:       chat.MessageRoleTool,
		Content:    content,
		ToolCallID: toolCall.ID,
		CreatedAt:  time.Now().Format(time.RFC3339),
	}
	sess.AddMessage(session.NewAgentMessage(a, &toolResponseMsg))
	_ = e.sessionStore.UpdateSession(ctx, sess)
}

// runTool executes agent tools from toolsets (MCP, filesystem, etc.).
func (e *toolExecutor) runTool(ctx context.Context, tool tools.Tool, toolCall tools.ToolCall, events chan Event, sess *session.Session, a *agent.Agent) {
	hooksExec := e.getHooksExecutor(a)

	if hooksExec != nil && hooksExec.HasPreToolUseHooks() {
		toolInput := parseToolInput(toolCall.Function.Arguments)
		input := &hooks.Input{
			SessionID: sess.ID,
			Cwd:       e.workingDir,
			ToolName:  toolCall.Function.Name,
			ToolUseID: toolCall.ID,
			ToolInput: toolInput,
		}

		result, err := hooksExec.ExecutePreToolUse(ctx, input)
		switch {
		case err != nil:
			slog.Warn("Pre-tool hook execution failed", "tool", toolCall.Function.Name, "error", err)
		case !result.Allowed:
			slog.Debug("Pre-tool hook blocked tool call", "tool", toolCall.Function.Name, "message", result.Message)
			events <- HookBlocked(toolCall, tool, result.Message, a.Name())
			e.addToolErrorResponse(ctx, sess, toolCall, tool, events, a, "Tool call blocked by hook: "+result.Message)
			return
		case result.SystemMessage != "":
			events <- Warning(result.SystemMessage, a.Name())
		}
	}

	e.executeToolWithHandler(ctx, toolCall, tool, events, sess, a, "runtime.tool.handler",
		func(ctx context.Context) (*tools.ToolCallResult, time.Duration, error) {
			res, err := tool.Handler(ctx, toolCall)
			return res, 0, err
		})

	if hooksExec != nil && hooksExec.HasPostToolUseHooks() {
		toolInput := parseToolInput(toolCall.Function.Arguments)
		input := &hooks.Input{
			SessionID:    sess.ID,
			Cwd:          e.workingDir,
			ToolName:     toolCall.Function.Name,
			ToolUseID:    toolCall.ID,
			ToolInput:    toolInput,
			ToolResponse: nil,
		}

		result, err := hooksExec.ExecutePostToolUse(ctx, input)
		if err != nil {
			slog.Warn("Post-tool hook execution failed", "tool", toolCall.Function.Name, "error", err)
		} else if result.SystemMessage != "" {
			events <- Warning(result.SystemMessage, a.Name())
		}
	}
}

func (e *toolExecutor) runAgentTool(ctx context.Context, handler ToolHandlerFunc, sess *session.Session, toolCall tools.ToolCall, tool tools.Tool, events chan Event, a *agent.Agent) {
	e.executeToolWithHandler(ctx, toolCall, tool, events, sess, a, "runtime.tool.handler.runtime",
		func(ctx context.Context) (*tools.ToolCallResult, time.Duration, error) {
			start := time.Now()
			res, err := handler(ctx, sess, toolCall, events)
			return res, time.Since(start), err
		})
}

// addToolErrorResponse adds a tool error response to the session and emits the event.
func (e *toolExecutor) addToolErrorResponse(ctx context.Context, sess *session.Session, toolCall tools.ToolCall, tool tools.Tool, events chan Event, a *agent.Agent, errorMsg string) {
	events <- ToolCallResponse(toolCall, tool, tools.ResultError(errorMsg), errorMsg, a.Name())

	toolResponseMsg := chat.Message{
		Role:       chat.MessageRoleTool,
		Content:    errorMsg,
		ToolCallID: toolCall.ID,
		CreatedAt:  time.Now().Format(time.RFC3339),
	}
	sess.AddMessage(session.NewAgentMessage(a, &toolResponseMsg))
	_ = e.sessionStore.UpdateSession(ctx, sess)
}

func (e *toolExecutor) getHooksExecutor(a *agent.Agent) *hooks.Executor {
	hooksCfg := hooks.FromConfig(a.Hooks())
	if hooksCfg == nil || hooksCfg.IsEmpty() {
		return nil
	}
	return hooks.NewExecutor(hooksCfg, e.workingDir, e.env)
}

func (e *toolExecutor) startSpan(ctx context.Context, name string, opts ...trace.SpanStartOption) (context.Context, trace.Span) {
	if e.tracer == nil {
		return ctx, trace.SpanFromContext(ctx)
	}
	return e.tracer.Start(ctx, name, opts...)
}

func parseToolInput(arguments string) map[string]any {
	var result map[string]any
	if err := json.Unmarshal([]byte(arguments), &result); err != nil {
		return nil
	}
	return result
}
