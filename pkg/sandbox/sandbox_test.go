package sandbox_test

import (
	"fmt"
	"os"
	"path/filepath"
	"testing"

	"github.com/docker/cli/cli-plugins/metadata"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/docker/docker-agent/pkg/sandbox"
	"github.com/docker/docker-agent/pkg/userconfig"
)

func TestCheckAvailable(t *testing.T) {
	tests := []struct {
		name      string
		script    string // empty means no fake binary (docker not found)
		wantErr   string
		wantNoErr bool
	}{
		{
			name:    "no docker installed",
			wantErr: "--sandbox requires Docker Desktop",
		},
		{
			name:    "docker without sandbox support",
			script:  "#!/bin/sh\nexit 1\n",
			wantErr: "--sandbox requires Docker Desktop with sandbox support",
		},
		{
			name:      "docker with sandbox support",
			script:    "#!/bin/sh\nexit 0\n",
			wantNoErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			fakeDir := t.TempDir()
			if tt.script != "" {
				require.NoError(t, os.WriteFile(filepath.Join(fakeDir, "docker"), []byte(tt.script), 0o755))
			}
			t.Setenv("PATH", fakeDir)

			err := sandbox.CheckAvailable(t.Context())
			if tt.wantNoErr {
				require.NoError(t, err)
			} else {
				require.Error(t, err)
				assert.Contains(t, err.Error(), tt.wantErr)
			}
		})
	}
}

func TestForWorkspace(t *testing.T) {
	tests := []struct {
		name     string
		json     string
		wd       string
		wantName string
	}{
		{
			name:     "matching workspace",
			json:     `{"vms":[{"name":"my-sandbox","workspaces":["/my/project"]}]}`,
			wd:       "/my/project",
			wantName: "my-sandbox",
		},
		{
			name: "no match",
			json: `{"vms":[{"name":"other","workspaces":["/other/project"]}]}`,
			wd:   "/my/project",
		},
		{
			name: "empty list",
			json: `{"vms":[]}`,
			wd:   "/my/project",
		},
		{
			name:     "multiple sandboxes",
			json:     `{"vms":[{"name":"a","workspaces":["/a"]},{"name":"b","workspaces":["/b"]}]}`,
			wd:       "/b",
			wantName: "b",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			fakeDir := t.TempDir()
			script := fmt.Sprintf("#!/bin/sh\necho '%s'\n", tt.json)
			require.NoError(t, os.WriteFile(filepath.Join(fakeDir, "docker"), []byte(script), 0o755))
			t.Setenv("PATH", fakeDir)

			got := sandbox.ForWorkspace(t.Context(), tt.wd)
			if tt.wantName == "" {
				assert.Nil(t, got)
			} else {
				require.NotNil(t, got)
				assert.Equal(t, tt.wantName, got.Name)
			}
		})
	}
}

func TestExisting_HasWorkspace(t *testing.T) {
	t.Parallel()

	s := &sandbox.Existing{
		Name:       "test",
		Workspaces: []string{"/workspace", "/extra:ro"},
	}

	assert.True(t, s.HasWorkspace("/workspace"))
	assert.True(t, s.HasWorkspace("/extra"), "should match ignoring :ro suffix")
	assert.False(t, s.HasWorkspace("/other"))
}

func TestBuildCagentArgs(t *testing.T) {
	tests := []struct {
		name string
		env  map[string]string
		argv []string
		want []string
	}{
		{
			name: "using cli plugin, strips run and --sandbox and --debug, adds --yolo",
			env:  map[string]string{metadata.ReexecEnvvar: "docker"}, // env var set by cli plugin execution
			argv: []string{"docker", "agent", "run", "./agent.yaml", "--sandbox", "--debug"},
			want: []string{"./agent.yaml", "--yolo"},
		},
		{
			name: "using cli plugin, strips -d shorthand",
			env:  map[string]string{metadata.ReexecEnvvar: "docker"}, // env var set by cli plugin execution
			argv: []string{"docker", "agent", "-d", "run", "./agent.yaml"},
			want: []string{"./agent.yaml", "--yolo"},
		},
		{
			name: "strips run and --sandbox and --debug, adds --yolo",
			argv: []string{"cagent", "run", "./agent.yaml", "--sandbox", "--debug"},
			want: []string{"./agent.yaml", "--yolo"},
		},
		{
			name: "strips --sandbox at end",
			argv: []string{"cagent", "run", "./agent.yaml", "--sandbox"},
			want: []string{"./agent.yaml", "--yolo"},
		},
		{
			name: "strips --debug before run",
			argv: []string{"cagent", "--debug", "run", "--sandbox", "default"},
			want: []string{"default", "--yolo"},
		},
		{
			name: "strips -d shorthand",
			argv: []string{"cagent", "-d", "run", "./agent.yaml"},
			want: []string{"./agent.yaml", "--yolo"},
		},
		{
			name: "strips --log-file with value",
			argv: []string{"cagent", "run", "./agent.yaml", "--debug", "--log-file", "/tmp/debug.log"},
			want: []string{"./agent.yaml", "--yolo"},
		},
		{
			name: "strips --log-file=value",
			argv: []string{"cagent", "run", "./agent.yaml", "--log-file=/tmp/debug.log"},
			want: []string{"./agent.yaml", "--yolo"},
		},
		{
			name: "no --sandbox present, still strips debug flags",
			argv: []string{"cagent", "run", "./agent.yaml", "--debug"},
			want: []string{"./agent.yaml", "--yolo"},
		},
		{
			name: "only binary and run",
			argv: []string{"cagent", "run"},
			want: []string{"--yolo"},
		},
		{
			name: "does not duplicate --yolo",
			argv: []string{"cagent", "run", "./agent.yaml", "--sandbox", "--yolo"},
			want: []string{"./agent.yaml", "--yolo"},
		},
		{
			name: "only strips first run occurrence",
			argv: []string{"cagent", "run", "--sandbox", "run.yaml"},
			want: []string{"run.yaml", "--yolo"},
		},
		{
			name: "sandbox and run only",
			argv: []string{"cagent", "run", "--sandbox"},
			want: []string{"--yolo"},
		},
		{
			name: "strips --template with value",
			argv: []string{"cagent", "run", "./agent.yaml", "--sandbox", "--template", "my-image:latest"},
			want: []string{"./agent.yaml", "--yolo"},
		},
		{
			name: "strips --template=value",
			argv: []string{"cagent", "run", "./agent.yaml", "--sandbox", "--template=my-image:latest"},
			want: []string{"./agent.yaml", "--yolo"},
		},
		{
			name: "strips --models-gateway with value",
			argv: []string{"cagent", "run", "./agent.yaml", "--sandbox", "--models-gateway", "http://gw:8080"},
			want: []string{"./agent.yaml", "--yolo"},
		},
		{
			name: "strips --models-gateway=value",
			argv: []string{"cagent", "run", "./agent.yaml", "--sandbox", "--models-gateway=http://gw:8080"},
			want: []string{"./agent.yaml", "--yolo"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Use an empty HOME so no real user aliases interfere.
			t.Setenv("HOME", t.TempDir())
			if tt.env != nil {
				for k, v := range tt.env {
					t.Setenv(k, v)
				}
			}

			got := sandbox.BuildCagentArgs(tt.argv)
			assert.Equal(t, tt.want, got)
		})
	}
}

