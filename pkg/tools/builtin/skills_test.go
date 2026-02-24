package builtin

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/docker/cagent/pkg/skills"
)

func TestSkillsToolset_ReadSkillContent_Local(t *testing.T) {
	tmpDir := t.TempDir()
	skillFile := filepath.Join(tmpDir, "SKILL.md")
	require.NoError(t, os.WriteFile(skillFile, []byte("# Local Skill\nDo the thing."), 0o644))

	st := NewSkillsToolset([]skills.Skill{
		{Name: "local-skill", Description: "A local skill", FilePath: skillFile, BaseDir: tmpDir},
	})

	content, err := st.ReadSkillContent("local-skill")
	require.NoError(t, err)
	assert.Equal(t, "# Local Skill\nDo the thing.", content)
}

func TestSkillsToolset_ReadSkillContent_NotFound(t *testing.T) {
	st := NewSkillsToolset([]skills.Skill{
		{Name: "exists", Description: "Exists", FilePath: "/tmp/nonexistent"},
	})

	_, err := st.ReadSkillContent("does-not-exist")
	require.Error(t, err)
	assert.Contains(t, err.Error(), "not found")
}

func TestSkillsToolset_ReadSkillFile(t *testing.T) {
	tmpDir := t.TempDir()
	require.NoError(t, os.WriteFile(filepath.Join(tmpDir, "SKILL.md"), []byte("# Main"), 0o644))
	require.NoError(t, os.MkdirAll(filepath.Join(tmpDir, "references"), 0o755))
	require.NoError(t, os.WriteFile(filepath.Join(tmpDir, "references", "FORMS.md"), []byte("# Forms Reference"), 0o644))

	st := NewSkillsToolset([]skills.Skill{
		{
			Name: "my-skill", Description: "My skill", FilePath: filepath.Join(tmpDir, "SKILL.md"), BaseDir: tmpDir,
			Files: []string{"SKILL.md", "references/FORMS.md"},
		},
	})

	content, err := st.ReadSkillFile("my-skill", "references/FORMS.md")
	require.NoError(t, err)
	assert.Equal(t, "# Forms Reference", content)
}

func TestSkillsToolset_ReadSkillFile_PathTraversal(t *testing.T) {
	tmpDir := t.TempDir()
	require.NoError(t, os.WriteFile(filepath.Join(tmpDir, "SKILL.md"), []byte("# Main"), 0o644))

	st := NewSkillsToolset([]skills.Skill{
		{Name: "my-skill", Description: "My skill", FilePath: filepath.Join(tmpDir, "SKILL.md"), BaseDir: tmpDir},
	})

	_, err := st.ReadSkillFile("my-skill", "../../../etc/passwd")
	require.Error(t, err)
	assert.Contains(t, err.Error(), "invalid file path")

	_, err = st.ReadSkillFile("my-skill", "/etc/passwd")
	require.Error(t, err)
	assert.Contains(t, err.Error(), "invalid file path")
}

func TestSkillsToolset_ReadSkillFile_SkillNotFound(t *testing.T) {
	st := NewSkillsToolset([]skills.Skill{
		{Name: "exists", Description: "Exists", FilePath: "/tmp/test"},
	})

	_, err := st.ReadSkillFile("nonexistent", "SKILL.md")
	require.Error(t, err)
	assert.Contains(t, err.Error(), "not found")
}

func TestSkillsToolset_Instructions(t *testing.T) {
	st := NewSkillsToolset([]skills.Skill{
		{Name: "skill-a", Description: "Does A"},
		{Name: "skill-b", Description: "Does B", Files: []string{"SKILL.md", "references/HELP.md"}},
	})

	instructions := st.Instructions()

	assert.Contains(t, instructions, "read_skill")
	assert.Contains(t, instructions, "read_skill_file")
	assert.Contains(t, instructions, "<available_skills>")
	assert.Contains(t, instructions, "<name>skill-a</name>")
	assert.Contains(t, instructions, "<description>Does A</description>")
	assert.Contains(t, instructions, "<name>skill-b</name>")
	assert.Contains(t, instructions, "<description>Does B</description>")
	assert.Contains(t, instructions, "<files>references/HELP.md</files>")
	// Should NOT contain file system paths
	assert.NotContains(t, instructions, "FilePath")
}

func TestSkillsToolset_Instructions_NoFiles(t *testing.T) {
	st := NewSkillsToolset([]skills.Skill{
		{Name: "simple", Description: "Simple skill"},
	})

	instructions := st.Instructions()

	assert.Contains(t, instructions, "read_skill")
	assert.NotContains(t, instructions, "read_skill_file")
	assert.NotContains(t, instructions, "<files>")
}

