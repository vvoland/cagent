package agent

import "github.com/rumpl/cagent/pkg/tools"

type AgentOpt func(a *Agent)

func WithInstruction(prompt string) AgentOpt {
	return func(a *Agent) {
		a.instruction = prompt
	}
}

func WithToolSet(toolSet []tools.ToolSet) AgentOpt {
	return func(a *Agent) {
		a.toolimpl = toolSet
	}
}

func WithDescription(description string) AgentOpt {
	return func(a *Agent) {
		a.description = description
	}
}

func WithName(name string) AgentOpt {
	return func(a *Agent) {
		a.name = name
	}
}

func WithModel(model string) AgentOpt {
	return func(a *Agent) {
		a.model = model
	}
}

func WithSubAgents(subAgents []*Agent) AgentOpt {
	return func(a *Agent) {
		a.subAgents = subAgents
		for _, subAgent := range subAgents {
			subAgent.parents = append(subAgent.parents, a)
		}
	}
}

func WithAddDate(addDate bool) AgentOpt {
	return func(a *Agent) {
		a.addDate = addDate
	}
}
