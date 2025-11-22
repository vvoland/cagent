package server

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/docker/cagent/pkg/api"
	"github.com/docker/cagent/pkg/config"
	"github.com/docker/cagent/pkg/config/latest"
	"github.com/docker/cagent/pkg/session"
	"github.com/docker/cagent/pkg/teamloader"
)

func TestServer_ListAgents(t *testing.T) {
	// t.Parallel()
	t.Setenv("OPENAI_API_KEY", "dummy")
	t.Setenv("ANTHROPIC_API_KEY", "dummy")

	ctx := t.Context()
	lnPath := startServer(t, ctx, prepareAgentsDir(t, "pirate.yaml", "contradict.yaml", "multi_agents.yaml"))

	buf := httpGET(t, ctx, lnPath, "/api/agents")

	var agents []api.Agent
	unmarshal(t, buf, &agents)

	assert.Len(t, agents, 3)

	assert.Equal(t, "contradict.yaml", agents[0].Name)
	assert.Equal(t, "Contrarian viewpoint provider", agents[0].Description)
	assert.False(t, agents[0].Multi)

	assert.Equal(t, "multi_agents.yaml", agents[1].Name)
	assert.Equal(t, "Multi Agent", agents[1].Description)
	assert.True(t, agents[1].Multi)

	assert.Equal(t, "pirate.yaml", agents[2].Name)
	assert.Equal(t, "Talk like a pirate", agents[2].Description)
	assert.False(t, agents[2].Multi)
}

func TestServer_GetAgent_NoExtension(t *testing.T) {
	t.Parallel()

	ctx := t.Context()
	lnPath := startServer(t, ctx, prepareAgentsDir(t, "pirate.yaml"))

	buf := httpGET(t, ctx, lnPath, "/api/agents/pirate")

	var cfg latest.Config
	unmarshal(t, buf, &cfg)

	assert.NotEmpty(t, cfg.Version)
	require.Len(t, cfg.Agents, 1)
	assert.Contains(t, cfg.Agents["root"].Instruction, "pirate")
}

func TestServer_GetAgent(t *testing.T) {
	t.Parallel()

	ctx := t.Context()
	lnPath := startServer(t, ctx, prepareAgentsDir(t, "pirate.yaml"))

	buf := httpGET(t, ctx, lnPath, "/api/agents/pirate.yaml")

	var cfg latest.Config
	unmarshal(t, buf, &cfg)

	assert.NotEmpty(t, cfg.Version)
	require.Len(t, cfg.Agents, 1)
	assert.Contains(t, cfg.Agents["root"].Instruction, "pirate")
}

func TestServer_GetSetYaml_NoExtension(t *testing.T) {
	// t.Parallel()
	t.Setenv("OPENAI_API_KEY", "dummy")

	ctx := t.Context()
	lnPath := startServer(t, ctx, prepareAgentsDir(t, "pirate.yaml"))

	url := "/api/agents/pirate/yaml"
	origContent := httpGET(t, ctx, lnPath, url)
	assert.Contains(t, string(origContent), "pirate")

	httpPUT(t, ctx, lnPath, url, origContent)
	assert.Equal(t, origContent, httpGET(t, ctx, lnPath, url))

	httpPUT(t, ctx, lnPath, url, []byte(`version: "2"`))
	assert.Equal(t, []byte(`version: "2"`), httpGET(t, ctx, lnPath, url))
}

func TestServer_GetSetYaml(t *testing.T) {
	// t.Parallel()
	t.Setenv("OPENAI_API_KEY", "dummy")

	ctx := t.Context()
	lnPath := startServer(t, ctx, prepareAgentsDir(t, "pirate.yaml"))

	url := "/api/agents/pirate.yaml/yaml"
	origContent := httpGET(t, ctx, lnPath, url)
	assert.Contains(t, string(origContent), "pirate")

	httpPUT(t, ctx, lnPath, url, origContent)
	assert.Equal(t, origContent, httpGET(t, ctx, lnPath, url))

	httpPUT(t, ctx, lnPath, url, []byte(`version: "2"`))
	assert.Equal(t, []byte(`version: "2"`), httpGET(t, ctx, lnPath, url))
}

