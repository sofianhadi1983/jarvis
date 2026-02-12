package skills

import (
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"strings"
)

type Skill struct {
	Name          string
	Description   string
	UserInvocable bool
	Body          string // Markdown instructions
	Dir           string // directory path for resources
}

type SkillLoader struct {
	skills    map[string]*Skill
	skillsDir string
}

func NewSkillLoader(dir string) *SkillLoader {
	sl := &SkillLoader{
		skills:    make(map[string]*Skill),
		skillsDir: dir,
	}
	sl.scan()
	return sl
}

func (sl *SkillLoader) scan() {
	entries, err := os.ReadDir(sl.skillsDir)
	if err != nil {
		return
	}

	for _, entry := range entries {
		if !entry.IsDir() {
			continue
		}
		skillPath := filepath.Join(sl.skillsDir, entry.Name(), "SKILL.md")
		if _, err := os.Stat(skillPath); err != nil {
			continue
		}
		skill, err := sl.parseSkillMD(skillPath, filepath.Join(sl.skillsDir, entry.Name()))
		if err != nil {
			continue
		}
		sl.skills[skill.Name] = skill
	}
}

var frontmatterRe = regexp.MustCompile(`(?s)^---\n(.*?)\n---\n(.*)$`)

func (sl *SkillLoader) parseSkillMD(path string, dir string) (*Skill, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("failed to read skill file: %w", err)
	}

	content := string(data)
	matches := frontmatterRe.FindStringSubmatch(content)
	if matches == nil {
		return nil, fmt.Errorf("invalid SKILL.md format: missing frontmatter in %s", path)
	}

	frontmatter := matches[1]
	body := strings.TrimSpace(matches[2])

	skill := &Skill{
		Dir:  dir,
		Body: body,
	}

	// Parse frontmatter as simple key: value lines
	for _, line := range strings.Split(frontmatter, "\n") {
		line = strings.TrimSpace(line)
		if line == "" {
			continue
		}
		parts := strings.SplitN(line, ":", 2)
		if len(parts) != 2 {
			continue
		}
		key := strings.TrimSpace(parts[0])
		value := strings.TrimSpace(parts[1])

		switch key {
		case "name":
			skill.Name = value
		case "description":
			skill.Description = value
		case "user_invocable":
			skill.UserInvocable = value == "true"
		}
	}

	if skill.Name == "" {
		return nil, fmt.Errorf("skill missing required 'name' field in %s", path)
	}
	if skill.Description == "" {
		return nil, fmt.Errorf("skill missing required 'description' field in %s", path)
	}

	// Scan for resource subdirectories and append hints
	resourceDirs := []string{"scripts", "references", "assets"}
	var hints []string
	for _, rd := range resourceDirs {
		rdPath := filepath.Join(dir, rd)
		if info, err := os.Stat(rdPath); err == nil && info.IsDir() {
			entries, err := os.ReadDir(rdPath)
			if err == nil && len(entries) > 0 {
				var files []string
				for _, e := range entries {
					files = append(files, e.Name())
				}
				hints = append(hints, fmt.Sprintf("Available %s: %s", rd, strings.Join(files, ", ")))
			}
		}
	}
	if len(hints) > 0 {
		skill.Body += "\n\n## Resources\n" + strings.Join(hints, "\n")
	}

	return skill, nil
}

// GetDescriptions returns a compact summary of all skills for system prompt injection.
func (sl *SkillLoader) GetDescriptions() string {
	var lines []string
	for _, s := range sl.skills {
		lines = append(lines, fmt.Sprintf("- %s: %s", s.Name, s.Description))
	}
	return strings.Join(lines, "\n")
}

// GetSkillContent returns the full body of a skill by name.
func (sl *SkillLoader) GetSkillContent(name string) (string, error) {
	skill, ok := sl.skills[name]
	if !ok {
		return "", fmt.Errorf("skill not found: %s", name)
	}
	return skill.Body, nil
}

// GetUserInvocableSkills returns names of skills with user_invocable: true.
func (sl *SkillLoader) GetUserInvocableSkills() []string {
	var names []string
	for _, s := range sl.skills {
		if s.UserInvocable {
			names = append(names, s.Name)
		}
	}
	return names
}

// List returns all skill names.
func (sl *SkillLoader) List() []string {
	var names []string
	for name := range sl.skills {
		names = append(names, name)
	}
	return names
}

// Has checks if a skill exists by name.
func (sl *SkillLoader) Has(name string) bool {
	_, ok := sl.skills[name]
	return ok
}
