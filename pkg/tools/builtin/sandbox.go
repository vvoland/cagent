package builtin

import (
	"bytes"
	"cmp"
	"context"
	"crypto/rand"
	"encoding/hex"
	"fmt"
	"log/slog"
	"os"
	"os/exec"
	"path/filepath"
	"strconv"
	"strings"
	"sync"
	"syscall"
	"time"

	"github.com/docker/cagent/pkg/config/latest"
	"github.com/docker/cagent/pkg/tools"
)

const (
	// sandboxLabelKey is the label used to identify cagent sandbox containers.
	sandboxLabelKey = "com.docker.cagent.sandbox"
	// sandboxLabelPID stores the PID of the cagent process that created the container.
	sandboxLabelPID = "com.docker.cagent.sandbox.pid"
)

// sandboxRunner handles command execution in a Docker container sandbox.
type sandboxRunner struct {
	config      *latest.SandboxConfig
	workingDir  string
	env         []string
	containerID string
	mu          sync.Mutex
}

func newSandboxRunner(config *latest.SandboxConfig, workingDir string, env []string) *sandboxRunner {
	// Clean up any orphaned containers from previous cagent runs
	cleanupOrphanedSandboxContainers()

	return &sandboxRunner{
		config:     config,
		workingDir: workingDir,
		env:        env,
	}
}

// cleanupOrphanedSandboxContainers removes sandbox containers from previous cagent processes
// that are no longer running. This handles cases where cagent was killed or crashed.
func cleanupOrphanedSandboxContainers() {
	cmd := exec.Command("docker", "ps", "-q", "--filter", "label="+sandboxLabelKey)
	output, err := cmd.Output()
	if err != nil {
		return // Docker not available or no containers
	}

	containerIDs := strings.Fields(string(output))
	currentPID := os.Getpid()

	for _, containerID := range containerIDs {
		pid := getContainerOwnerPID(containerID)
		if pid == 0 || pid == currentPID || isProcessRunning(pid) {
			continue
		}

		slog.Debug("Cleaning up orphaned sandbox container", "container", containerID, "pid", pid)
		stopCmd := exec.Command("docker", "stop", "-t", "1", containerID)
		_ = stopCmd.Run()
	}
}

// getContainerOwnerPID returns the PID that created the container, or 0 if unknown.
func getContainerOwnerPID(containerID string) int {
	cmd := exec.Command("docker", "inspect", "-f",
		"{{index .Config.Labels \""+sandboxLabelPID+"\"}}", containerID)
	output, err := cmd.Output()
	if err != nil {
		return 0
	}
	pid, _ := strconv.Atoi(strings.TrimSpace(string(output)))
	return pid
}

// isProcessRunning checks if a process with the given PID is still running.
func isProcessRunning(pid int) bool {
	process, err := os.FindProcess(pid)
	if err != nil {
		return false
	}
	// On Unix, FindProcess always succeeds, so we need to send signal 0
	// to check if the process actually exists
	err = process.Signal(syscall.Signal(0))
	return err == nil
}

// runCommand executes a command inside the sandbox container.
func (s *sandboxRunner) runCommand(timeoutCtx, ctx context.Context, command, cwd string, timeout time.Duration) *tools.ToolCallResult {
	containerID, err := s.ensureContainer(ctx)
	if err != nil {
		return tools.ResultError(fmt.Sprintf("Failed to start sandbox container: %s", err))
	}

	args := []string{"exec", "-w", cwd}
	args = append(args, s.buildEnvVars()...)
	args = append(args, containerID, "/bin/sh", "-c", command)

	cmd := exec.CommandContext(timeoutCtx, "docker", args...)
	var outBuf bytes.Buffer
	cmd.Stdout = &outBuf
	cmd.Stderr = &outBuf

	err = cmd.Run()

	output := formatCommandOutput(timeoutCtx, ctx, err, outBuf.String(), timeout)
	return tools.ResultSuccess(limitOutput(output))
}

// stop stops and removes the sandbox container.
func (s *sandboxRunner) stop() {
	s.mu.Lock()
	defer s.mu.Unlock()

	if s.containerID == "" {
		return
	}

	stopCmd := exec.Command("docker", "stop", "-t", "1", s.containerID)
	_ = stopCmd.Run()

	s.containerID = ""
}