func TestServer_Edit_Noop(t *testing.T) {
	// t.Parallel()
	t.Setenv("OPENAI_API_KEY", "dummy")

	ctx := t.Context()
	lnPath := startServer(t, ctx, prepareAgentsDir(t, "pirate.yaml"))

	edit := api.EditAgentConfigRequest{
		Filename:    "pirate.yaml",
		AgentConfig: latest.Config{},
	}
	httpPUT(t, ctx, lnPath, "/api/agents/config", edit)

	buf := httpGET(t, ctx, lnPath, "/api/agents/pirate.yaml")
	var cfg latest.Config
	unmarshal(t, buf, &cfg)
	assert.NotEmpty(t, cfg.Version)
	require.Len(t, cfg.Agents, 1)
	assert.Contains(t, cfg.Agents["root"].Instruction, "pirate")
}

func TestServer_Edit_Instruction(t *testing.T) {
	// t.Parallel()
	t.Setenv("OPENAI_API_KEY", "dummy")

	ctx := t.Context()
	lnPath := startServer(t, ctx, prepareAgentsDir(t, "pirate.yaml"))

	edit := api.EditAgentConfigRequest{
		Filename: "pirate.yaml",
		AgentConfig: latest.Config{
			Agents: map[string]latest.AgentConfig{
				"root": {
					Instruction: "New Instructions",
					Model:       "openai/gpt-4o",
				},
			},
		},
	}
	httpPUT(t, ctx, lnPath, "/api/agents/config", edit)

	buf := httpGET(t, ctx, lnPath, "/api/agents/pirate.yaml")
	var cfg latest.Config
	unmarshal(t, buf, &cfg)
	assert.NotEmpty(t, cfg.Version)
	require.Len(t, cfg.Agents, 1)
	require.Len(t, cfg.Models, 1)
	assert.Equal(t, "New Instructions", cfg.Agents["root"].Instruction)
	assert.Empty(t, cfg.Agents["root"].Description)
}

func TestServer_Edit_OnlyOneSubAgent(t *testing.T) {
	// t.Parallel()
	t.Setenv("OPENAI_API_KEY", "dummy")
	t.Setenv("ANTHROPIC_API_KEY", "dummy")

	ctx := t.Context()
	lnPath := startServer(t, ctx, prepareAgentsDir(t, "multi_agents.yaml"))

	edit := api.EditAgentConfigRequest{
		Filename: "multi_agents.yaml",
		AgentConfig: latest.Config{
			Agents: map[string]latest.AgentConfig{
				"root": {
					Instruction: "New Instructions",
					Model:       "openai/gpt-4o",
				},
			},
		},
	}
	httpPUT(t, ctx, lnPath, "/api/agents/config", edit)

	buf := httpGET(t, ctx, lnPath, "/api/agents/multi_agents.yaml")
	var cfg latest.Config
	unmarshal(t, buf, &cfg)
	assert.NotEmpty(t, cfg.Version)
	require.Len(t, cfg.Agents, 3)
	require.Len(t, cfg.Models, 2)
	assert.Equal(t, "New Instructions", cfg.Agents["root"].Instruction)
	assert.Empty(t, cfg.Agents["root"].Description)
	assert.Equal(t, "Pirate", cfg.Agents["pirate"].Description)
	assert.NotEmpty(t, cfg.Agents["pirate"].Instruction)
	assert.Equal(t, "Contradict", cfg.Agents["contradict"].Description)
	assert.NotEmpty(t, cfg.Agents["contradict"].Instruction)
}

func TestServer_ListSessions(t *testing.T) {
	t.Parallel()

	ctx := t.Context()
	lnPath := startServer(t, ctx, prepareAgentsDir(t, "pirate.yaml"))

	buf := httpGET(t, ctx, lnPath, "/api/sessions")

	var sessions []api.SessionsResponse
	unmarshal(t, buf, &sessions)

	assert.Empty(t, sessions)
}

