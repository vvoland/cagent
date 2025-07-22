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
