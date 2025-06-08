package agent

import (
	"github.com/rumpl/cagent/pkg/model/provider"
	"github.com/rumpl/cagent/pkg/tools"
)

type Opt func(a *Agent)

func WithInstruction(prompt string) Opt {
	return func(a *Agent) {
		a.instruction = prompt
	}
}

func WithToolSets(toolSet []tools.ToolSet) Opt {
	return func(a *Agent) {
		a.toolimpl = toolSet
	}
}

func WithTools(tools []tools.Tool) Opt {
	return func(a *Agent) {
		a.toolwrapper = toolwrapper{
			allTools: tools,
		}
	}
}

func WithDescription(description string) Opt {
	return func(a *Agent) {
		a.description = description
	}
}

func WithName(name string) Opt {
	return func(a *Agent) {
		a.name = name
	}
}

func WithModel(model provider.Provider) Opt {
	return func(a *Agent) {
		a.model = model
	}
}

func WithSubAgents(subAgents []*Agent) Opt {
	return func(a *Agent) {
		a.subAgents = subAgents
		for _, subAgent := range subAgents {
			subAgent.parents = append(subAgent.parents, a)
		}
	}
}

func WithAddDate(addDate bool) Opt {
	return func(a *Agent) {
		a.addDate = addDate
	}
}
