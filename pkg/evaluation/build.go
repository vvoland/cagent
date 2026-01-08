package evaluation

import (
	"bytes"
	"context"
	_ "embed"
	"errors"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"text/template"
)

var (
	//go:embed Dockerfile.template
	dockerfileTmpl string

	dockerfileTemplate = template.Must(template.New("Dockerfile").Parse(dockerfileTmpl))
)

func (r *Runner) buildEvalImage(ctx context.Context, workingDir string) (string, error) {
	var buildContext string
	var data struct {
		CopyWorkingDir bool
	}

	if workingDir == "" {
		buildContext = r.evalsDir
		data.CopyWorkingDir = false
	} else {
		buildContext = filepath.Join(r.evalsDir, "working_dirs", workingDir)
		if _, err := os.Stat(buildContext); os.IsNotExist(err) {
			return "", fmt.Errorf("working directory not found: %s", buildContext)
		}
		data.CopyWorkingDir = true
	}

	var dockerfile bytes.Buffer
	if err := dockerfileTemplate.Execute(&dockerfile, data); err != nil {
		return "", fmt.Errorf("executing dockerfile template: %w", err)
	}

	cmd := exec.CommandContext(ctx, "docker", "build", "-q", "-f-", ".")
	cmd.Dir = buildContext
	cmd.Stdin = &dockerfile

	output, err := cmd.Output()
	if err != nil {
		var exitErr *exec.ExitError
		if errors.As(err, &exitErr) {
			return "", fmt.Errorf("docker build failed: %s", string(exitErr.Stderr))
		}
		return "", fmt.Errorf("docker build failed: %w", err)
	}

	return strings.TrimSpace(string(output)), nil
}
