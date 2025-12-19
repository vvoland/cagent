package skills

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestParseFrontmatter(t *testing.T) {
	tests := []struct {
		name     string
		content  string
		wantFM   frontmatter
		wantBody string
	}{
		{
			name: "valid frontmatter",
			content: `---
name: my-skill
description: A test skill
---

# Skill Content`,
			wantFM:   frontmatter{Name: "my-skill", Description: "A test skill"},
			wantBody: "# Skill Content",
		},
		{
			name: "quoted values",
			content: `---
name: "quoted-name"
description: 'single quoted desc'
---

Body`,
			wantFM:   frontmatter{Name: "quoted-name", Description: "single quoted desc"},
			wantBody: "Body",
		},
		{
			name:     "no frontmatter",
			content:  "# Just content\n\nNo frontmatter here.",
			wantFM:   frontmatter{},
			wantBody: "# Just content\n\nNo frontmatter here.",
		},
		{
			name: "only description",
			content: `---
description: Just a description
---

Content`,
			wantFM:   frontmatter{Description: "Just a description"},
			wantBody: "Content",
		},
		{
			name:     "windows line endings",
			content:  "---\r\nname: windows\r\ndescription: Windows skill\r\n---\r\n\r\nBody",
			wantFM:   frontmatter{Name: "windows", Description: "Windows skill"},
			wantBody: "Body",
		},
		{
			name:     "unclosed frontmatter",
			content:  "---\nname: unclosed\ndescription: No closing\n\nBody",
			wantFM:   frontmatter{},
			wantBody: "---\nname: unclosed\ndescription: No closing\n\nBody",
		},
		{
			name: "all optional fields",
			content: `---
name: full-skill
description: A complete skill
license: Apache-2.0
compatibility: Requires docker and git
metadata:
  author: test-org
  version: "1.0"
allowed-tools: Bash(git:*) Read Write
---

Body`,
			wantFM: frontmatter{
				Name:          "full-skill",
				Description:   "A complete skill",
				License:       "Apache-2.0",
				Compatibility: "Requires docker and git",
				Metadata:      map[string]string{"author": "test-org", "version": "1.0"},
				AllowedTools:  []string{"Bash(git:*)", "Read", "Write"},
			},
			wantBody: "Body",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			fm, body := parseFrontmatter(tt.content)
			assert.Equal(t, tt.wantFM, fm)
			assert.Equal(t, tt.wantBody, body)
		})
	}
}

func TestStripQuotes(t *testing.T) {
	tests := []struct {
		input string
		want  string
	}{
		{`"double quoted"`, "double quoted"},
		{`'single quoted'`, "single quoted"},
		{`no quotes`, "no quotes"},
		{`"mismatched'`, `"mismatched'`},
		{`""`, ""},
		{`''`, ""},
		{`"`, `"`},
	}

	for _, tt := range tests {
		t.Run(tt.input, func(t *testing.T) {
			got := stripQuotes(tt.input)
			assert.Equal(t, tt.want, got)
		})
	}
}

func TestIsValidName(t *testing.T) {
	tests := []struct {
		name  string
		valid bool
	}{
		{"pdf-processing", true},
		{"data-analysis", true},
		{"code-review", true},
		{"simple", true},
		{"a1b2c3", true},
		{"skill123", true},
		{"my-skill-name", true},
		{"", false},
		{"PDF-Processing", false},
		{"-pdf", false},
		{"pdf-", false},
		{"pdf--processing", false},
		{"pdf_processing", false},
		{"pdf processing", false},
		{"pdf.processing", false},
		{strings.Repeat("a", 64), true},
		{strings.Repeat("a", 65), false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := isValidName(tt.name)
			assert.Equal(t, tt.valid, got, "isValidName(%q) = %v, want %v", tt.name, got, tt.valid)
		})
	}
}

func TestLoadSkillsFromDir_Claude(t *testing.T) {
	tmpDir := t.TempDir()

	skillDir := filepath.Join(tmpDir, "pdf-extractor")
	require.NoError(t, os.MkdirAll(skillDir, 0o755))

	skillContent := `---
description: Extract text from PDF files
---

# PDF Extraction

Use pdftotext to extract content.
`
	require.NoError(t, os.WriteFile(filepath.Join(skillDir, "SKILL.md"), []byte(skillContent), 0o644))

	skills := loadSkillsFromDir(tmpDir, formatClaude)

	require.Len(t, skills, 1)
	assert.Equal(t, "pdf-extractor", skills[0].Name)
	assert.Equal(t, "Extract text from PDF files", skills[0].Description)
	assert.Equal(t, filepath.Join(skillDir, "SKILL.md"), skills[0].FilePath)
	assert.Equal(t, skillDir, skills[0].BaseDir)
}