func TestSkillsToolset_Instructions_Empty(t *testing.T) {
	st := NewSkillsToolset(nil)
	assert.Empty(t, st.Instructions())

	st = NewSkillsToolset([]skills.Skill{})
	assert.Empty(t, st.Instructions())
}

func TestSkillsToolset_Tools_WithFiles(t *testing.T) {
	st := NewSkillsToolset([]skills.Skill{
		{Name: "test", Description: "Test skill", Files: []string{"SKILL.md", "references/HELP.md"}},
	})

	tools, err := st.Tools(t.Context())
	require.NoError(t, err)
	require.Len(t, tools, 2)

	assert.Equal(t, ToolNameReadSkill, tools[0].Name)
	assert.Equal(t, ToolNameReadSkillFile, tools[1].Name)
}

func TestSkillsToolset_Tools_WithoutFiles(t *testing.T) {
	st := NewSkillsToolset([]skills.Skill{
		{Name: "test", Description: "Test skill"},
	})

	tools, err := st.Tools(t.Context())
	require.NoError(t, err)
	require.Len(t, tools, 1)

	assert.Equal(t, ToolNameReadSkill, tools[0].Name)
}

func TestSkillsToolset_Tools_Empty(t *testing.T) {
	st := NewSkillsToolset(nil)

	tools, err := st.Tools(t.Context())
	require.NoError(t, err)
	assert.Empty(t, tools)
}

func TestSkillsToolset_Skills(t *testing.T) {
	input := []skills.Skill{
		{Name: "a", Description: "A"},
		{Name: "b", Description: "B"},
	}
	st := NewSkillsToolset(input)

	assert.Equal(t, input, st.Skills())
}

func TestSkillsToolset_HandleReadSkill(t *testing.T) {
	tmpDir := t.TempDir()
	skillFile := filepath.Join(tmpDir, "SKILL.md")
	require.NoError(t, os.WriteFile(skillFile, []byte("skill instructions"), 0o644))

	st := NewSkillsToolset([]skills.Skill{
		{Name: "test-skill", Description: "Test", FilePath: skillFile, BaseDir: tmpDir},
	})

	result, err := st.handleReadSkill(t.Context(), readSkillArgs{Name: "test-skill"})
	require.NoError(t, err)
	assert.False(t, result.IsError)
	assert.Contains(t, result.Output, "skill instructions")
}

func TestSkillsToolset_HandleReadSkill_NotFound(t *testing.T) {
	st := NewSkillsToolset([]skills.Skill{
		{Name: "exists", Description: "Exists", FilePath: "/tmp/test"},
	})

	result, err := st.handleReadSkill(t.Context(), readSkillArgs{Name: "missing"})
	require.NoError(t, err)
	assert.True(t, result.IsError)
	assert.Contains(t, result.Output, "not found")
}

func TestSkillsToolset_HandleReadSkillFile(t *testing.T) {
	tmpDir := t.TempDir()
	require.NoError(t, os.WriteFile(filepath.Join(tmpDir, "SKILL.md"), []byte("# Main"), 0o644))
	require.NoError(t, os.MkdirAll(filepath.Join(tmpDir, "scripts"), 0o755))
	require.NoError(t, os.WriteFile(filepath.Join(tmpDir, "scripts", "deploy.sh"), []byte("#!/bin/bash\necho deploy"), 0o644))

	st := NewSkillsToolset([]skills.Skill{
		{
			Name: "my-skill", Description: "My skill", FilePath: filepath.Join(tmpDir, "SKILL.md"), BaseDir: tmpDir,
			Files: []string{"SKILL.md", "scripts/deploy.sh"},
		},
	})

	result, err := st.handleReadSkillFile(t.Context(), readSkillFileArgs{SkillName: "my-skill", Path: "scripts/deploy.sh"})
	require.NoError(t, err)
	assert.False(t, result.IsError)
	assert.Contains(t, result.Output, "echo deploy")
}

func TestSkillsToolset_HandleReadSkillFile_PathTraversal(t *testing.T) {
	tmpDir := t.TempDir()
	require.NoError(t, os.WriteFile(filepath.Join(tmpDir, "SKILL.md"), []byte("# Main"), 0o644))

	st := NewSkillsToolset([]skills.Skill{
		{Name: "my-skill", Description: "My skill", FilePath: filepath.Join(tmpDir, "SKILL.md"), BaseDir: tmpDir},
	})

	result, err := st.handleReadSkillFile(t.Context(), readSkillFileArgs{SkillName: "my-skill", Path: "../../../etc/passwd"})
	require.NoError(t, err)
	assert.True(t, result.IsError)
	assert.Contains(t, result.Output, "invalid file path")
}
