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

	//go:embed Dockerfile.custom.template
	dockerfileCustomTmpl string

	dockerfileTemplate       = template.Must(template.New("Dockerfile").Parse(dockerfileTmpl))
	dockerfileCustomTemplate = template.Must(template.New("DockerfileCustom").Parse(dockerfileCustomTmpl))
)

// getOrBuildImage returns a cached image ID or builds a new one.
// Images are cached by working directory to avoid redundant builds.
// Concurrent calls for the same working directory are deduplicated
// using singleflight so that only one build runs at a time per key.
func (r *Runner) getOrBuildImage(ctx context.Context, workingDir string) (string, error) {
	r.imageCacheMu.Lock()
	if imageID, ok := r.imageCache[workingDir]; ok {
		r.imageCacheMu.Unlock()
		return imageID, nil
	}
	r.imageCacheMu.Unlock()

	// singleflight ensures only one build per working directory runs at a time.
	// The cache write inside the callback guarantees the result is available
	// before singleflight releases the key, so subsequent callers always
	// hit the cache above.
	v, err, _ := r.imageBuildGroup.Do(workingDir, func() (any, error) {
		imageID, err := r.buildEvalImage(ctx, workingDir)
		if err != nil {
			return "", err
		}

		r.imageCacheMu.Lock()
		r.imageCache[workingDir] = imageID
		r.imageCacheMu.Unlock()

		return imageID, nil
	})
	if err != nil {
		return "", err
	}

	return v.(string), nil
}

func (r *Runner) buildEvalImage(ctx context.Context, workingDir string) (string, error) {
	var buildContext string
	var data struct {
		CopyWorkingDir bool
		BaseImage      string
	}

	if workingDir == "" {
		buildContext = r.EvalsDir
		data.CopyWorkingDir = false
	} else {
		buildContext = filepath.Join(r.EvalsDir, "working_dirs", workingDir)
		if _, err := os.Stat(buildContext); os.IsNotExist(err) {
			return "", fmt.Errorf("working directory not found: %s", buildContext)
		}
		data.CopyWorkingDir = true
	}

	// Choose template based on whether a custom base image is provided
	tmpl := dockerfileTemplate
	if r.BaseImage != "" {
		tmpl = dockerfileCustomTemplate
		data.BaseImage = r.BaseImage
	}

	var dockerfile bytes.Buffer
	if err := tmpl.Execute(&dockerfile, data); err != nil {
		return "", fmt.Errorf("executing dockerfile template: %w", err)
	}

	cmd := exec.CommandContext(ctx, "docker", "build", "-q", "-f-", ".")
	cmd.Dir = buildContext
	cmd.Stdin = &dockerfile

	output, err := cmd.Output()
	if err != nil {
		if exitErr, ok := errors.AsType[*exec.ExitError](err); ok {
			return "", fmt.Errorf("docker build failed: %s", string(exitErr.Stderr))
		}
		return "", fmt.Errorf("docker build failed: %w", err)
	}

	return strings.TrimSpace(string(output)), nil
}
