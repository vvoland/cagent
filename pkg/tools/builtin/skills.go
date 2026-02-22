package builtin

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/docker/cagent/pkg/skills"
	"github.com/docker/cagent/pkg/tools"
)

const (
	ToolNameReadSkill     = "read_skill"
	ToolNameReadSkillFile = "read_skill_file"
)

var (
	_ tools.ToolSet      = (*SkillsToolset)(nil)
	_ tools.Instructable = (*SkillsToolset)(nil)
)

// SkillsToolset provides the read_skill and read_skill_file tools that let an
// agent load skill content and supporting resources by name. It hides whether
// a skill is local or remote — the agent just sees a name and description.
type SkillsToolset struct {
	skills []skills.Skill
}

func NewSkillsToolset(loadedSkills []skills.Skill) *SkillsToolset {
	return &SkillsToolset{
		skills: loadedSkills,
	}
}

// Skills returns the loaded skills (used by the app layer for slash commands).
func (s *SkillsToolset) Skills() []skills.Skill {
	return s.skills
}

func (s *SkillsToolset) findSkill(name string) *skills.Skill {
	for i := range s.skills {
		if s.skills[i].Name == name {
			return &s.skills[i]
		}
	}
	return nil
}

// ReadSkillContent returns the content of a skill's SKILL.md by name.
func (s *SkillsToolset) ReadSkillContent(name string) (string, error) {
	skill := s.findSkill(name)
	if skill == nil {
		return "", fmt.Errorf("skill %q not found", name)
	}

	content, err := readFileContent(skill.FilePath)
	if err != nil {
		return "", err
	}

	return content, nil
}

// ReadSkillFile returns the content of a supporting file within a skill.
// The path is relative to the skill's base directory (e.g. "references/FORMS.md").
func (s *SkillsToolset) ReadSkillFile(skillName, relativePath string) (string, error) {
	skill := s.findSkill(skillName)
	if skill == nil {
		return "", fmt.Errorf("skill %q not found", skillName)
	}

	if !isValidRelativePath(relativePath) {
		return "", fmt.Errorf("invalid file path %q", relativePath)
	}

	absPath := filepath.Join(skill.BaseDir, filepath.FromSlash(relativePath))

	// Ensure the resolved path stays within the skill's base directory
	cleanBase := filepath.Clean(skill.BaseDir)
	cleanPath := filepath.Clean(absPath)
	if !strings.HasPrefix(cleanPath, cleanBase+string(filepath.Separator)) && cleanPath != cleanBase {
		return "", fmt.Errorf("path %q escapes skill directory", relativePath)
	}

	content, err := readFileContent(absPath)
	if err != nil {
		return "", err
	}

	return content, nil
}

func readFileContent(path string) (string, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return "", fmt.Errorf("reading file: %w", err)
	}
	return string(data), nil
}

func isValidRelativePath(p string) bool {
	if p == "" || strings.HasPrefix(p, "/") || strings.HasPrefix(p, "\\") {
		return false
	}
	if strings.Contains(p, "..") {
		return false
	}
	return true
}

type readSkillArgs struct {
	Name string `json:"name" jsonschema:"The name of the skill to read"`
}

type readSkillFileArgs struct {
	SkillName string `json:"skill_name" jsonschema:"The name of the skill that contains the file"`
	Path      string `json:"path" jsonschema:"The relative path to the file within the skill (e.g. references/FORMS.md)"`
}

func (s *SkillsToolset) handleReadSkill(_ context.Context, args readSkillArgs) (*tools.ToolCallResult, error) {
	content, err := s.ReadSkillContent(args.Name)
	if err != nil {
		return tools.ResultError(err.Error()), nil
	}
	return tools.ResultSuccess(content), nil
}

func (s *SkillsToolset) handleReadSkillFile(_ context.Context, args readSkillFileArgs) (*tools.ToolCallResult, error) {
	content, err := s.ReadSkillFile(args.SkillName, args.Path)
	if err != nil {
		return tools.ResultError(err.Error()), nil
	}
	return tools.ResultSuccess(content), nil
}

func (s *SkillsToolset) Instructions() string {
	if len(s.skills) == 0 {
		return ""
	}

	hasFiles := false
	for _, skill := range s.skills {
		if len(skill.Files) > 1 {
			hasFiles = true
			break
		}
	}

	var sb strings.Builder
	sb.WriteString("The following skills provide specialized instructions for specific tasks. ")
	sb.WriteString("Each skill's description indicates what it does and when to use it.\n\n")
	sb.WriteString("When a user's request matches a skill's description, use the read_skill tool to load the skill's content. ")
	sb.WriteString("The content contains detailed instructions to follow for that task.\n\n")

	if hasFiles {
		sb.WriteString("Some skills reference supporting files (scripts, documentation, templates). ")
		sb.WriteString("When skill instructions reference a file path, use the read_skill_file tool to load it on demand. ")
		sb.WriteString("Do not load all files upfront — only load them as needed.\n\n")
	}

	sb.WriteString("<available_skills>\n")
	for _, skill := range s.skills {
		sb.WriteString("  <skill>\n")
		sb.WriteString("    <name>")
		sb.WriteString(skill.Name)
		sb.WriteString("</name>\n")
		sb.WriteString("    <description>")
		sb.WriteString(skill.Description)
		sb.WriteString("</description>\n")
		if len(skill.Files) > 1 {
			sb.WriteString("    <files>")
			// List files excluding SKILL.md itself
			first := true
			for _, f := range skill.Files {
				if f == "SKILL.md" {
					continue
				}
				if !first {
					sb.WriteString(", ")
				}
				sb.WriteString(f)
				first = false
			}
			sb.WriteString("</files>\n")
		}
		sb.WriteString("  </skill>\n")
	}
	sb.WriteString("</available_skills>")

	return sb.String()
}

func (s *SkillsToolset) Tools(context.Context) ([]tools.Tool, error) {
	if len(s.skills) == 0 {
		return nil, nil
	}

	result := []tools.Tool{
		{
			Name:         ToolNameReadSkill,
			Category:     "skills",
			Description:  "Read the content of a skill by name. Use this when a user's request matches an available skill.",
			Parameters:   tools.MustSchemaFor[readSkillArgs](),
			OutputSchema: tools.MustSchemaFor[string](),
			Handler:      tools.NewHandler(s.handleReadSkill),
			Annotations: tools.ToolAnnotations{
				Title:        "Read Skill",
				ReadOnlyHint: true,
			},
		},
	}

	// Only expose read_skill_file if any skill has supporting files
	hasFiles := false
	for _, skill := range s.skills {
		if len(skill.Files) > 1 {
			hasFiles = true
			break
		}
	}
	if hasFiles {
		result = append(result, tools.Tool{
			Name:         ToolNameReadSkillFile,
			Category:     "skills",
			Description:  "Read a supporting file from a skill (e.g. references, scripts, assets). Use when skill instructions reference additional files.",
			Parameters:   tools.MustSchemaFor[readSkillFileArgs](),
			OutputSchema: tools.MustSchemaFor[string](),
			Handler:      tools.NewHandler(s.handleReadSkillFile),
			Annotations: tools.ToolAnnotations{
				Title:        "Read Skill File",
				ReadOnlyHint: true,
			},
		})
	}

	return result, nil
}
