package environment

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/docker/cagent/pkg/path"
	"github.com/docker/cagent/pkg/paths"
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
	p, err := expandTildePath(relOrAbsPath)
	if err != nil {
		return "", err
	}

	// For absolute paths (including tilde-expanded ones), validate against directory traversal
	if filepath.IsAbs(p) {
		if strings.Contains(relOrAbsPath, "..") {
			return "", fmt.Errorf("invalid environment file path: path contains directory traversal sequences")
		}
		return p, nil
	}

	validatedPath, err := path.ValidatePathInDirectory(p, parentDir)
	if err != nil {
		return "", fmt.Errorf("invalid environment file path: %w", err)
	}

	return validatedPath, nil
}

// expandTildePath expands ~ in file paths to the user's home directory
func expandTildePath(p string) (string, error) {
	if !strings.HasPrefix(p, "~") {
		return p, nil
	}

	homeDir := paths.GetHomeDir()
	if homeDir == "" {
		return "", fmt.Errorf("failed to get user home directory")
	}

	if p == "~" {
		return homeDir, nil
	}

	if strings.HasPrefix(p, "~/") {
		return filepath.Join(homeDir, p[2:]), nil
	}

	// Handle ~username/ format - not commonly supported in this context
	return "", fmt.Errorf("unsupported tilde expansion format: %s", p)
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

func ReadEnvFile(absolutePath string) ([]KeyValuePair, error) {
	buf, err := os.ReadFile(absolutePath)
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

		k = strings.TrimSpace(k)
		v = strings.TrimSpace(v)

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