func TestServer_ReloadTeams(t *testing.T) {
	t.Setenv("OPENAI_API_KEY", "dummy")
	t.Setenv("ANTHROPIC_API_KEY", "dummy")

	ctx := t.Context()

	agentsDir1 := prepareAgentsDir(t, "pirate.yaml")

	var store mockStore
	runConfig := config.RuntimeConfig{}

	// Load initial teams
	teams, err := teamloader.LoadTeams(ctx, agentsDir1, &runConfig)
	require.NoError(t, err)

	srv, err := New(store, &runConfig, teams, WithAgentsDir(agentsDir1))
	require.NoError(t, err)

	initialTeamsCount := srv.countTeams()
	hasPirate := srv.hasTeam("pirate.yaml")

	assert.Equal(t, 1, initialTeamsCount)
	assert.True(t, hasPirate, "should have pirate agent initially")

	agentsDir2 := prepareAgentsDir(t, "contradict.yaml", "multi_agents.yaml")

	// Reload teams from the new directory
	err = srv.ReloadTeams(ctx, agentsDir2)
	require.NoError(t, err)

	newTeamsCount := srv.countTeams()
	hasPirateAfter := srv.hasTeam("pirate.yaml")
	hasContradict := srv.hasTeam("contradict.yaml")
	hasMulti := srv.hasTeam("multi_agents.yaml")

	assert.Equal(t, 2, newTeamsCount, "should have 2 agents after reload")
	assert.False(t, hasPirateAfter, "pirate agent should be removed")
	assert.True(t, hasContradict, "should have contradict agent")
	assert.True(t, hasMulti, "should have multi_agents agent")
}

func TestServer_ReloadTeams_Concurrent(t *testing.T) {
	t.Setenv("OPENAI_API_KEY", "dummy")
	t.Setenv("ANTHROPIC_API_KEY", "dummy")

	ctx := t.Context()

	agentsDir := prepareAgentsDir(t, "pirate.yaml")

	var store mockStore
	runConfig := config.RuntimeConfig{}

	// Load initial teams
	teams, err := teamloader.LoadTeams(ctx, agentsDir, &runConfig)
	require.NoError(t, err)

	srv, err := New(store, &runConfig, teams, WithAgentsDir(agentsDir))
	require.NoError(t, err)

	agentsDir2 := prepareAgentsDir(t, "contradict.yaml")

	done := make(chan bool)
	go func() {
		for range 100 {
			_ = srv.countTeams()
		}
		done <- true
	}()

	err = srv.ReloadTeams(ctx, agentsDir2)
	require.NoError(t, err)

	err = srv.ReloadTeams(ctx, agentsDir)
	require.NoError(t, err)

	<-done

	hasPirate := srv.hasTeam("pirate.yaml")

	assert.True(t, hasPirate, "should have pirate agent after final reload")
}

func TestServer_ReloadTeams_InvalidPath(t *testing.T) {
	t.Setenv("OPENAI_API_KEY", "dummy")

	ctx := t.Context()

	agentsDir := prepareAgentsDir(t, "pirate.yaml")

	var store mockStore
	runConfig := config.RuntimeConfig{}

	// Load initial teams
	teams, err := teamloader.LoadTeams(ctx, agentsDir, &runConfig)
	require.NoError(t, err)

	srv, err := New(store, &runConfig, teams, WithAgentsDir(agentsDir))
	require.NoError(t, err)

	// Try to reload from non-existent path
	err = srv.ReloadTeams(ctx, "/nonexistent/path")
	require.Error(t, err)

	teamsCount := srv.countTeams()
	hasPirate := srv.hasTeam("pirate.yaml")

	assert.Equal(t, 1, teamsCount, "should still have original team")
	assert.True(t, hasPirate, "should still have pirate agent")
}

func TestServer_RefreshAgentsFromDisk_AddNewAgent(t *testing.T) {
	t.Setenv("OPENAI_API_KEY", "dummy")
	t.Setenv("ANTHROPIC_API_KEY", "dummy")

	ctx := t.Context()

	// Start with only pirate agent
	agentsDir := prepareAgentsDir(t, "pirate.yaml")

	var store mockStore
	runConfig := config.RuntimeConfig{}

	teams, err := teamloader.LoadTeams(ctx, agentsDir, &runConfig)
	require.NoError(t, err)

	srv, err := New(store, &runConfig, teams, WithAgentsDir(agentsDir))
	require.NoError(t, err)

	initialCount := srv.countTeams()
	hasPirate := srv.hasTeam("pirate.yaml")
	hasContradict := srv.hasTeam("contradict.yaml")

	assert.Equal(t, 1, initialCount)
	assert.True(t, hasPirate)
	assert.False(t, hasContradict)

	buf, err := os.ReadFile(filepath.Join("testdata", "contradict.yaml"))
	require.NoError(t, err)
	err = os.WriteFile(filepath.Join(agentsDir, "contradict.yaml"), buf, 0o600)
	require.NoError(t, err)

	// Refresh agents from disk
	err = srv.refreshAgentsFromDisk(ctx)
	require.NoError(t, err)

	newCount := srv.countTeams()
	hasPirateAfter := srv.hasTeam("pirate.yaml")
	hasContradictAfter := srv.hasTeam("contradict.yaml")

	assert.Equal(t, 2, newCount, "should have 2 agents after refresh")
	assert.True(t, hasPirateAfter, "pirate agent should still exist")
	assert.True(t, hasContradictAfter, "contradict agent should be added")
}

