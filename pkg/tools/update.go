package tools

import (
	"encoding/json"
	"fmt"
	"os"
	"path"
	"strings"
)

type UpdateFileInput struct {
	Path   string `json:"path" jsonschema_description:"The path to the file."`
	OldStr string `json:"old_str" jsonschema_description:"Text to search for - must match exactly and must only have one match"`
	NewStr string `json:"new_str" jsonschema_description:"Text to replace old_str with"`
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

	return fmt.Sprintf("Successfully created file %s", filePath), nil
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

	newContent := strings.Replace(oldContent, updateFileInput.OldStr, updateFileInput.NewStr, 1)

	err = os.WriteFile(updateFileInput.Path, []byte(newContent), 0644)
	if err != nil {
		return "", err
	}

	return "OK", nil
}

var UpdateFileDefinition = ToolDefinition{
	Name: "Update",
	Description: `Make edits to a text file.
Replace 'old_str' with 'new_str' in the given file. 'old_str' and 'new_str' MUST be different from each other.
If the file specified with path doesn't exist, it will be created.`,
	InputSchema: GenerateSchema[UpdateFileInput](),
	Function:    UpdateFile,
}
