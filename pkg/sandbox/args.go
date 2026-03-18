package sandbox

import (
	"log/slog"
	"os"
	"path/filepath"
	"strings"

	"github.com/docker/cli/cli-plugins/plugin"

	"github.com/docker/docker-agent/pkg/userconfig"
)

// Flags stripped when forwarding args to the sandbox.
var (
	// stripFlags are standalone boolean flags that don't take a value.
	stripFlags = map[string]bool{
		"--sandbox": true,
		"--debug":   true,
		"-d":        true,
	}
	// stripFlagsWithValue are flags followed by a separate value argument.
	// They are also recognised in --flag=value form.
	stripFlagsWithValue = map[string]bool{
		"--log-file":       true,
		"--template":       true,
		"--models-gateway": true,
	}
)

// BuildCagentArgs takes os.Args and returns the arguments to pass after
// "--" to the sandbox. It strips the binary name, "--sandbox", and the first
// occurrence of the "run" subcommand. If the agent reference is a user-defined
// alias, it is resolved to its path so the sandbox (which lacks the host's
// user config) receives a concrete reference.
// It also injects --yolo since the sandbox provides isolation.
// Host-only flags --debug/-d, --log-file, --template, and --models-gateway
// are stripped because they reference host paths/logging, sandbox creation
// options, or settings forwarded via env var that don't apply inside the
// sandbox.
//
//	["cagent", "run", "./agent.yaml", "--sandbox", "--debug"] → ["./agent.yaml", "--yolo"]
//	["cagent", "--debug", "run", "--sandbox", "myalias"]      → ["/path/to/agent.yaml", "--yolo"]
func BuildCagentArgs(argv []string) []string {
	out := make([]string, 0, len(argv))
	runStripped := false
	agentStripped := false
	agentResolved := false
	hasYolo := false
	skipNext := false
	for _, a := range argv[1:] { // skip binary name
		if skipNext {
			skipNext = false
			continue
		}
		if stripFlags[a] {
			continue
		}
		if stripFlagsWithValue[a] {
			skipNext = true
			continue
		}
		if isEqualsFormOf(a, stripFlagsWithValue) {
			continue
		}
		if a == "--yolo" {
			hasYolo = true
		}
		if !runStripped && a == "run" {
			runStripped = true
			continue
		}
		if !plugin.RunningStandalone() && !agentStripped && a == "agent" {
			agentStripped = true
			continue
		}
		// The first positional arg after "run" is the agent reference.
		// Resolve it if it's a user-defined alias.
		if runStripped && !agentResolved && !strings.HasPrefix(a, "-") {
			agentResolved = true
			if resolved := ResolveAlias(a); resolved != "" {
				slog.Debug("Resolved alias for sandbox", "alias", a, "path", resolved)
				a = resolved
			}
		}
		out = append(out, a)
	}
	// The sandbox provides isolation, so auto-approve all tool calls.
	if !hasYolo {
		out = append(out, "--yolo")
	}
	return out
}

// isEqualsFormOf reports whether arg matches "--flag=..." for any flag in the set.
func isEqualsFormOf(arg string, flags map[string]bool) bool {
	for f := range flags {
		if strings.HasPrefix(arg, f+"=") {
			return true
		}
	}
	return false
}

// AgentRefFromArgs returns the first positional (non-flag) argument from the
// docker-agent arg list, which is the agent file reference. Returns "" if there are
// no positional arguments.
func AgentRefFromArgs(args []string) string {
	for _, a := range args {
		if !strings.HasPrefix(a, "-") {
			return a
		}
	}
	return ""
}

// ResolveAlias returns the alias path if name is a user-defined alias,
// or an empty string otherwise.
func ResolveAlias(name string) string {
	cfg, err := userconfig.Load()
	if err != nil {
		return ""
	}
	alias, ok := cfg.GetAlias(name)
	if !ok {
		return ""
	}
	return alias.Path
}

// ExtraWorkspace returns the directory to mount as a read-only extra workspace
// when the agent file lives outside the main workspace. Returns "" if no
// extra mount is needed (file is under wd, is not a local path, etc.).
func ExtraWorkspace(wd, agentRef string) string {
	if agentRef == "" {
		return ""
	}

	// Make the agent reference absolute so we can compare with wd.
	abs, err := filepath.Abs(agentRef)
	if err != nil {
		return ""
	}

	// Only consider paths that look like local files.
	if !looksLikeLocalFile(abs) {
		return ""
	}

	agentDir := filepath.Dir(abs)

	// No extra mount needed if the file is already under the workspace.
	if strings.HasPrefix(agentDir, wd) {
		return ""
	}

	return agentDir
}

// looksLikeLocalFile reports whether path looks like a local agent file
// (has a YAML extension or exists on disk).
func looksLikeLocalFile(path string) bool {
	ext := strings.ToLower(filepath.Ext(path))
	if ext == ".yaml" || ext == ".yml" {
		return true
	}
	info, err := os.Stat(path)
	return err == nil && !info.IsDir()
}

// AppendFlagIfMissing appends "--flag value" to args unless args already
// contain the flag (in either "--flag value" or "--flag=value" form).
func AppendFlagIfMissing(args []string, flag, value string) []string {
	for _, a := range args {
		if a == flag || strings.HasPrefix(a, flag+"=") {
			return args
		}
	}
	return append(args, flag, value)
}
