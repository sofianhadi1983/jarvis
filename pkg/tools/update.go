package tools

import (
	"encoding/json"
	"fmt"
	"os"
	"path"
	"strings"

	"github.com/sergi/go-diff/diffmatchpatch"
)

type UpdateFileInput struct {
	Path   string `json:"path" jsonschema_description:"The path to the file."`
	OldStr string `json:"old_str" jsonschema_description:"Text to search for - must match exactly and must only have one match"`
	NewStr string `json:"new_str" jsonschema_description:"Text to replace old_str with"`
}

// UpdateResult contains the result of an update operation with diff info
type UpdateResult struct {
	Success      bool   `json:"success"`
	Message      string `json:"message"`
	FilePath     string `json:"file_path"`
	UnifiedDiff  string `json:"unified_diff"` // Proper unified diff format
	AddedLines   int    `json:"added_lines"`
	RemovedLines int    `json:"removed_lines"`
	StartLine    int    `json:"start_line"`
	IsNewFile    bool   `json:"is_new_file"`
}

func createNewFile(filePath, content string) (string, error) {
	dir := path.Dir(filePath)
	if dir != "." {
		err := os.MkdirAll(dir, 0755)
		if err != nil {
			return "", fmt.Errorf("failed to create directory: %w", err)
		}
	}

	err := os.WriteFile(filePath, []byte(content), 0644)
	if err != nil {
		return "", fmt.Errorf("failed to create file: %w", err)
	}

	// Generate diff for new file (all additions)
	diff := generateUnifiedDiff("", content, filePath)

	result := UpdateResult{
		Success:      true,
		Message:      fmt.Sprintf("Successfully created file %s", filePath),
		FilePath:     filePath,
		UnifiedDiff:  diff,
		AddedLines:   countLines(content),
		RemovedLines: 0,
		StartLine:    1,
		IsNewFile:    true,
	}

	jsonResult, _ := json.Marshal(result)
	return string(jsonResult), nil
}

func countLines(s string) int {
	if s == "" {
		return 0
	}
	return strings.Count(s, "\n") + 1
}

// findLineNumber finds the line number where oldStr starts in the content
func findLineNumber(content, oldStr string) int {
	if oldStr == "" {
		return 1
	}
	idx := strings.Index(content, oldStr)
	if idx == -1 {
		return 1
	}
	return strings.Count(content[:idx], "\n") + 1
}

// generateUnifiedDiff creates a unified diff between old and new content
func generateUnifiedDiff(oldContent, newContent, filePath string) string {
	dmp := diffmatchpatch.New()

	// Generate line-based diff
	a, b, c := dmp.DiffLinesToChars(oldContent, newContent)
	diffs := dmp.DiffMain(a, b, false)
	diffs = dmp.DiffCharsToLines(diffs, c)

	// Build unified diff output
	var sb strings.Builder

	oldLines := strings.Split(oldContent, "\n")
	newLines := strings.Split(newContent, "\n")

	oldLineNum := 1
	newLineNum := 1

	for _, diff := range diffs {
		lines := strings.Split(strings.TrimSuffix(diff.Text, "\n"), "\n")

		switch diff.Type {
		case diffmatchpatch.DiffEqual:
			for range lines {
				oldLineNum++
				newLineNum++
			}
		case diffmatchpatch.DiffDelete:
			for _, line := range lines {
				sb.WriteString(fmt.Sprintf("%4d - %s\n", oldLineNum, line))
				oldLineNum++
			}
		case diffmatchpatch.DiffInsert:
			for _, line := range lines {
				sb.WriteString(fmt.Sprintf("%4d + %s\n", newLineNum, line))
				newLineNum++
			}
		}
	}

	_ = oldLines
	_ = newLines

	return sb.String()
}

func UpdateFile(input json.RawMessage) (string, error) {
	var updateFileInput UpdateFileInput
	err := json.Unmarshal(input, &updateFileInput)
	if err != nil {
		return "", fmt.Errorf("failed to parse input: %w", err)
	}

	if updateFileInput.Path == "" || updateFileInput.OldStr == updateFileInput.NewStr {
		return "", fmt.Errorf("invalid input parameters")
	}

	content, err := os.ReadFile(updateFileInput.Path)
	if err != nil {
		if os.IsNotExist(err) && updateFileInput.OldStr == "" {
			return createNewFile(updateFileInput.Path, updateFileInput.NewStr)
		}
		return "", err
	}

	oldContent := string(content)

	count := strings.Count(oldContent, updateFileInput.OldStr)
	if count == 0 && updateFileInput.OldStr != "" {
		return "", fmt.Errorf("old_str not found in the file")
	}
	if count > 1 {
		return "", fmt.Errorf("old_str found %d times, must be unique", count)
	}

	// Find the starting line number
	startLine := findLineNumber(oldContent, updateFileInput.OldStr)

	newContent := strings.Replace(oldContent, updateFileInput.OldStr, updateFileInput.NewStr, 1)

	err = os.WriteFile(updateFileInput.Path, []byte(newContent), 0644)
	if err != nil {
		return "", err
	}

	// Generate unified diff
	diff := generateUnifiedDiff(updateFileInput.OldStr, updateFileInput.NewStr, updateFileInput.Path)

	// Calculate line changes
	oldLines := countLines(updateFileInput.OldStr)
	newLines := countLines(updateFileInput.NewStr)

	result := UpdateResult{
		Success:      true,
		Message:      "OK",
		FilePath:     updateFileInput.Path,
		UnifiedDiff:  diff,
		AddedLines:   newLines,
		RemovedLines: oldLines,
		StartLine:    startLine,
		IsNewFile:    false,
	}

	jsonResult, _ := json.Marshal(result)
	return string(jsonResult), nil
}

var UpdateFileDefinition = ToolDefinition{
	Name: "Update",
	Description: `Make edits to a text file.
Replace 'old_str' with 'new_str' in the given file. 'old_str' and 'new_str' MUST be different from each other.
If the file specified with path doesn't exist, it will be created.`,
	InputSchema: GenerateSchema[UpdateFileInput](),
	Function:    UpdateFile,
}