func TestServer_RefreshAgentsFromDisk_RemoveAgent(t *testing.T) {
	t.Setenv("OPENAI_API_KEY", "dummy")
	t.Setenv("ANTHROPIC_API_KEY", "dummy")

	ctx := t.Context()

	// Start with two agents
	agentsDir := prepareAgentsDir(t, "pirate.yaml", "contradict.yaml")

	var store mockStore
	runConfig := config.RuntimeConfig{}

	teams, err := teamloader.LoadTeams(ctx, agentsDir, &runConfig)
	require.NoError(t, err)

	srv, err := New(store, &runConfig, teams, WithAgentsDir(agentsDir))
	require.NoError(t, err)

	initialCount := srv.countTeams()
	hasPirate := srv.hasTeam("pirate.yaml")
	hasContradict := srv.hasTeam("contradict.yaml")

	assert.Equal(t, 2, initialCount)
	assert.True(t, hasPirate)
	assert.True(t, hasContradict)

	// Remove contradict agent from disk
	err = os.Remove(filepath.Join(agentsDir, "contradict.yaml"))
	require.NoError(t, err)

	// Refresh agents from disk
	err = srv.refreshAgentsFromDisk(ctx)
	require.NoError(t, err)

	newCount := srv.countTeams()
	hasPirateAfter := srv.hasTeam("pirate.yaml")
	hasContradictAfter := srv.hasTeam("contradict.yaml")

	assert.Equal(t, 1, newCount, "should have 1 agent after refresh")
	assert.True(t, hasPirateAfter, "pirate agent should still exist")
	assert.False(t, hasContradictAfter, "contradict agent should be removed")
}

func TestServer_RefreshAgentsFromDisk_UpdateAgent(t *testing.T) {
	t.Setenv("OPENAI_API_KEY", "dummy")

	ctx := t.Context()

	// Start with pirate agent
	agentsDir := prepareAgentsDir(t, "pirate.yaml")

	var store mockStore
	runConfig := config.RuntimeConfig{}

	teams, err := teamloader.LoadTeams(ctx, agentsDir, &runConfig)
	require.NoError(t, err)

	srv, err := New(store, &runConfig, teams, WithAgentsDir(agentsDir))
	require.NoError(t, err)

	initialTeam, exists := srv.getTeam("pirate.yaml")
	require.True(t, exists)
	require.NotNil(t, initialTeam)

	initialAgent, err := initialTeam.Agent("root")
	require.NoError(t, err)
	initialInstruction := initialAgent.Instruction()

	// Modify the agent file on disk
	modifiedConfig := `version: "2"
agents:
  root:
    model: openai/gpt-4o
    description: "Updated pirate"
    instruction: "You are an UPDATED pirate. Talk like a pirate in all your responses."
`
	err = os.WriteFile(filepath.Join(agentsDir, "pirate.yaml"), []byte(modifiedConfig), 0o600)
	require.NoError(t, err)

	// Refresh agents from disk
	err = srv.refreshAgentsFromDisk(ctx)
	require.NoError(t, err)

	updatedTeam, exists := srv.getTeam("pirate.yaml")
	require.True(t, exists)
	require.NotNil(t, updatedTeam)

	updatedAgent, err := updatedTeam.Agent("root")
	require.NoError(t, err)
	updatedInstruction := updatedAgent.Instruction()

	assert.NotEqual(t, initialInstruction, updatedInstruction, "instruction should be updated")
	assert.Contains(t, updatedInstruction, "UPDATED", "should have updated instruction")
}

