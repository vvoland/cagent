// Package userconfig provides user-level configuration for cagent.
// This configuration is stored in ~/.config/cagent/config.yaml and contains
// user preferences like aliases.
package userconfig

import (
	"bytes"
	"errors"
	"fmt"
	"log/slog"
	"os"
	"path/filepath"
	"regexp"

	"github.com/goccy/go-yaml"
	"github.com/natefinch/atomic"

	"github.com/docker/cagent/pkg/paths"
)

// Config represents the user-level cagent configuration
type Config struct {
	// Aliases maps alias names to agent paths or OCI references
	Aliases map[string]string `yaml:"aliases,omitempty"`
}

// Path returns the path to the config file
func Path() string {
	return filepath.Join(paths.GetConfigDir(), "config.yaml")
}

// legacyAliasesPath returns the path to the legacy aliases.yaml file
func legacyAliasesPath() string {
	return filepath.Join(paths.GetConfigDir(), "aliases.yaml")
}

// Load loads the user configuration from the config file.
// If the config file doesn't exist but a legacy aliases.yaml does,
// the aliases are migrated to the new config file.
func Load() (*Config, error) {
	return loadFrom(Path(), legacyAliasesPath())
}

func loadFrom(configPath, legacyPath string) (*Config, error) {
	config, err := readConfig(configPath)
	if err != nil {
		return nil, err
	}

	// Try migrating from legacy file if no aliases exist yet
	if len(config.Aliases) == 0 && config.migrateFromLegacy(legacyPath) {
		if err := config.saveTo(configPath); err != nil {
			return nil, fmt.Errorf("failed to save migrated config: %w", err)
		}
	}

	return config, nil
}

// readConfig reads and parses the config file, returning an empty config if file doesn't exist.
func readConfig(configPath string) (*Config, error) {
	config := &Config{Aliases: make(map[string]string)}

	data, err := os.ReadFile(configPath)
	if err != nil {
		if os.IsNotExist(err) {
			return config, nil
		}
		return nil, fmt.Errorf("failed to read config file: %w", err)
	}

	if err := yaml.Unmarshal(data, config); err != nil {
		return nil, fmt.Errorf("failed to parse config file: %w", err)
	}

	if config.Aliases == nil {
		config.Aliases = make(map[string]string)
	}

	return config, nil
}

// migrateFromLegacy migrates aliases from the legacy aliases.yaml file.
// Returns true if any aliases were migrated.
// After successful migration, the legacy file is deleted.
func (c *Config) migrateFromLegacy(legacyPath string) bool {
	if legacyPath == "" {
		return false
	}

	data, err := os.ReadFile(legacyPath)
	if err != nil {
		return false
	}

	var legacy map[string]string
	if err := yaml.Unmarshal(data, &legacy); err != nil {
		slog.Warn("Failed to parse legacy aliases file", "path", legacyPath, "error", err)
		return false
	}

	if len(legacy) == 0 {
		return false
	}

	for name, path := range legacy {
		c.Aliases[name] = path
	}

	slog.Info("Migrated aliases from legacy file", "path", legacyPath, "count", len(legacy))

	if err := os.Remove(legacyPath); err != nil {
		slog.Warn("Failed to remove legacy aliases file", "path", legacyPath, "error", err)
	}

	return true
}

// Save saves the configuration to the config file
func (c *Config) Save() error {
	return c.saveTo(Path())
}

func (c *Config) saveTo(path string) error {
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		return fmt.Errorf("failed to create config directory: %w", err)
	}

	data, err := yaml.Marshal(c)
	if err != nil {
		return fmt.Errorf("failed to marshal config: %w", err)
	}

	return atomic.WriteFile(path, bytes.NewReader(data))
}

// GetAlias retrieves the agent path for an alias
func (c *Config) GetAlias(name string) (string, bool) {
	path, ok := c.Aliases[name]
	return path, ok
}

// validAliasNameRegex matches valid alias names: alphanumeric characters, hyphens, and underscores.
// Must start with an alphanumeric character.
var validAliasNameRegex = regexp.MustCompile(`^[a-zA-Z0-9][a-zA-Z0-9_-]*$`)

// ValidateAliasName checks if an alias name is valid.
// Valid names must:
// - Not be empty
// - Start with an alphanumeric character
// - Contain only alphanumeric characters, hyphens, and underscores
// - Not contain path separators or special characters
func ValidateAliasName(name string) error {
	if name == "" {
		return errors.New("alias name cannot be empty")
	}
	if !validAliasNameRegex.MatchString(name) {
		return fmt.Errorf("invalid alias name %q: must start with a letter or digit and contain only letters, digits, hyphens, and underscores", name)
	}
	return nil
}

// SetAlias creates or updates an alias.
// Returns an error if the alias name is invalid.
func (c *Config) SetAlias(name, agentPath string) error {
	if err := ValidateAliasName(name); err != nil {
		return err
	}
	if agentPath == "" {
		return errors.New("agent path cannot be empty")
	}
	c.Aliases[name] = agentPath
	return nil
}

// DeleteAlias removes an alias. Returns true if the alias existed.
func (c *Config) DeleteAlias(name string) bool {
	if _, exists := c.Aliases[name]; exists {
		delete(c.Aliases, name)
		return true
	}
	return false
}
