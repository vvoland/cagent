package agent

import (
	"context"
	"sync"

	"github.com/docker/cagent/pkg/config/latest"
	"github.com/docker/cagent/pkg/config/types"
	"github.com/docker/cagent/pkg/model/provider"
	"github.com/docker/cagent/pkg/tools"
)

type Opt func(a *Agent)

func WithInstruction(instruction string) Opt {
	return func(a *Agent) {
		a.instruction = instruction
	}
}

func WithToolSets(toolSet ...tools.ToolSet) Opt {
	var startableToolSet []*StartableToolSet
	for _, ts := range toolSet {
		startableToolSet = append(startableToolSet, &StartableToolSet{
			ToolSet: ts,
		})
	}

	return func(a *Agent) {
		a.toolsets = startableToolSet
	}
}

func WithTools(allTools ...tools.Tool) Opt {
	return func(a *Agent) {
		a.tools = allTools
	}
}

func WithDescription(description string) Opt {
	return func(a *Agent) {
		a.description = description
	}
}

func WithWelcomeMessage(welcomeMessage string) Opt {
	return func(a *Agent) {
		a.welcomeMessage = welcomeMessage
	}
}

func WithName(name string) Opt {
	return func(a *Agent) {
		a.name = name
	}
}

func WithModel(model provider.Provider) Opt {
	return func(a *Agent) {
		a.models = append(a.models, model)
	}
}

func WithSubAgents(subAgents ...*Agent) Opt {
	return func(a *Agent) {
		a.subAgents = subAgents
		for _, subAgent := range subAgents {
			subAgent.parents = append(subAgent.parents, a)
		}
	}
}

func WithHandoffs(handoffs ...*Agent) Opt {
	return func(a *Agent) {
		a.handoffs = handoffs
	}
}

func WithAddDate(addDate bool) Opt {
	return func(a *Agent) {
		a.addDate = addDate
	}
}

func WithAddEnvironmentInfo(addEnvironmentInfo bool) Opt {
	return func(a *Agent) {
		a.addEnvironmentInfo = addEnvironmentInfo
	}
}

func WithAddPromptFiles(addPromptFiles []string) Opt {
	return func(a *Agent) {
		a.addPromptFiles = addPromptFiles
	}
}

func WithMaxIterations(maxIterations int) Opt {
	return func(a *Agent) {
		a.maxIterations = maxIterations
	}
}

func WithNumHistoryItems(numHistoryItems int) Opt {
	return func(a *Agent) {
		a.numHistoryItems = numHistoryItems
	}
}

func WithCommands(commands types.Commands) Opt {
	return func(a *Agent) {
		a.commands = commands
	}
}

func WithLoadTimeWarnings(warnings []string) Opt {
	return func(a *Agent) {
		for _, w := range warnings {
			a.addToolWarning(w)
		}
	}
}

func WithSkillsEnabled(enabled bool) Opt {
	return func(a *Agent) {
		a.skillsEnabled = enabled
	}
}

func WithHooks(hooks *latest.HooksConfig) Opt {
	return func(a *Agent) {
		a.hooks = hooks
	}
}

type StartableToolSet struct {
	tools.ToolSet
	startOnce sync.Once
	started   bool
	startErr  error
}

// Start starts the toolset exactly once. Concurrent callers will block until
// the first call completes. Returns the error from the first start attempt.
func (s *StartableToolSet) Start(ctx context.Context) error {
	s.startOnce.Do(func() {
		s.startErr = s.ToolSet.Start(ctx)
		if s.startErr == nil {
			s.started = true
		}
	})
	return s.startErr
}

// IsStarted returns whether the toolset has been successfully started.
func (s *StartableToolSet) IsStarted() bool {
	return s.started
}