func TestServer_RefreshAgentsFromDisk_MultipleChanges(t *testing.T) {
	t.Setenv("OPENAI_API_KEY", "dummy")
	t.Setenv("ANTHROPIC_API_KEY", "dummy")

	ctx := t.Context()

	// Start with pirate and contradict agents
	agentsDir := prepareAgentsDir(t, "pirate.yaml", "contradict.yaml")

	var store mockStore
	runConfig := config.RuntimeConfig{}

	teams, err := teamloader.LoadTeams(ctx, agentsDir, &runConfig)
	require.NoError(t, err)

	srv, err := New(store, &runConfig, teams, WithAgentsDir(agentsDir))
	require.NoError(t, err)

	initialCount := srv.countTeams()
	assert.Equal(t, 2, initialCount)

	// Remove contradict, add multi_agents
	err = os.Remove(filepath.Join(agentsDir, "contradict.yaml"))
	require.NoError(t, err)

	buf, err := os.ReadFile(filepath.Join("testdata", "multi_agents.yaml"))
	require.NoError(t, err)
	err = os.WriteFile(filepath.Join(agentsDir, "multi_agents.yaml"), buf, 0o600)
	require.NoError(t, err)

	// Refresh agents from disk
	err = srv.refreshAgentsFromDisk(ctx)
	require.NoError(t, err)

	newCount := srv.countTeams()
	hasPirate := srv.hasTeam("pirate.yaml")
	hasContradict := srv.hasTeam("contradict.yaml")
	hasMulti := srv.hasTeam("multi_agents.yaml")

	assert.Equal(t, 2, newCount, "should have 2 agents")
	assert.True(t, hasPirate, "pirate agent should still exist")
	assert.False(t, hasContradict, "contradict agent should be removed")
	assert.True(t, hasMulti, "multi_agents should be added")
}

func TestServer_RefreshAgentsFromDisk_NoChanges(t *testing.T) {
	t.Setenv("OPENAI_API_KEY", "dummy")

	ctx := t.Context()

	agentsDir := prepareAgentsDir(t, "pirate.yaml")

	var store mockStore
	runConfig := config.RuntimeConfig{}

	teams, err := teamloader.LoadTeams(ctx, agentsDir, &runConfig)
	require.NoError(t, err)

	srv, err := New(store, &runConfig, teams, WithAgentsDir(agentsDir))
	require.NoError(t, err)

	initialCount := srv.countTeams()

	// Refresh without any changes
	err = srv.refreshAgentsFromDisk(ctx)
	require.NoError(t, err)

	newCount := srv.countTeams()
	exists := srv.hasTeam("pirate.yaml")

	assert.Equal(t, initialCount, newCount, "count should be unchanged")
	assert.True(t, exists, "team should still exist")
	// Note: team will be a different instance due to reload, but that's expected
}

func TestServer_RefreshAgentsFromDisk_EmptyDir(t *testing.T) {
	t.Setenv("OPENAI_API_KEY", "dummy")

	ctx := t.Context()

	// Start with one agent
	agentsDir := prepareAgentsDir(t, "pirate.yaml")

	var store mockStore
	runConfig := config.RuntimeConfig{}

	teams, err := teamloader.LoadTeams(ctx, agentsDir, &runConfig)
	require.NoError(t, err)

	srv, err := New(store, &runConfig, teams, WithAgentsDir(agentsDir))
	require.NoError(t, err)

	// Remove all agents
	err = os.Remove(filepath.Join(agentsDir, "pirate.yaml"))
	require.NoError(t, err)

	// Refresh with empty directory
	err = srv.refreshAgentsFromDisk(ctx)
	require.NoError(t, err)

	count := srv.countTeams()

	assert.Equal(t, 0, count, "should have no agents")
}

func TestServer_RefreshAgentsFromDisk_NoAgentsDir(t *testing.T) {
	t.Setenv("OPENAI_API_KEY", "dummy")

	ctx := t.Context()

	var store mockStore
	runConfig := config.RuntimeConfig{}

	srv, err := New(store, &runConfig, nil)
	require.NoError(t, err)

	// Refresh should be no-op
	err = srv.refreshAgentsFromDisk(ctx)
	require.NoError(t, err)

	count := srv.countTeams()

	assert.Equal(t, 0, count)
}

