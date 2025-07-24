package loader

import (
	"fmt"
	"os"

	"github.com/docker/cagent/pkg/environment"
)

func toolsetEnv(env map[string]string, envFiles []string, parentDir string) ([]string, error) {
	var envSlice []string

	for k, v := range env {
		v = environment.Expand(v, os.Environ())
		envSlice = append(envSlice, fmt.Sprintf("%s=%s", k, v))
	}

	absoluteEnvFiles, err := environment.AbsolutePaths(parentDir, envFiles)
	if err != nil {
		return nil, err
	}

	keyValues, err := environment.ReadEnvFiles(absoluteEnvFiles)
	if err != nil {
		return nil, err
	}

	for _, kv := range keyValues {
		v := environment.Expand(kv.Value, os.Environ())
		envSlice = append(envSlice, fmt.Sprintf("%s=%s", kv.Key, v))
	}

	return envSlice, nil
}
