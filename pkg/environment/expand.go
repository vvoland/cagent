package environment

import (
	"context"
	"fmt"
	"os"
	"slices"
)

func ExpandAll(ctx context.Context, values []string, env Provider) ([]string, error) {
	var expandedEnv []string

	for _, value := range values {
		expanded, err := Expand(ctx, value, env)
		if err != nil {
			return nil, err
		}

		expandedEnv = append(expandedEnv, expanded)
	}

	return expandedEnv, nil
}

func Expand(ctx context.Context, value string, env Provider) (string, error) {
	var err error

	expanded := os.Expand(value, func(name string) string {
		v, found := env.Get(ctx, name)
		if !found {
			err = fmt.Errorf("environment variable %q not set", name)
		}
		return v
	})
	if err != nil {
		return "", err
	}

	return expanded, nil
}

func ToValues(envMap map[string]string) []string {
	var values []string
	for k, v := range envMap {
		values = append(values, fmt.Sprintf("%s=%s", k, v))
	}
	slices.Sort(values)
	return values
}
