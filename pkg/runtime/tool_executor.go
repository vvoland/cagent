package runtime

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"log/slog"
	"slices"
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
	"github.com/docker/cagent/pkg/team"
	"github.com/docker/cagent/pkg/telemetry"
	"github.com/docker/cagent/pkg/tools"
	"github.com/docker/cagent/pkg/tools/builtin"
)

type toolExecutor struct {
	tracer          trace.Tracer
	sessionStore    session.Store
	resumeChan      chan ResumeType
	toolMap         map[string]ToolHandler
	permissions     *permissions.Checker
	workingDir      string
	env             []string
	team            *team.Team
	getCurrentAgent func() string
	setCurrentAgent func(string)
}

type toolExecutorConfig struct {
	tracer          trace.Tracer
	sessionStore    session.Store
	resumeChan      chan ResumeType
	permissions     *permissions.Checker
	workingDir      string
	env             []string
	team            *team.Team
	getCurrentAgent func() string
	setCurrentAgent func(string)
}

func newToolExecutor(cfg toolExecutorConfig) *toolExecutor {
	te := &toolExecutor{
		tracer:          cfg.tracer,
		sessionStore:    cfg.sessionStore,
		resumeChan:      cfg.resumeChan,
		toolMap:         make(map[string]ToolHandler),
		permissions:     cfg.permissions,
		workingDir:      cfg.workingDir,
		env:             cfg.env,
		team:            cfg.team,
		getCurrentAgent: cfg.getCurrentAgent,
		setCurrentAgent: cfg.setCurrentAgent,
	}
	te.registerAgentTools()
	return te
}

