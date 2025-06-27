package config

// Toolset represents a tool configuration
type Toolset struct {
	Type    string            `yaml:"type,omitempty"`
	Command string            `yaml:"command,omitempty"`
	Args    []string          `yaml:"args,omitempty"`
	Env     map[string]string `yaml:"env,omitempty"`
	Tools   []string          `yaml:"tools,omitempty"`
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
	Todo         bool         `yaml:"todo,omitempty"`
	MemoryConfig MemoryConfig `yaml:"memory,omitempty"`
}

type MemoryConfig struct {
	Path string `yaml:"path,omitempty"`
}

// ModelConfig represents the configuration for a model
type ModelConfig struct {
	Type             string  `yaml:"type,omitempty"`
	Model            string  `yaml:"model,omitempty"`
	Temperature      float64 `yaml:"temperature,omitempty"`
	MaxTokens        int     `yaml:"max_tokens,omitempty"`
	TopP             float64 `yaml:"top_p,omitempty"`
	FrequencyPenalty float64 `yaml:"frequency_penalty,omitempty"`
	PresencePenalty  float64 `yaml:"presence_penalty,omitempty"`
	BaseURL          string  `yaml:"base_url,omitempty"`
}

// Config represents the entire configuration file
type Config struct {
	Agents map[string]AgentConfig `yaml:"agents,omitempty"`
	Models map[string]ModelConfig `yaml:"models,omitempty"`
}