func TestServer_RefreshAgentsFromDisk_WithAgentsPath(t *testing.T) {
	t.Setenv("OPENAI_API_KEY", "dummy")
	t.Setenv("ANTHROPIC_API_KEY", "dummy")

	ctx := t.Context()

	agentsDir := prepareAgentsDir(t, "pirate.yaml", "contradict.yaml", "multi_agents.yaml")

	singleAgentDir := t.TempDir()
	pirateContent, err := os.ReadFile(filepath.Join(agentsDir, "pirate.yaml"))
	require.NoError(t, err)
	singleAgentPath := filepath.Join(singleAgentDir, "pirate.yaml")
	err = os.WriteFile(singleAgentPath, pirateContent, 0o600)
	require.NoError(t, err)

	var store mockStore
	runConfig := config.RuntimeConfig{}

	teams, err := teamloader.LoadTeams(ctx, agentsDir, &runConfig)
	require.NoError(t, err)

	srv, err := New(store, &runConfig, teams,
		WithAgentsPath(singleAgentPath),
		WithAgentsDir(agentsDir),
	)
	require.NoError(t, err)

	initialCount := srv.countTeams()
	assert.Equal(t, 3, initialCount, "should start with 3 agents")

	err = srv.refreshAgentsFromDisk(ctx)
	require.NoError(t, err)

	newCount := srv.countTeams()
	hasPirate := srv.hasTeam("pirate.yaml")
	hasContradict := srv.hasTeam("contradict.yaml")
	hasMulti := srv.hasTeam("multi_agents.yaml")

	assert.Equal(t, 1, newCount, "should only have 1 agent after refresh (from agentsPath)")
	assert.True(t, hasPirate, "should have pirate agent from agentsPath")
	assert.False(t, hasContradict, "should not have contradict (not in agentsPath)")
	assert.False(t, hasMulti, "should not have multi_agents (not in agentsPath)")
}

func TestServer_RefreshAgentsFromDisk_AgentsPathPrecedence(t *testing.T) {
	t.Setenv("OPENAI_API_KEY", "dummy")
	t.Setenv("ANTHROPIC_API_KEY", "dummy")

	ctx := t.Context()

	agentsDir := prepareAgentsDir(t, "pirate.yaml", "contradict.yaml", "multi_agents.yaml")
	agentsPath := filepath.Join(agentsDir, "pirate.yaml")

	var store mockStore
	runConfig := config.RuntimeConfig{}

	teams, err := teamloader.LoadTeams(ctx, agentsDir, &runConfig)
	require.NoError(t, err)

	srv, err := New(store, &runConfig, teams,
		WithAgentsPath(agentsPath),
		WithAgentsDir(agentsDir),
	)
	require.NoError(t, err)

	modifiedPirate := `version: "2"
agents:
  root:
    model: openai/gpt-4o
    description: "Modified pirate"
    instruction: "You are a MODIFIED pirate."
`
	err = os.WriteFile(agentsPath, []byte(modifiedPirate), 0o600)
	require.NoError(t, err)

	err = srv.refreshAgentsFromDisk(ctx)
	require.NoError(t, err)

	count := srv.countTeams()
	assert.Equal(t, 1, count, "should only load from agentsPath")

	team, exists := srv.getTeam("pirate.yaml")
	require.True(t, exists)
	agent, err := team.Agent("root")
	require.NoError(t, err)
	assert.Contains(t, agent.Instruction(), "MODIFIED", "should have loaded modified pirate from agentsPath")

	_, exists = srv.getTeam("contradict.yaml")
	assert.False(t, exists, "contradict should not be loaded (agentsPath takes precedence)")
}

