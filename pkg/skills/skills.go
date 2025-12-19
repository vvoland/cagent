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

	maxNameLength        = 64
	maxDescriptionLength = 1024
	maxCompatLength      = 500
)

var namePattern = regexp.MustCompile(`^[a-z0-9]+(-[a-z0-9]+)*$`)

type Skill struct {
	Name          string
	Description   string
	FilePath      string
	BaseDir       string
	License       string
	Compatibility string
	Metadata      map[string]string
	AllowedTools  []string
}

type frontmatter struct {
	Name          string
	Description   string
	License       string
	Compatibility string
	Metadata      map[string]string
	AllowedTools  []string
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

func isValidName(name string) bool {
	if name == "" || len(name) > maxNameLength {
		return false
	}
	return namePattern.MatchString(name)
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

	lineRegex := regexp.MustCompile(`^([\w-]+):\s*(.*)$`)
	metadataRegex := regexp.MustCompile(`^\s+(\w+):\s*(.*)$`)
	inMetadata := false

	for line := range strings.SplitSeq(frontmatterBlock, "\n") {
		if inMetadata {
			matches := metadataRegex.FindStringSubmatch(line)
			if matches != nil {
				key := matches[1]
				value := stripQuotes(strings.TrimSpace(matches[2]))
				if fm.Metadata == nil {
					fm.Metadata = make(map[string]string)
				}
				fm.Metadata[key] = value
				continue
			}
			inMetadata = false
		}

		matches := lineRegex.FindStringSubmatch(line)
		if matches != nil {
			key := matches[1]
			value := stripQuotes(strings.TrimSpace(matches[2]))
			switch key {
			case "name":
				fm.Name = value
			case "description":
				fm.Description = value
			case "license":
				fm.License = value
			case "compatibility":
				fm.Compatibility = value
			case "metadata":
				inMetadata = true
				fm.Metadata = make(map[string]string)
			case "allowed-tools":
				if value != "" {
					fm.AllowedTools = strings.Fields(value)
				}
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

			skillFilePath := filepath.Join(fullPath, skillFile)
			rawContent, err := os.ReadFile(skillFilePath)
			if err != nil {
				continue
			}

			fm, _ := parseFrontmatter(string(rawContent))
			if !isValidFrontmatter(fm, entry.Name()) {
				continue
			}

			name := fm.Name
			if name == "" {
				name = entry.Name()
			}

			skills = append(skills, Skill{
				Name:          name,
				Description:   fm.Description,
				FilePath:      skillFilePath,
				BaseDir:       fullPath,
				License:       fm.License,
				Compatibility: fm.Compatibility,
				Metadata:      fm.Metadata,
				AllowedTools:  fm.AllowedTools,
			})

		case formatCodex:
			if entry.IsDir() {
				skills = append(skills, loadSkillsFromDir(fullPath, format)...)
			} else if entry.Name() == skillFile {
				rawContent, err := os.ReadFile(fullPath)
				if err != nil {
					continue
				}

				skillDir := filepath.Dir(fullPath)
				dirName := filepath.Base(skillDir)

				fm, _ := parseFrontmatter(string(rawContent))
				if !isValidFrontmatter(fm, dirName) {
					continue
				}

				name := fm.Name
				if name == "" {
					name = dirName
				}

				skills = append(skills, Skill{
					Name:          name,
					Description:   fm.Description,
					FilePath:      fullPath,
					BaseDir:       skillDir,
					License:       fm.License,
					Compatibility: fm.Compatibility,
					Metadata:      fm.Metadata,
					AllowedTools:  fm.AllowedTools,
				})
			}
		}
	}

	return skills
}

func isValidFrontmatter(fm frontmatter, dirName string) bool {
	if fm.Description == "" || len(fm.Description) > maxDescriptionLength {
		return false
	}

	if fm.Compatibility != "" && len(fm.Compatibility) > maxCompatLength {
		return false
	}

	if fm.Name != "" {
		if !isValidName(fm.Name) {
			return false
		}
		if fm.Name != dirName {
			return false
		}
	}

	return true
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
	sb.WriteString("The following skills provide specialized instructions for specific tasks. ")
	sb.WriteString("Each skill's description indicates what it does and when to use it.\n\n")
	sb.WriteString("When a user's request matches a skill's description, use the read_file tool to load the skill's SKILL.md file from the location path. ")
	sb.WriteString("The file contains detailed instructions to follow for that task.\n\n")

	sb.WriteString("\n\n<available_skills>\n")
	for _, skill := range skills {
		sb.WriteString("  <skill>\n")
		sb.WriteString("    <name>")
		sb.WriteString(skill.Name)
		sb.WriteString("</name>\n")
		sb.WriteString("    <description>")
		sb.WriteString(skill.Description)
		sb.WriteString("</description>\n")
		sb.WriteString("    <location>")
		sb.WriteString(skill.FilePath)
		sb.WriteString("</location>\n")
		sb.WriteString("  </skill>\n")
	}

	sb.WriteString("</available_skills>")
	return sb.String()
}
