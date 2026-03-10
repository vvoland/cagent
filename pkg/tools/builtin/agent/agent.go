package agent

import (
	"context"
	"encoding/json"
	"fmt"
	"log/slog"
	"slices"
	"strings"
	"sync"
	"sync/atomic"
	"time"

	"github.com/google/uuid"

	"github.com/docker/docker-agent/pkg/concurrent"
	"github.com/docker/docker-agent/pkg/session"
	"github.com/docker/docker-agent/pkg/tools"
)

const (
	ToolNameRunBackgroundAgent   = "run_background_agent"
	ToolNameListBackgroundAgents = "list_background_agents"
	ToolNameViewBackgroundAgent  = "view_background_agent"
	ToolNameStopBackgroundAgent  = "stop_background_agent"
)

const (
	// maxConcurrentTasks is the maximum number of simultaneously running background agent tasks.
	maxConcurrentTasks = 20
	// maxTotalTasks caps total stored tasks (running + completed) to prevent unbounded memory growth.
	maxTotalTasks = 100
	// maxOutputBytes caps the live output buffer per task, mirroring the shell tool's limit.
	maxOutputBytes = 10 * 1024 * 1024 // 10 MB
)

// RunBackgroundAgentArgs specifies the parameters for dispatching a sub-agent task asynchronously.
type RunBackgroundAgentArgs struct {
	Agent          string `json:"agent" jsonschema:"The name of the sub-agent to run in the background."`
	Task           string `json:"task" jsonschema:"A clear and concise description of the task the agent should achieve."`
	ExpectedOutput string `json:"expected_output,omitempty" jsonschema:"The expected output from the agent (optional)."`
}

// ViewBackgroundAgentArgs specifies the task ID to inspect.
type ViewBackgroundAgentArgs struct {
	TaskID string `json:"task_id" jsonschema:"The ID of the background agent task to view."`
}

// StopBackgroundAgentArgs specifies the task ID to cancel.
type StopBackgroundAgentArgs struct {
	TaskID string `json:"task_id" jsonschema:"The ID of the background agent task to stop."`
}

// RunParams holds the parameters for running a sub-agent.
type RunParams struct {
	AgentName      string
	Task           string
	ExpectedOutput string
	ParentSession  *session.Session
	OnContent      func(content string)
}

// RunResult holds the outcome of a sub-agent execution.
type RunResult struct {
	Result string // final assistant message on completion
	ErrMsg string // error detail if failed
}

// Runner abstracts the runtime dependency for background agent execution.
type Runner interface {
	// CurrentAgentSubAgentNames returns the names of the current agent's sub-agents.
	CurrentAgentSubAgentNames() []string
	// RunAgent starts a sub-agent and blocks until completion or cancellation.
	RunAgent(ctx context.Context, params RunParams) *RunResult
}

// taskStatus represents the lifecycle state of a background agent task.
type taskStatus int32

const (
	taskRunning taskStatus = iota
	taskCompleted
	taskStopped
	taskFailed
)

var taskStatusStrings = map[taskStatus]string{
	taskRunning:   "running",
	taskCompleted: "completed",
	taskStopped:   "stopped",
	taskFailed:    "failed",
}

func statusToString(s taskStatus) string {
	if str, ok := taskStatusStrings[s]; ok {
		return str
	}
	return "unknown"
}

// task tracks a single background sub-agent execution.
type task struct {
	id        string
	agentName string
	taskDesc  string

	cancel      context.CancelFunc
	outputMu    sync.RWMutex
	output      strings.Builder
	outputBytes int
	startTime   time.Time
	status      atomic.Int32
	result      string
	errMsg      string
}

func (t *task) loadStatus() taskStatus {
	return taskStatus(t.status.Load())
}

func (t *task) storeStatus(s taskStatus) {
	t.status.Store(int32(s))
}

func (t *task) casStatus(old, next taskStatus) bool {
	return t.status.CompareAndSwap(int32(old), int32(next))
}

// Handler owns all background agent tasks and provides tool handlers.
type Handler struct {
	runner Runner
	wg     sync.WaitGroup
	tasks  *concurrent.Map[string, *task]
}

// NewHandler creates a new Handler with the given Runner.
func NewHandler(runner Runner) *Handler {
	return &Handler{
		runner: runner,
		tasks:  concurrent.NewMap[string, *task](),
	}
}

func newTaskID() string {
	return "agent_task_" + uuid.New().String()
}

func (h *Handler) runningTaskCount() int {
	var count int
	h.tasks.Range(func(_ string, t *task) bool {
		if t.loadStatus() == taskRunning {
			count++
		}
		return true
	})
	return count
}

func (h *Handler) totalTaskCount() int {
	return h.tasks.Length()
}

func (h *Handler) pruneCompleted() {
	var toDelete []string
	h.tasks.Range(func(id string, t *task) bool {
		s := t.loadStatus()
		if s == taskCompleted || s == taskStopped || s == taskFailed {
			toDelete = append(toDelete, id)
		}
		return true
	})
	for _, id := range toDelete {
		h.tasks.Delete(id)
	}
}