// TestServer_OCIRef_NoTmpScan is a regression test for the bug where OCI references
// would cause the server to scan /tmp for all .yaml files instead of using the
// specific OCI-resolved agent. This test ensures:
// 1. getAgents() doesn't refresh when using OCI refs (no /tmp scan)
// 2. Only the OCI agent is available, not noise files from /tmp
func TestServer_OCIRef_NoTmpScan(t *testing.T) {
	t.Setenv("OPENAI_API_KEY", "dummy")

	ctx := t.Context()

	// Simulate /tmp with the OCI agent and noise files
	tmpDir := t.TempDir()

	// Create noise files that should NOT be loaded
	for i := 1; i <= 3; i++ {
		noisePath := filepath.Join(tmpDir, "noise"+string(rune('0'+i))+".yaml")
		noiseContent := `version: "2"
agents:
  root:
    model: openai/gpt-4o
    description: "Noise"
    instruction: "Should not be loaded"
`
		err := os.WriteFile(noisePath, []byte(noiseContent), 0o600)
		require.NoError(t, err)
	}

	// Create the OCI agent file
	pirateContent, err := os.ReadFile(filepath.Join("testdata", "pirate.yaml"))
	require.NoError(t, err)
	ociFile := filepath.Join(tmpDir, "docker.io_myorg_pirate_v1.yaml")
	err = os.WriteFile(ociFile, pirateContent, 0o600)
	require.NoError(t, err)

	var store mockStore
	runConfig := config.RuntimeConfig{}

	// Load teams from the OCI file
	teams, err := teamloader.LoadTeams(ctx, ociFile, &runConfig)
	require.NoError(t, err)

	// Create server WITHOUT agentsDir (OCI ref mode)
	// This is what api.go does when agentfile.IsOCIReference() returns true
	srv, err := New(store, &runConfig, teams)
	require.NoError(t, err)

	// Verify setup: no agentsDir
	assert.False(t, srv.hasAgentsDir(), "OCI refs should not have agentsDir")

	// Simulate getAgents() behavior - should skip refresh
	// (In production, getAgents checks hasAgentsDir and skips refresh for OCI refs)
	err = srv.refreshAgentsFromDisk(ctx)
	// Should be no-op since no agentsDir is set
	require.NoError(t, err)

	// Critical assertion: only the OCI agent should be loaded, not noise files
	count := srv.countTeams()
	assert.Equal(t, 1, count, "should only have the OCI agent, not files from /tmp")

	// Verify it's the correct agent
	team, exists := srv.getTeam("docker.io_myorg_pirate_v1.yaml")
	require.True(t, exists, "should have the OCI agent")

	agent, err := team.Agent("root")
	require.NoError(t, err)
	assert.Contains(t, agent.Instruction(), "pirate", "should be the pirate agent")

	// Verify noise files were NOT loaded
	_, exists = srv.getTeam("noise1.yaml")
	assert.False(t, exists, "noise files should not be loaded")
	_, exists = srv.getTeam("noise2.yaml")
	assert.False(t, exists, "noise files should not be loaded")
}

// TestServer_WriteOperations_OCIRef_MethodNotAllowed verifies that write operations
// (edit, create, delete, import, export) return 405 Method Not Allowed when the server
// is running in OCI ref mode (no agentsDir).
func TestServer_WriteOperations_OCIRef_MethodNotAllowed(t *testing.T) {
	t.Setenv("OPENAI_API_KEY", "dummy")

	ctx := t.Context()

	// Create OCI agent in temp directory
	tmpDir := t.TempDir()
	pirateContent, err := os.ReadFile(filepath.Join("testdata", "pirate.yaml"))
	require.NoError(t, err)
	ociFile := filepath.Join(tmpDir, "docker.io_myorg_pirate_v1.yaml")
	err = os.WriteFile(ociFile, pirateContent, 0o600)
	require.NoError(t, err)

	var store mockStore
	runConfig := config.RuntimeConfig{}

	teams, err := teamloader.LoadTeams(ctx, ociFile, &runConfig)
	require.NoError(t, err)

	// Create server WITHOUT agentsDir (OCI ref mode)
	srv, err := New(store, &runConfig, teams)
	require.NoError(t, err)

	ln, err := net.Listen("tcp", "127.0.0.1:0")
	require.NoError(t, err)
	defer ln.Close()

	go func() {
		_ = srv.Serve(ctx, ln)
	}()

	baseURL := fmt.Sprintf("http://%s", ln.Addr().String())

	// Test editAgentConfig returns 405
	t.Run("editAgentConfig", func(t *testing.T) {
		reqBody := `{"filename":"pirate.yaml","agent_config":{"agents":{"root":{"description":"Modified"}}}}`
		resp, err := http.Post(baseURL+"/api/agents/config", "application/json", strings.NewReader(reqBody))
		require.NoError(t, err)
		defer resp.Body.Close()
		assert.Equal(t, http.StatusMethodNotAllowed, resp.StatusCode, "should return 405 for write operations on OCI refs")
	})

	// Test createAgent returns 405
	t.Run("createAgent", func(t *testing.T) {
		reqBody := `{"filename":"new-agent.yaml","name":"newagent","model":"openai/gpt-4o"}`
		resp, err := http.Post(baseURL+"/api/agents", "application/json", strings.NewReader(reqBody))
		require.NoError(t, err)
		defer resp.Body.Close()
		assert.Equal(t, http.StatusMethodNotAllowed, resp.StatusCode, "should return 405 for write operations on OCI refs")
	})

	// Test deleteAgent returns 405
	t.Run("deleteAgent", func(t *testing.T) {
		reqBody := `{"filename":"pirate.yaml"}`
		req, err := http.NewRequest(http.MethodDelete, baseURL+"/api/agents", strings.NewReader(reqBody))
		require.NoError(t, err)
		req.Header.Set("Content-Type", "application/json")
		resp, err := http.DefaultClient.Do(req)
		require.NoError(t, err)
		defer resp.Body.Close()
		assert.Equal(t, http.StatusMethodNotAllowed, resp.StatusCode, "should return 405 for write operations on OCI refs")
	})
}