func TestLoadSkillsFromDir_Codex(t *testing.T) {
	tmpDir := t.TempDir()

	nestedDir := filepath.Join(tmpDir, "db", "migrate")
	require.NoError(t, os.MkdirAll(nestedDir, 0o755))

	skillContent := `---
name: migrate
description: Database migration helper
---

# DB Migration

Run migrations with care.
`
	require.NoError(t, os.WriteFile(filepath.Join(nestedDir, "SKILL.md"), []byte(skillContent), 0o644))

	skills := loadSkillsFromDir(tmpDir, formatCodex)

	require.Len(t, skills, 1)
	assert.Equal(t, "migrate", skills[0].Name)
	assert.Equal(t, "Database migration helper", skills[0].Description)
	assert.Equal(t, filepath.Join(nestedDir, "SKILL.md"), skills[0].FilePath)
	assert.Equal(t, nestedDir, skills[0].BaseDir)
}

func TestLoadSkillsFromDir_SkipHiddenAndSymlinks(t *testing.T) {
	tmpDir := t.TempDir()

	hiddenDir := filepath.Join(tmpDir, ".hidden-skill")
	require.NoError(t, os.MkdirAll(hiddenDir, 0o755))
	require.NoError(t, os.WriteFile(filepath.Join(hiddenDir, "SKILL.md"), []byte("---\ndescription: Hidden\n---\n"), 0o644))

	skills := loadSkillsFromDir(tmpDir, formatClaude)
	assert.Empty(t, skills)
}

func TestLoadSkillsFromDir_SkipNoDescription(t *testing.T) {
	tmpDir := t.TempDir()

	skillDir := filepath.Join(tmpDir, "no-desc")
	require.NoError(t, os.MkdirAll(skillDir, 0o755))

	skillContent := `---
name: no-description
---

# No Description

This skill has no description field.
`
	require.NoError(t, os.WriteFile(filepath.Join(skillDir, "SKILL.md"), []byte(skillContent), 0o644))

	skills := loadSkillsFromDir(tmpDir, formatClaude)
	assert.Empty(t, skills)
}

func TestLoadSkillsFromDir_SkipNameMismatch(t *testing.T) {
	tmpDir := t.TempDir()

	skillDir := filepath.Join(tmpDir, "actual-dir-name")
	require.NoError(t, os.MkdirAll(skillDir, 0o755))

	skillContent := `---
name: different-name
description: A skill with mismatched name
---

# Mismatched
`
	require.NoError(t, os.WriteFile(filepath.Join(skillDir, "SKILL.md"), []byte(skillContent), 0o644))

	skills := loadSkillsFromDir(tmpDir, formatClaude)
	assert.Empty(t, skills, "skill with name not matching directory should be skipped")
}

func TestLoadSkillsFromDir_SkipInvalidName(t *testing.T) {
	tmpDir := t.TempDir()

	skillDir := filepath.Join(tmpDir, "Invalid-Name")
	require.NoError(t, os.MkdirAll(skillDir, 0o755))

	skillContent := `---
name: Invalid-Name
description: A skill with invalid name format
---

# Invalid
`
	require.NoError(t, os.WriteFile(filepath.Join(skillDir, "SKILL.md"), []byte(skillContent), 0o644))

	skills := loadSkillsFromDir(tmpDir, formatClaude)
	assert.Empty(t, skills, "skill with invalid name format should be skipped")
}

func TestLoadSkillsFromDir_SkipDescriptionTooLong(t *testing.T) {
	tmpDir := t.TempDir()

	skillDir := filepath.Join(tmpDir, "long-desc")
	require.NoError(t, os.MkdirAll(skillDir, 0o755))

	longDesc := strings.Repeat("a", 1025)
	skillContent := "---\nname: long-desc\ndescription: " + longDesc + "\n---\n\n# Long\n"
	require.NoError(t, os.WriteFile(filepath.Join(skillDir, "SKILL.md"), []byte(skillContent), 0o644))

	skills := loadSkillsFromDir(tmpDir, formatClaude)
	assert.Empty(t, skills, "skill with description > 1024 chars should be skipped")
}

