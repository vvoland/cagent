package loader

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
)

type KeyValuePair struct {
	Key   string
	Value string
}

func toolsetEnv(env map[string]string, envFiles []string, parentDir string) ([]string, error) {
	var envSlice []string

	for k, v := range env {
		v = expandEnv(v, append(os.Environ(), envSlice...))
		envSlice = append(envSlice, fmt.Sprintf("%s=%s", k, v))
	}

	keyValues, err := readEnvFiles(parentDir, envFiles)
	if err != nil {
		return nil, err
	}

	for _, kv := range keyValues {
		v := expandEnv(kv.Value, append(os.Environ(), envSlice...))
		envSlice = append(envSlice, fmt.Sprintf("%s=%s", kv.Key, v))
	}

	return envSlice, nil
}

func expandEnv(value string, env []string) string {
	return os.Expand(value, func(name string) string {
		for _, e := range env {
			if after, ok := strings.CutPrefix(e, name+"="); ok {
				return after
			}
		}
		return ""
	})
}

// expandTildePath expands ~ in file paths to the user's home directory
func expandTildePath(path string) (string, error) {
	if !strings.HasPrefix(path, "~") {
		return path, nil
	}

	homeDir, err := os.UserHomeDir()
	if err != nil {
		return "", fmt.Errorf("failed to get user home directory: %w", err)
	}

	if path == "~" {
		return homeDir, nil
	}

	if strings.HasPrefix(path, "~/") {
		return filepath.Join(homeDir, path[2:]), nil
	}

	// Handle ~username/ format - not commonly supported in this context
	return "", fmt.Errorf("unsupported tilde expansion format: %s", path)
}

func readEnvFiles(parentDir string, paths []string) ([]KeyValuePair, error) {
	if len(paths) == 0 {
		return nil, nil
	}

	var allLines []KeyValuePair

	for _, path := range paths {
		lines, err := readEnvFile(parentDir, path)
		if err != nil {
			return nil, err
		}
		allLines = append(allLines, lines...)
	}

	return allLines, nil
}

func readEnvFile(parentDir, path string) ([]KeyValuePair, error) {
	// First expand tilde paths
	expandedPath, err := expandTildePath(path)
	if err != nil {
		return nil, err
	}
	path = expandedPath

	if !filepath.IsAbs(path) {
		path = filepath.Join(parentDir, path)
	}

	buf, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}

	var lines []KeyValuePair

	for line := range strings.SplitSeq(string(buf), "\n") {
		line = strings.TrimSpace(line)
		if line == "" || strings.HasPrefix(line, "#") {
			continue
		}

		k, v, ok := strings.Cut(line, "=")
		if !ok {
			return nil, fmt.Errorf("invalid env file line: %s", line)
		}

		if strings.HasPrefix(v, `"`) && strings.HasSuffix(v, `"`) {
			v = strings.TrimSuffix(strings.TrimPrefix(v, `"`), `"`)
		}

		lines = append(lines, KeyValuePair{
			Key:   k,
			Value: v,
		})
	}

	return lines, nil
}