func TestBuildCagentArgs_ResolvesAlias(t *testing.T) {
	// Set up a temporary HOME with an alias "mydev" -> "/path/to/dev.yaml".
	home := t.TempDir()
	t.Setenv("HOME", home)

	cfg, err := userconfig.Load()
	require.NoError(t, err)
	require.NoError(t, cfg.SetAlias("mydev", &userconfig.Alias{Path: "/path/to/dev.yaml"}))
	require.NoError(t, cfg.Save())

	got := sandbox.BuildCagentArgs([]string{"cagent", "run", "--sandbox", "mydev", "--debug"})
	assert.Equal(t, []string{"/path/to/dev.yaml", "--yolo"}, got)
}

func TestAgentRefFromArgs(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name string
		args []string
		want string
	}{
		{"file first", []string{"./agent.yaml", "--debug"}, "./agent.yaml"},
		{"flags before file", []string{"--debug", "./agent.yaml"}, "./agent.yaml"},
		{"no positional", []string{"--debug", "--yolo"}, ""},
		{"empty", nil, ""},
		{"built-in name", []string{"default"}, "default"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			assert.Equal(t, tt.want, sandbox.AgentRefFromArgs(tt.args))
		})
	}
}

func TestExtraWorkspace(t *testing.T) {
	tests := []struct {
		name     string
		wd       string
		agentRef string
		setup    func(t *testing.T) // create files if needed
		want     string
	}{
		{
			name: "empty ref",
			wd:   "/workspace",
			want: "",
		},
		{
			name:     "yaml outside workspace",
			wd:       "/workspace",
			agentRef: "/other/dir/agent.yaml",
			want:     "/other/dir",
		},
		{
			name:     "yaml inside workspace",
			wd:       "/workspace",
			agentRef: "/workspace/sub/agent.yaml",
			want:     "",
		},
		{
			name:     "yaml in workspace root",
			wd:       "/workspace",
			agentRef: "/workspace/agent.yaml",
			want:     "",
		},
		{
			name:     "built-in name",
			wd:       "/workspace",
			agentRef: "default",
			want:     "",
		},
		{
			name:     "OCI reference",
			wd:       "/workspace",
			agentRef: "docker.io/myorg/agent:latest",
			want:     "",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if tt.setup != nil {
				tt.setup(t)
			}
			got := sandbox.ExtraWorkspace(tt.wd, tt.agentRef)
			assert.Equal(t, tt.want, got)
		})
	}
}

func TestAppendFlagIfMissing(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name  string
		args  []string
		flag  string
		value string
		want  []string
	}{
		{
			name:  "appends when missing",
			args:  []string{"./agent.yaml", "--yolo"},
			flag:  "--models-gateway",
			value: "http://gw:8080",
			want:  []string{"./agent.yaml", "--yolo", "--models-gateway", "http://gw:8080"},
		},
		{
			name:  "skips when present as separate arg",
			args:  []string{"./agent.yaml", "--models-gateway", "http://gw:8080"},
			flag:  "--models-gateway",
			value: "http://other:9090",
			want:  []string{"./agent.yaml", "--models-gateway", "http://gw:8080"},
		},
		{
			name:  "skips when present as equals form",
			args:  []string{"./agent.yaml", "--data-dir=/custom/path"},
			flag:  "--data-dir",
			value: "/default/path",
			want:  []string{"./agent.yaml", "--data-dir=/custom/path"},
		},
		{
			name:  "empty args",
			args:  nil,
			flag:  "--data-dir",
			value: "/some/path",
			want:  []string{"--data-dir", "/some/path"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			got := sandbox.AppendFlagIfMissing(tt.args, tt.flag, tt.value)
			assert.Equal(t, tt.want, got)
		})
	}
}