// HandleRun starts a sub-agent task asynchronously and returns a task ID immediately.
func (h *Handler) HandleRun(ctx context.Context, sess *session.Session, toolCall tools.ToolCall) (*tools.ToolCallResult, error) {
	var params RunBackgroundAgentArgs
	if err := json.Unmarshal([]byte(toolCall.Function.Arguments), &params); err != nil {
		return nil, fmt.Errorf("invalid arguments: %w", err)
	}

	if strings.TrimSpace(params.Agent) == "" {
		return tools.ResultError("agent name must not be empty"), nil
	}
	if strings.TrimSpace(params.Task) == "" {
		return tools.ResultError("task must not be empty"), nil
	}

	subAgentNames := h.runner.CurrentAgentSubAgentNames()
	valid := slices.Contains(subAgentNames, params.Agent)
	if !valid {
		if len(subAgentNames) > 0 {
			return tools.ResultError(fmt.Sprintf("agent %q is not in the sub-agents list. Available: %s", params.Agent, strings.Join(subAgentNames, ", "))), nil
		}
		return tools.ResultError(fmt.Sprintf("agent %q is not in the sub-agents list. This agent has no sub-agents configured.", params.Agent)), nil
	}

	// Enforce concurrency cap.
	if h.runningTaskCount() >= maxConcurrentTasks {
		return tools.ResultError(fmt.Sprintf("maximum concurrent background agent tasks (%d) reached; stop or wait for existing tasks to complete", maxConcurrentTasks)), nil
	}

	// Enforce total cap, pruning finished tasks first.
	if h.totalTaskCount() >= maxTotalTasks {
		h.pruneCompleted()
		if h.totalTaskCount() >= maxTotalTasks {
			return tools.ResultError(fmt.Sprintf("maximum total background agent tasks (%d) reached; view and discard old tasks first", maxTotalTasks)), nil
		}
	}

	taskID := newTaskID()

	taskCtx, cancel := context.WithCancel(ctx)

	t := &task{
		id:        taskID,
		agentName: params.Agent,
		taskDesc:  params.Task,
		cancel:    cancel,
		startTime: time.Now(),
	}
	t.storeStatus(taskRunning)
	h.tasks.Store(taskID, t)

	h.wg.Go(func() {
		defer cancel()

		slog.Debug("Starting background agent task", "task_id", taskID, "agent", params.Agent)

		result := h.runner.RunAgent(taskCtx, RunParams{
			AgentName:      params.Agent,
			Task:           params.Task,
			ExpectedOutput: params.ExpectedOutput,
			ParentSession:  sess,
			OnContent: func(content string) {
				t.outputMu.Lock()
				if t.outputBytes < maxOutputBytes {
					n, _ := t.output.WriteString(content)
					t.outputBytes += n
				}
				t.outputMu.Unlock()
			},
		})

		if result.ErrMsg != "" {
			t.errMsg = result.ErrMsg
			t.storeStatus(taskFailed)
			slog.Debug("Background agent task failed", "task_id", taskID, "agent", params.Agent, "error", result.ErrMsg)
			return
		}

		if taskCtx.Err() != nil && t.loadStatus() == taskRunning {
			t.storeStatus(taskStopped)
			slog.Debug("Background agent task stopped", "task_id", taskID)
			return
		}

		// Write result before CAS so readers who observe taskCompleted
		// always see the populated result field.
		t.result = result.Result
		if t.casStatus(taskRunning, taskCompleted) {
			slog.Debug("Background agent task completed", "task_id", taskID, "agent", params.Agent)
		}
	})

	return tools.ResultSuccess(fmt.Sprintf("Background agent task started with ID: %s\nAgent: %s\nTask: %s",
		taskID, params.Agent, params.Task)), nil
}

// HandleList lists all background agent tasks.
func (h *Handler) HandleList(_ context.Context, _ *session.Session, _ tools.ToolCall) (*tools.ToolCallResult, error) {
	var out strings.Builder
	out.WriteString("Background Agent Tasks:\n\n")

	var count int
	h.tasks.Range(func(_ string, t *task) bool {
		count++
		status := t.loadStatus()
		elapsed := time.Since(t.startTime).Round(time.Second)
		fmt.Fprintf(&out, "ID: %s\n", t.id)
		fmt.Fprintf(&out, "  Agent:   %s\n", t.agentName)
		fmt.Fprintf(&out, "  Status:  %s\n", statusToString(status))
		fmt.Fprintf(&out, "  Runtime: %s\n", elapsed)
		out.WriteString("\n")
		return true
	})

	if count == 0 {
		out.WriteString("No background agent tasks found.\n")
	}

	return tools.ResultSuccess(out.String()), nil
}