// ensureContainer ensures the sandbox container is running, starting it if necessary.
func (s *sandboxRunner) ensureContainer(ctx context.Context) (string, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	if s.containerID != "" && s.isContainerRunning(ctx) {
		return s.containerID, nil
	}
	s.containerID = ""

	return s.startContainer(ctx)
}

func (s *sandboxRunner) isContainerRunning(ctx context.Context) bool {
	cmd := exec.CommandContext(ctx, "docker", "container", "inspect", "-f", "{{.State.Running}}", s.containerID)
	output, err := cmd.Output()
	return err == nil && strings.TrimSpace(string(output)) == "true"
}

func (s *sandboxRunner) startContainer(ctx context.Context) (string, error) {
	containerName := s.generateContainerName()
	image := cmp.Or(s.config.Image, "alpine:latest")

	args := []string{
		"run", "-d",
		"--name", containerName,
		"--rm", "--init", "--network", "host",
		"--label", sandboxLabelKey + "=true",
		"--label", fmt.Sprintf("%s=%d", sandboxLabelPID, os.Getpid()),
	}
	args = append(args, s.buildVolumeMounts()...)
	args = append(args, s.buildEnvVars()...)
	args = append(args, "-w", s.workingDir, image, "tail", "-f", "/dev/null")

	cmd := exec.CommandContext(ctx, "docker", args...)
	var stderr bytes.Buffer
	cmd.Stderr = &stderr

	output, err := cmd.Output()
	if err != nil {
		return "", fmt.Errorf("failed to start sandbox container: %w\nstderr: %s", err, stderr.String())
	}

	s.containerID = strings.TrimSpace(string(output))
	return s.containerID, nil
}

func (s *sandboxRunner) generateContainerName() string {
	randomBytes := make([]byte, 4)
	_, _ = rand.Read(randomBytes)
	return fmt.Sprintf("cagent-sandbox-%s", hex.EncodeToString(randomBytes))
}

func (s *sandboxRunner) buildVolumeMounts() []string {
	var args []string
	for _, pathSpec := range s.config.Paths {
		hostPath, mode := parseSandboxPath(pathSpec)

		// Resolve to absolute path
		if !filepath.IsAbs(hostPath) {
			if s.workingDir != "" {
				hostPath = filepath.Join(s.workingDir, hostPath)
			} else {
				// If workingDir is empty, resolve relative to current directory
				var err error
				hostPath, err = filepath.Abs(hostPath)
				if err != nil {
					// Skip invalid paths
					continue
				}
			}
		}
		hostPath = filepath.Clean(hostPath)

		// Container path mirrors host path for simplicity
		mountSpec := fmt.Sprintf("%s:%s:%s", hostPath, hostPath, mode)
		args = append(args, "-v", mountSpec)
	}
	return args
}

// buildEnvVars creates Docker -e flags for environment variables.
// This forwards the host environment to the sandbox container.
// Only variables with valid POSIX names are forwarded.
func (s *sandboxRunner) buildEnvVars() []string {
	var args []string
	for _, envVar := range s.env {
		// Each env var is in KEY=VALUE format
		// Only forward variables with valid names to avoid Docker issues
		if idx := strings.Index(envVar, "="); idx > 0 {
			key := envVar[:idx]
			if isValidEnvVarName(key) {
				args = append(args, "-e", envVar)
			}
		}
	}
	return args
}

// isValidEnvVarName checks if an environment variable name is valid for POSIX.
// Valid names start with a letter or underscore and contain only alphanumerics and underscores.
func isValidEnvVarName(name string) bool {
	if name == "" {
		return false
	}
	for i, c := range name {
		isValid := (c >= 'a' && c <= 'z') || (c >= 'A' && c <= 'Z') || c == '_' || (i > 0 && c >= '0' && c <= '9')
		if !isValid {
			return false
		}
	}
	return true
}

// parseSandboxPath parses a path specification like "./path" or "/path:ro" into path and mode.
func parseSandboxPath(pathSpec string) (path, mode string) {
	mode = "rw" // Default to read-write

	switch {
	case strings.HasSuffix(pathSpec, ":ro"):
		path = strings.TrimSuffix(pathSpec, ":ro")
		mode = "ro"
	case strings.HasSuffix(pathSpec, ":rw"):
		path = strings.TrimSuffix(pathSpec, ":rw")
		mode = "rw"
	default:
		path = pathSpec
	}

	return path, mode
}
