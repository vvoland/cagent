package environment

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

func AbsolutePaths(parentDir string, relOrAbsPaths []string) ([]string, error) {
	var absPaths []string

	for _, relOrAbsPath := range relOrAbsPaths {
		absPath, err := AbsolutePath(parentDir, relOrAbsPath)
		if err != nil {
			return nil, fmt.Errorf("failed to resolve path %s: %w", relOrAbsPath, err)
		}
		absPaths = append(absPaths, absPath)
	}

	return absPaths, nil
}

func AbsolutePath(parentDir, relOrAbsPath string) (string, error) {
	path, err := expandTildePath(relOrAbsPath)
	if err != nil {
		return "", err
	}

	if filepath.IsAbs(path) {
		return path, nil
	}

	return filepath.Join(parentDir, path), nil
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

func ReadEnvFiles(absolutePaths []string) ([]KeyValuePair, error) {
	if len(absolutePaths) == 0 {
		return nil, nil
	}

	var allLines []KeyValuePair

	for _, absolutePath := range absolutePaths {
		lines, err := ReadEnvFile(absolutePath)
		if err != nil {
			return nil, err
		}
		allLines = append(allLines, lines...)
	}

	return allLines, nil
}

func ReadEnvFile(absolutePaths string) ([]KeyValuePair, error) {
	buf, err := os.ReadFile(absolutePaths)
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
