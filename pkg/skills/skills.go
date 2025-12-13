package skills

import (
	"os"
	"path/filepath"
	"regexp"
	"strings"

	"github.com/docker/cagent/pkg/paths"
)

type skillFormat string

const (
	formatClaude skillFormat = "claude"
	formatCodex  skillFormat = "codex"

	skillFile = "SKILL.md"
)

type Skill struct {
	Name        string
	Description string
	FilePath    string
	BaseDir     string
}

type frontmatter struct {
	Name        string
	Description string
}

func stripQuotes(value string) string {
	if len(value) >= 2 {
		if (value[0] == '"' && value[len(value)-1] == '"') ||
			(value[0] == '\'' && value[len(value)-1] == '\'') {
			return value[1 : len(value)-1]
		}
	}
	return value
}

func parseFrontmatter(content string) (frontmatter, string) {
	fm := frontmatter{}

	normalizedContent := strings.ReplaceAll(content, "\r\n", "\n")
	normalizedContent = strings.ReplaceAll(normalizedContent, "\r", "\n")

	if !strings.HasPrefix(normalizedContent, "---") {
		return fm, normalizedContent
	}

	endIndex := strings.Index(normalizedContent[3:], "\n---")
	if endIndex == -1 {
		return fm, normalizedContent
	}

	frontmatterBlock := normalizedContent[4 : endIndex+3]
	body := strings.TrimSpace(normalizedContent[endIndex+7:])

	lineRegex := regexp.MustCompile(`^(\w+):\s*(.*)$`)
	for line := range strings.SplitSeq(frontmatterBlock, "\n") {
		matches := lineRegex.FindStringSubmatch(line)
		if matches != nil {
			key := matches[1]
			value := stripQuotes(strings.TrimSpace(matches[2]))
			switch key {
			case "name":
				fm.Name = value
			case "description":
				fm.Description = value
			}
		}
	}

	return fm, body
}

func loadSkillsFromDir(dir string, format skillFormat) []Skill {
	var skills []Skill

	info, err := os.Stat(dir)
	if err != nil || !info.IsDir() {
		return skills
	}

	entries, err := os.ReadDir(dir)
	if err != nil {
		return skills
	}

	for _, entry := range entries {
		if strings.HasPrefix(entry.Name(), ".") {
			continue
		}

		if entry.Type()&os.ModeSymlink != 0 {
			continue
		}

		fullPath := filepath.Join(dir, entry.Name())

		switch format {
		case formatClaude:
			if !entry.IsDir() {
				continue
			}

			skillFile := filepath.Join(fullPath, skillFile)
			rawContent, err := os.ReadFile(skillFile)
			if err != nil {
				continue
			}

			fm, _ := parseFrontmatter(string(rawContent))
			if fm.Description == "" {
				continue
			}

			name := fm.Name
			if name == "" {
				name = entry.Name()
			}

			skills = append(skills, Skill{
				Name:        name,
				Description: fm.Description,
				FilePath:    skillFile,
				BaseDir:     fullPath,
			})

		case formatCodex:
			if entry.IsDir() {
				skills = append(skills, loadSkillsFromDir(fullPath, format)...)
			} else if entry.Name() == skillFile {
				rawContent, err := os.ReadFile(fullPath)
				if err != nil {
					continue
				}

				fm, _ := parseFrontmatter(string(rawContent))
				if fm.Description == "" {
					continue
				}

				skillDir := filepath.Dir(fullPath)
				name := fm.Name
				if name == "" {
					name = filepath.Base(skillDir)
				}

				skills = append(skills, Skill{
					Name:        name,
					Description: fm.Description,
					FilePath:    fullPath,
					BaseDir:     skillDir,
				})
			}
		}
	}

	return skills
}

func Load() []Skill {
	skillMap := make(map[string]Skill)

	homeDir := paths.GetHomeDir()
	if homeDir == "" {
		return nil
	}

	codexUserDir := filepath.Join(homeDir, ".codex", "skills")
	for _, skill := range loadSkillsFromDir(codexUserDir, formatCodex) {
		skillMap[skill.Name] = skill
	}

	claudeUserDir := filepath.Join(homeDir, ".claude", "skills")
	for _, skill := range loadSkillsFromDir(claudeUserDir, formatClaude) {
		skillMap[skill.Name] = skill
	}

	cwd, err := os.Getwd()
	if err == nil {
		claudeProjectDir := filepath.Join(cwd, ".claude", "skills")
		for _, skill := range loadSkillsFromDir(claudeProjectDir, formatClaude) {
			skillMap[skill.Name] = skill
		}
	}

	result := make([]Skill, 0, len(skillMap))
	for _, skill := range skillMap {
		result = append(result, skill)
	}

	return result
}

func BuildSkillsPrompt(skills []Skill) string {
	if len(skills) == 0 {
		return ""
	}

	var sb strings.Builder
	sb.WriteString("\n\n<available_skills>\n")
	sb.WriteString("The following skills provide specialized instructions for specific tasks.\n")
	sb.WriteString("Use the read_file tool to load a skill's file when the task matches its description.\n")
	sb.WriteString("Skills may contain {baseDir} placeholders - replace them with the skill's base directory path.\n\n")

	for _, skill := range skills {
		sb.WriteString("- ")
		sb.WriteString(skill.Name)
		sb.WriteString(": ")
		sb.WriteString(skill.Description)
		sb.WriteString("\n  File: ")
		sb.WriteString(skill.FilePath)
		sb.WriteString("\n  Base directory: ")
		sb.WriteString(skill.BaseDir)
		sb.WriteString("\n")
	}

	sb.WriteString("</available_skills>")
	return sb.String()
}