// HandleView returns the output and status of a specific background agent task.
func (h *Handler) HandleView(_ context.Context, _ *session.Session, toolCall tools.ToolCall) (*tools.ToolCallResult, error) {
	var params ViewBackgroundAgentArgs
	if err := json.Unmarshal([]byte(toolCall.Function.Arguments), &params); err != nil {
		return nil, fmt.Errorf("invalid arguments: %w", err)
	}

	t, exists := h.tasks.Load(params.TaskID)
	if !exists {
		return tools.ResultError("task not found: " + params.TaskID), nil
	}

	status := t.loadStatus()
	elapsed := time.Since(t.startTime).Round(time.Second)

	var out strings.Builder
	fmt.Fprintf(&out, "Task ID: %s\n", t.id)
	fmt.Fprintf(&out, "Agent:   %s\n", t.agentName)
	fmt.Fprintf(&out, "Status:  %s\n", statusToString(status))
	fmt.Fprintf(&out, "Runtime: %s\n", elapsed)
	out.WriteString("\n--- Output ---\n")

	switch status {
	case taskCompleted:
		if t.result != "" {
			out.WriteString(t.result)
		} else {
			out.WriteString("<no output>")
		}
	case taskFailed:
		out.WriteString("<task failed>")
		if t.errMsg != "" {
			fmt.Fprintf(&out, "\nError: %s", t.errMsg)
		}
	case taskStopped:
		out.WriteString("<task was stopped>")
	default:
		t.outputMu.RLock()
		progress := t.output.String()
		truncated := t.outputBytes >= maxOutputBytes
		t.outputMu.RUnlock()
		if progress != "" {
			out.WriteString(progress)
			if truncated {
				out.WriteString("\n\n[output truncated at 10MB limit — still running...]")
			} else {
				out.WriteString("\n\n[still running...]")
			}
		} else {
			out.WriteString("<no output yet — still running>")
		}
	}

	return tools.ResultSuccess(out.String()), nil
}

// HandleStop cancels a running background agent task.
func (h *Handler) HandleStop(_ context.Context, _ *session.Session, toolCall tools.ToolCall) (*tools.ToolCallResult, error) {
	var params StopBackgroundAgentArgs
	if err := json.Unmarshal([]byte(toolCall.Function.Arguments), &params); err != nil {
		return nil, fmt.Errorf("invalid arguments: %w", err)
	}

	t, exists := h.tasks.Load(params.TaskID)
	if !exists {
		return tools.ResultError("task not found: " + params.TaskID), nil
	}

	if !t.casStatus(taskRunning, taskStopped) {
		current := t.loadStatus()
		return tools.ResultError(fmt.Sprintf("task %s is not running (status: %s)", params.TaskID, statusToString(current))), nil
	}

	t.cancel()

	return tools.ResultSuccess(fmt.Sprintf("Background agent task %s stopped.", params.TaskID)), nil
}

// StopAll cancels all running tasks and waits for their goroutines to exit.
// Called during runtime shutdown to ensure clean teardown.
func (h *Handler) StopAll() {
	h.tasks.Range(func(_ string, t *task) bool {
		if t.casStatus(taskRunning, taskStopped) {
			t.cancel()
		}
		return true
	})
	h.wg.Wait()
}

// toolSet is a lightweight ToolSet that returns just the tool definitions
// without requiring a Runner. Used by teamloader to register tool schemas.
type toolSet struct{}

// NewToolSet returns a ToolSet for registering background agent tool definitions.
// This does not require a Runner and is suitable for use in teamloader.
func NewToolSet() tools.ToolSet {
	return &toolSet{}
}

func (t *toolSet) Tools(ctx context.Context) ([]tools.Tool, error) {
	return backgroundAgentTools()
}

// Tools returns the four background agent tool definitions.
func (h *Handler) Tools(ctx context.Context) ([]tools.Tool, error) {
	return backgroundAgentTools()
}

func backgroundAgentTools() ([]tools.Tool, error) {
	return []tools.Tool{
		{
			Name:     ToolNameRunBackgroundAgent,
			Category: "transfer",
			Description: `Start a sub-agent task in the background and return immediately with a task ID.
Use this to dispatch work to multiple sub-agents concurrently. The sub-agent runs with all tools
pre-approved — use only with trusted sub-agents and well-scoped tasks. Check progress with
view_background_agent and collect results once the task is complete.`,
			Parameters:  tools.MustSchemaFor[RunBackgroundAgentArgs](),
			Annotations: tools.ToolAnnotations{Title: "Run Background Agent"},
		},
		{
			Name:        ToolNameListBackgroundAgents,
			Category:    "transfer",
			Description: `List all background agent tasks with their status and runtime.`,
			Annotations: tools.ToolAnnotations{
				Title:        "List Background Agents",
				ReadOnlyHint: true,
			},
		},
		{
			Name:        ToolNameViewBackgroundAgent,
			Category:    "transfer",
			Description: `View the output and status of a specific background agent task by task ID. Returns live buffered output if still running, or the final result if complete.`,
			Parameters:  tools.MustSchemaFor[ViewBackgroundAgentArgs](),
			Annotations: tools.ToolAnnotations{
				Title:        "View Background Agent",
				ReadOnlyHint: true,
			},
		},
		{
			Name:        ToolNameStopBackgroundAgent,
			Category:    "transfer",
			Description: `Stop a running background agent task by task ID.`,
			Parameters:  tools.MustSchemaFor[StopBackgroundAgentArgs](),
			Annotations: tools.ToolAnnotations{
				Title: "Stop Background Agent",
			},
		},
	}, nil
}
