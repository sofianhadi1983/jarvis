package prompt

import (
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"strings"
)

type Loader struct {
	promptsDir string
}

func NewLoader(dir string) *Loader {
	if dir == "" {
		dir = "./prompts"
	}
	return &Loader{
		promptsDir: dir,
	}
}

func (l *Loader) LoadPrompt(files []string) (string, error) {
	if len(files) == 0 {
		return "", fmt.Errorf("no prompt files specified")
	}

	var prompts []string

	for _, file := range files {
		content, err := l.readFile(file)
		if err != nil {
			return "", fmt.Errorf("failed to load prompt file %s: %w", file, err)
		}
		prompts = append(prompts, strings.TrimSpace(content))
	}

	return strings.Join(prompts, "\n\n"), nil
}

func (l *Loader) LoadSinglePrompt(file string) (string, error) {
	if file == "" {
		return "", fmt.Errorf("no prompt file specified")
	}

	content, err := l.readFile(file)
	if err != nil {
		return "", fmt.Errorf("failed to load prompt file %s: %w", file, err)
	}

	return strings.TrimSpace(content), nil
}

func (l *Loader) readFile(filename string) (string, error) {
	visited := make(map[string]bool)
	return l.readFileRecursive(filename, visited)
}

func (l *Loader) readFileRecursive(filename string, visited map[string]bool) (string, error) {
	var filePath string

	if filepath.IsAbs(filename) {
		filePath = filename
	} else {
		filePath = filepath.Join(l.promptsDir, filename)
	}

	absPath, err := filepath.Abs(filePath)
	if err != nil {
		return "", err
	}
	if visited[absPath] {
		return "", fmt.Errorf("circular include detected: %s", filename)
	}
	visited[absPath] = true

	data, err := os.ReadFile(filePath)
	if err != nil {
		return "", err
	}

	content := string(data)

	content, err = l.processIncludes(content, visited)
	if err != nil {
		return "", fmt.Errorf("error processing includes in %s: %w", filename, err)
	}

	return content, nil
}

func (l *Loader) processIncludes(content string, visited map[string]bool) (string, error) {
	includeRegex := regexp.MustCompile(`\{\{\s*include\s+["']([^"']+)["']\s*\}\}`)

	matches := includeRegex.FindAllStringSubmatch(content, -1)
	for _, match := range matches {
		fullMatch := match[0]
		includeFile := match[1]

		includedContent, err := l.readFileRecursive(includeFile, visited)
		if err != nil {
			return "", fmt.Errorf("failed to include %s: %w", includeFile, err)
		}

		content = strings.Replace(content, fullMatch, includedContent, 1)
	}

	return content, nil
}
