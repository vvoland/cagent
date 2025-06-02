package config

// Tool represents a tool configuration
type Tool struct {
	Type    string            `yaml:"type,omitempty"`
	Command string            `yaml:"command,omitempty"`
	Args    []string          `yaml:"args,omitempty"`
	Env     map[string]string `yaml:"env,omitempty"`
}

// AgentConfig represents a single agent configuration
type AgentConfig struct {
	Name        string   `yaml:"name"`
	Model       string   `yaml:"model"`
	Description string   `yaml:"description"`
	Tools       []Tool   `yaml:"tools"`
	Instruction string   `yaml:"instruction"`
	SubAgents   []string `yaml:"sub_agents,omitempty"`
	AddDate     bool     `yaml:"add_date,omitempty"`
	Think       bool     `yaml:"think,omitempty"`
}

// ModelConfig represents the configuration for a model
type ModelConfig struct {
	Type             string  `yaml:"type"`
	Model            string  `yaml:"model"`
	Temperature      float64 `yaml:"temperature"`
	MaxTokens        int     `yaml:"max_tokens"`
	TopP             float64 `yaml:"top_p"`
	FrequencyPenalty float64 `yaml:"frequency_penalty"`
	PresencePenalty  float64 `yaml:"presence_penalty"`
}

// Config represents the entire configuration file
type Config struct {
	Agents map[string]AgentConfig `yaml:"agents"`
	Models map[string]ModelConfig `yaml:"models"`
}
