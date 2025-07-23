package config

// Toolset represents a tool configuration
type Toolset struct {
	Type     string            `yaml:"type,omitempty"`
	Command  string            `yaml:"command,omitempty"`
	Args     []string          `yaml:"args,omitempty"`
	Env      map[string]string `yaml:"env,omitempty"`
	Envfiles StringOrList      `yaml:"env_file,omitempty"`
	Tools    []string          `yaml:"tools,omitempty"`
}

// TodoConfig represents todo configuration that can be either a boolean or an object
type TodoConfig struct {
	Enabled bool `yaml:"-"`
	Shared  bool `yaml:"shared,omitempty"`
}

// UnmarshalYAML implements custom unmarshaling for TodoConfig to support both boolean and object formats
func (t *TodoConfig) UnmarshalYAML(unmarshal func(any) error) error {
	type todoConfigAlias TodoConfig

	var config todoConfigAlias
	if err := unmarshal(&config); err == nil {
		*t = TodoConfig(config)
		t.Enabled = true
		return nil
	}

	var enabled bool
	if err := unmarshal(&enabled); err != nil {
		return err
	}

	t.Enabled = enabled

	return nil
}

// AgentConfig represents a single agent configuration
type AgentConfig struct {
	Name         string       `yaml:"name,omitempty"`
	Model        string       `yaml:"model,omitempty"`
	Description  string       `yaml:"description,omitempty"`
	Toolsets     []Toolset    `yaml:"toolsets,omitempty"`
	Instruction  string       `yaml:"instruction,omitempty"`
	SubAgents    []string     `yaml:"sub_agents,omitempty"`
	AddDate      bool         `yaml:"add_date,omitempty"`
	Think        bool         `yaml:"think,omitempty"`
	Todo         TodoConfig   `yaml:"todo,omitempty"`
	MemoryConfig MemoryConfig `yaml:"memory,omitempty"`
}

type MemoryConfig struct {
	Path string `yaml:"path,omitempty"`
}

// ModelConfig represents the configuration for a model
type ModelConfig struct {
	Type              string  `yaml:"type,omitempty"`
	Model             string  `yaml:"model,omitempty"`
	Temperature       float64 `yaml:"temperature,omitempty"`
	MaxTokens         int     `yaml:"max_tokens,omitempty"`
	TopP              float64 `yaml:"top_p,omitempty"`
	FrequencyPenalty  float64 `yaml:"frequency_penalty,omitempty"`
	PresencePenalty   float64 `yaml:"presence_penalty,omitempty"`
	BaseURL           string  `yaml:"base_url,omitempty"`
	ParallelToolCalls *bool   `yaml:"parallel_tool_calls,omitempty"`
}

// Config represents the entire configuration file
type Config struct {
	Agents map[string]AgentConfig `yaml:"agents,omitempty"`
	Models map[string]ModelConfig `yaml:"models,omitempty"`
}

type StringOrList []string

func (sm *StringOrList) UnmarshalYAML(unmarshal func(any) error) error {
	var multi []string
	if err := unmarshal(&multi); err != nil {
		var single string
		if err := unmarshal(&single); err != nil {
			return err
		}

		*sm = []string{single}
		return nil
	}

	*sm = multi
	return nil
}
