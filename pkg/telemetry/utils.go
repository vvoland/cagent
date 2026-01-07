package telemetry

import (
	"cmp"
	"encoding/json"
	"flag"
	"fmt"
	"os"
	"path/filepath"
	"runtime"
	"strings"

	"github.com/google/uuid"

	"github.com/docker/cagent/pkg/paths"
)

// getSystemInfo collects system information for events
func getSystemInfo() (osName, osVersion, osLanguage string) {
	osInfo := runtime.GOOS
	osLang := cmp.Or(os.Getenv("LANG"), "en-US")
	return osInfo, "", osLang
}

func GetTelemetryEnabled() bool {
	// Disable telemetry when running in tests to prevent HTTP calls
	if flag.Lookup("test.v") != nil {
		return false
	}
	return getTelemetryEnabledFromEnv()
}

// getTelemetryEnabledFromEnv checks only the environment variable,
// without the test detection bypass. This allows testing the env var logic.
func getTelemetryEnabledFromEnv() bool {
	if env := os.Getenv("TELEMETRY_ENABLED"); env != "" {
		// Only disable if explicitly set to "false"
		return env != "false"
	}
	// Default to true (telemetry enabled)
	return true
}

// getUserUUIDFilePath returns the path to the user UUID file
func getUserUUIDFilePath() string {
	configDir := paths.GetConfigDir()
	return filepath.Join(configDir, "user-uuid")
}

// getUserUUID gets or creates a persistent user UUID
func getUserUUID() string {
	uuidFile := getUserUUIDFilePath()

	// Try to read existing UUID
	if data, err := os.ReadFile(uuidFile); err == nil {
		existingUUID := strings.TrimSpace(string(data))
		if existingUUID != "" {
			return existingUUID
		}
		// UUID file exists but is empty/invalid - will generate new one
	}

	// Generate new UUID and save it
	newUUID := uuid.New().String()
	if err := saveUserUUID(newUUID); err != nil {
		// If we can't save, still return a UUID for this session
		// but it won't persist across runs
		return newUUID
	}

	return newUUID
}

// saveUserUUID saves the UUID to disk
func saveUserUUID(newUUID string) error {
	uuidFile := getUserUUIDFilePath()

	// Ensure directory exists
	dir := filepath.Dir(uuidFile)
	if err := os.MkdirAll(dir, 0o755); err != nil {
		return err
	}

	// Write UUID to file (readable only by user)
	return os.WriteFile(uuidFile, []byte(newUUID), 0o600)
}

// structToMap converts a struct to map[string]any using JSON marshaling
// This automatically handles all fields and respects JSON tags (including omitempty)
func structToMap(v any) (map[string]any, error) {
	data, err := json.Marshal(v)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal struct: %w", err)
	}

	var result map[string]any
	err = json.Unmarshal(data, &result)
	if err != nil {
		return nil, fmt.Errorf("failed to unmarshal to map: %w", err)
	}

	return result, nil
}

// CommandInfo represents the parsed command information
type CommandInfo struct {
	Action string
	Args   []string
	Flags  []string
}