func (e *toolExecutor) registerAgentTools() {
	slog.Debug("Registering agent tools")

	tt := builtin.NewTransferTaskTool()
	ht := builtin.NewHandoffTool()
	ttTools, _ := tt.Tools(context.TODO())
	htTools, _ := ht.Tools(context.TODO())
	allTools := append(ttTools, htTools...)

	handlers := map[string]ToolHandlerFunc{
		builtin.ToolNameTransferTask: e.handleTaskTransfer,
		builtin.ToolNameHandoff:      e.handleHandoff,
	}

	for _, t := range allTools {
		if h, exists := handlers[t.Name]; exists {
			e.toolMap[t.Name] = ToolHandler{handler: h, tool: t}
		} else {
			slog.Warn("No handler found for agent tool", "tool", t.Name)
		}
	}

	slog.Debug("Registered agent tools", "count", len(handlers))
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

func (e *toolExecutor) handleTaskTransfer(ctx context.Context, sess *session.Session, toolCall tools.ToolCall, evts chan Event) (*tools.ToolCallResult, error) {
	var params builtin.TransferTaskArgs
	if err := json.Unmarshal([]byte(toolCall.Function.Arguments), &params); err != nil {
		return nil, fmt.Errorf("invalid arguments: %w", err)
	}

	currentAgentName := e.getCurrentAgent()
	a, err := e.team.Agent(currentAgentName)
	if err != nil {
		return nil, fmt.Errorf("current agent not found: %w", err)
	}

	ctx, span := e.startSpan(ctx, "runtime.task_transfer", trace.WithAttributes(
		attribute.String("from.agent", a.Name()),
		attribute.String("to.agent", params.Agent),
		attribute.String("session.id", sess.ID),
	))
	defer span.End()

	slog.Debug("Transferring task to agent", "from_agent", a.Name(), "to_agent", params.Agent, "task", params.Task)

	// Emit agent switching start event
	evts <- AgentSwitching(true, currentAgentName, params.Agent)

	e.setCurrentAgent(params.Agent)
	defer func() {
		e.setCurrentAgent(currentAgentName)

		// Emit agent switching end event
		evts <- AgentSwitching(false, params.Agent, currentAgentName)

		// Restore original agent info in sidebar
		if originalAgent, err := e.team.Agent(currentAgentName); err == nil {
			evts <- AgentInfo(originalAgent.Name(), getAgentModelID(originalAgent), originalAgent.Description(), originalAgent.WelcomeMessage())
		}
	}()

	// Emit agent info for the new agent
	if newAgent, err := e.team.Agent(params.Agent); err == nil {
		evts <- AgentInfo(newAgent.Name(), getAgentModelID(newAgent), newAgent.Description(), newAgent.WelcomeMessage())
	}

	memberAgentTask := "You are a member of a team of agents. Your goal is to complete the following task:"
	memberAgentTask += fmt.Sprintf("\n\n<task>\n%s\n</task>", params.Task)
	if params.ExpectedOutput != "" {
		memberAgentTask += fmt.Sprintf("\n\n<expected_output>\n%s\n</expected_output>", params.ExpectedOutput)
	}

	slog.Debug("Creating new session with parent session", "parent_session_id", sess.ID, "tools_approved", sess.ToolsApproved)

	child, err := e.team.Agent(params.Agent)
	if err != nil {
		return nil, err
	}

	s := session.New(
		session.WithSystemMessage(memberAgentTask),
		session.WithImplicitUserMessage("Please proceed."),
		session.WithMaxIterations(child.MaxIterations()),
		session.WithTitle("Transferred task"),
		session.WithToolsApproved(sess.ToolsApproved),
		session.WithSendUserMessage(false),
	)

	childRuntime, err := New(e.team, WithCurrentAgent(params.Agent), WithTracer(e.tracer))
	if err != nil {
		span.RecordError(err)
		span.SetStatus(codes.Error, "failed to create child runtime")
		return nil, fmt.Errorf("failed to create child runtime: %w", err)
	}

	for event := range childRuntime.RunStream(ctx, s) {
		evts <- event
		if errEvent, ok := event.(*ErrorEvent); ok {
			span.RecordError(fmt.Errorf("%s", errEvent.Error))
			span.SetStatus(codes.Error, "error in transferred session")
			return nil, fmt.Errorf("%s", errEvent.Error)
		}
	}

	sess.ToolsApproved = s.ToolsApproved
	sess.AddSubSession(s)

	slog.Debug("Task transfer completed", "agent", params.Agent, "task", params.Task)

	span.SetStatus(codes.Ok, "task transfer completed")
	return tools.ResultSuccess(s.GetLastAssistantMessageContent()), nil
}

func (e *toolExecutor) handleHandoff(_ context.Context, _ *session.Session, toolCall tools.ToolCall, _ chan Event) (*tools.ToolCallResult, error) {
	var params builtin.HandoffArgs
	if err := json.Unmarshal([]byte(toolCall.Function.Arguments), &params); err != nil {
		return nil, fmt.Errorf("invalid arguments: %w", err)
	}

	currentAgentName := e.getCurrentAgent()
	currentAgent, err := e.team.Agent(currentAgentName)
	if err != nil {
		return nil, fmt.Errorf("current agent not found: %w", err)
	}

	// Validate that the target agent is in the current agent's handoffs list
	handoffs := currentAgent.Handoffs()
	if !slices.ContainsFunc(handoffs, func(a *agent.Agent) bool { return a.Name() == params.Agent }) {
		var handoffNames []string
		for _, h := range handoffs {
			handoffNames = append(handoffNames, h.Name())
		}
		var errorMsg string
		if len(handoffNames) > 0 {
			errorMsg = fmt.Sprintf("Agent %s cannot hand off to %s: target agent not in handoffs list. Available handoff agent IDs are: %s", currentAgentName, params.Agent, strings.Join(handoffNames, ", "))
		} else {
			errorMsg = fmt.Sprintf("Agent %s cannot hand off to %s: target agent not in handoffs list. This agent has no handoff agents configured.", currentAgentName, params.Agent)
		}
		return tools.ResultError(errorMsg), nil
	}

	next, err := e.team.Agent(params.Agent)
	if err != nil {
		return nil, err
	}

	e.setCurrentAgent(next.Name())
	handoffMessage := "The agent " + currentAgentName + " handed off the conversation to you. " +
		"Your available handoff agents and tools are specified in the system messages that follow. " +
		"Only use those capabilities - do not attempt to use tools or hand off to agents that you see " +
		"in the conversation history from previous agents, as those were available to different agents " +
		"with different capabilities. Look at the conversation history for context, but only use the " +
		"handoff agents and tools that are listed in your system messages below. " +
		"Complete your part of the task and hand off to the next appropriate agent in your workflow " +
		"(if any are available to you), or respond directly to the user if you are the final agent."
	return tools.ResultSuccess(handoffMessage), nil
}