func TestLoadSkillsFromDir_SkipCompatibilityTooLong(t *testing.T) {
	tmpDir := t.TempDir()

	skillDir := filepath.Join(tmpDir, "long-compat")
	require.NoError(t, os.MkdirAll(skillDir, 0o755))

	longCompat := strings.Repeat("a", 501)
	skillContent := "---\nname: long-compat\ndescription: A skill\ncompatibility: " + longCompat + "\n---\n\n# Long\n"
	require.NoError(t, os.WriteFile(filepath.Join(skillDir, "SKILL.md"), []byte(skillContent), 0o644))

	skills := loadSkillsFromDir(tmpDir, formatClaude)
	assert.Empty(t, skills, "skill with compatibility > 500 chars should be skipped")
}

func TestLoadSkillsFromDir_AllOptionalFields(t *testing.T) {
	tmpDir := t.TempDir()

	skillDir := filepath.Join(tmpDir, "full-skill")
	require.NoError(t, os.MkdirAll(skillDir, 0o755))

	skillContent := `---
name: full-skill
description: A complete skill with all fields
license: Apache-2.0
compatibility: Requires docker
metadata:
  author: test-org
  version: "2.0"
allowed-tools: Bash(git:*) Read
---

# Full Skill
`
	require.NoError(t, os.WriteFile(filepath.Join(skillDir, "SKILL.md"), []byte(skillContent), 0o644))

	skills := loadSkillsFromDir(tmpDir, formatClaude)

	require.Len(t, skills, 1)
	assert.Equal(t, "full-skill", skills[0].Name)
	assert.Equal(t, "A complete skill with all fields", skills[0].Description)
	assert.Equal(t, "Apache-2.0", skills[0].License)
	assert.Equal(t, "Requires docker", skills[0].Compatibility)
	assert.Equal(t, map[string]string{"author": "test-org", "version": "2.0"}, skills[0].Metadata)
	assert.Equal(t, []string{"Bash(git:*)", "Read"}, skills[0].AllowedTools)
}

func TestLoadSkillsFromDir_NonExistentDir(t *testing.T) {
	skills := loadSkillsFromDir("/nonexistent/path/12345", formatClaude)
	assert.Empty(t, skills)
}

func TestBuildSkillsPrompt(t *testing.T) {
	skills := []Skill{
		{
			Name:        "pdf-extractor",
			Description: "Extract text from PDFs",
			FilePath:    "/home/user/.claude/skills/pdf-extractor/SKILL.md",
			BaseDir:     "/home/user/.claude/skills/pdf-extractor",
		},
		{
			Name:        "code-review",
			Description: "Perform code reviews",
			FilePath:    "/project/.claude/skills/code-review/SKILL.md",
			BaseDir:     "/project/.claude/skills/code-review",
		},
	}

	prompt := BuildSkillsPrompt(skills)

	assert.Contains(t, prompt, "<available_skills>")
	assert.Contains(t, prompt, "</available_skills>")
	assert.Contains(t, prompt, "<skill>")
	assert.Contains(t, prompt, "</skill>")
	assert.Contains(t, prompt, "<name>pdf-extractor</name>")
	assert.Contains(t, prompt, "<description>Extract text from PDFs</description>")
	assert.Contains(t, prompt, "<location>/home/user/.claude/skills/pdf-extractor/SKILL.md</location>")
	assert.Contains(t, prompt, "<name>code-review</name>")
	assert.Contains(t, prompt, "<description>Perform code reviews</description>")
	assert.Contains(t, prompt, "<location>/project/.claude/skills/code-review/SKILL.md</location>")
	assert.Contains(t, prompt, "use the read_file tool to load the skill's SKILL.md file")
	assert.Contains(t, prompt, "description indicates what it does and when to use it")
}

func TestBuildSkillsPrompt_Empty(t *testing.T) {
	prompt := BuildSkillsPrompt(nil)
	assert.Empty(t, prompt)

	prompt = BuildSkillsPrompt([]Skill{})
	assert.Empty(t, prompt)
}

func TestLoad_Integration(t *testing.T) {
	originalWd, err := os.Getwd()
	require.NoError(t, err)
	defer func() { _ = os.Chdir(originalWd) }()

	tmpDir := t.TempDir()
	require.NoError(t, os.Chdir(tmpDir))

	claudeProjectDir := filepath.Join(tmpDir, ".claude", "skills", "test-skill")
	require.NoError(t, os.MkdirAll(claudeProjectDir, 0o755))

	skillContent := `---
description: Test project skill
---

# Test Skill
`
	require.NoError(t, os.WriteFile(filepath.Join(claudeProjectDir, "SKILL.md"), []byte(skillContent), 0o644))

	skills := Load()

	found := false
	for _, s := range skills {
		if s.Name == "test-skill" {
			found = true
			assert.Equal(t, "Test project skill", s.Description)
			break
		}
	}
	assert.True(t, found, "Expected to find test-skill from project directory")
}