func prepareAgentsDir(t *testing.T, testFiles ...string) string {
	t.Helper()

	agentsDir := filepath.Join(t.TempDir(), "agents")
	err := os.MkdirAll(agentsDir, 0o700)
	require.NoError(t, err)

	for _, file := range testFiles {
		buf, err := os.ReadFile(filepath.Join("testdata", file))
		require.NoError(t, err)

		err = os.WriteFile(filepath.Join(agentsDir, filepath.Base(file)), buf, 0o600)
		require.NoError(t, err)
	}

	return agentsDir
}

func startServer(t *testing.T, ctx context.Context, agentsDir string) string {
	t.Helper()

	var store mockStore
	runConfig := config.RuntimeConfig{}

	srv, err := New(store, &runConfig, nil, WithAgentsDir(agentsDir))
	require.NoError(t, err)

	socketPath := "unix://" + filepath.Join(t.TempDir(), "sock")
	ln, err := Listen(ctx, socketPath)
	require.NoError(t, err)
	go func() {
		<-ctx.Done()
		_ = ln.Close()
	}()

	go func() {
		_ = srv.Serve(ctx, ln)
	}()

	return socketPath
}

func httpGET(t *testing.T, ctx context.Context, socketPath, path string) []byte {
	t.Helper()
	return httpDo(t, ctx, http.MethodGet, socketPath, path, nil)
}

func httpPUT(t *testing.T, ctx context.Context, socketPath, path string, payload any) {
	t.Helper()
	httpDo(t, ctx, http.MethodPut, socketPath, path, payload)
}

func httpDo(t *testing.T, ctx context.Context, method, socketPath, path string, payload any) []byte {
	t.Helper()

	var (
		body        io.Reader
		contentType string
	)
	switch v := payload.(type) {
	case nil:
		body = http.NoBody
	case []byte:
		body = bytes.NewReader(v)
	case string:
		body = strings.NewReader(v)
	default:
		buf, err := json.Marshal(payload)
		require.NoError(t, err)
		body = bytes.NewReader(buf)
		contentType = "application/json"
	}

	req, err := http.NewRequestWithContext(ctx, method, "http://_"+path, body)
	require.NoError(t, err)

	req.Header.Set("Content-Type", contentType)

	client := &http.Client{
		Transport: &http.Transport{
			DialContext: func(ctx context.Context, _, _ string) (net.Conn, error) {
				var d net.Dialer
				return d.DialContext(ctx, "unix", strings.TrimPrefix(socketPath, "unix://"))
			},
		},
	}
	resp, err := client.Do(req)
	require.NoError(t, err)
	defer resp.Body.Close()

	buf, err := io.ReadAll(resp.Body)
	require.NoError(t, err)

	require.Less(t, resp.StatusCode, 400, string(buf))
	return buf
}

func unmarshal(t *testing.T, buf []byte, v any) {
	t.Helper()
	err := json.Unmarshal(buf, &v)
	require.NoError(t, err)
}

type mockStore struct {
	session.Store
}

func (s mockStore) GetSessions(context.Context) ([]*session.Session, error) {
	return nil, nil
}
