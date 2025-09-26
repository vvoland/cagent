package v2

import (
	"errors"
	"strings"
)

func (t *Config) UnmarshalYAML(unmarshal func(any) error) error {
	type alias Config
	var tmp alias
	if err := unmarshal(&tmp); err != nil {
		return err
	}
	*t = Config(tmp)
	return t.validate()
}

func (t *Config) validate() error {
	for _, agent := range t.Agents {
		for _, toolSet := range agent.Toolsets {
			if err := toolSet.validate(); err != nil {
				return err
			}
		}
	}

	return nil
}

// Ensure that either Command, Remote or Ref is set, but not all empty
func (t *Toolset) validate() error {
	if len(t.Shell) > 0 && t.Type != "script" {
		return errors.New("shell can only be used with type 'script'")
	}
	if t.Type != "mcp" {
		return nil
	}

	count := 0
	if t.Command != "" {
		count++
	}
	if t.Remote.URL != "" {
		count++
	}
	if t.Ref != "" {
		count++
	}
	if count == 0 {
		return errors.New("either command, remote or ref must be set")
	}
	if count > 1 {
		return errors.New("either command, remote or ref must be set, but only one of those")
	}

	if t.Ref != "" && !strings.Contains(t.Ref, "docker:") {
		return errors.New("only docker refs are supported for MCP tools, e.g., 'docker:context7'")
	}

	return nil
}
