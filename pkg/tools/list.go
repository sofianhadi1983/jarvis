package tools

import (
	"encoding/json"
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"sort"
	"strings"
)

const (
	treePipe       = "│   "
	treeMiddleItem = "├── "
	treeLastItem   = "└── "
	treeEmptySpace = "    "
)

type TreeStats struct {
	Dirs  int
	Files int
}

type TreeResult struct {
	Tree  string
	Stats TreeStats
}

func BuildTree(root string) (*TreeResult, error) {
	info, err := os.Stat(root)
	if err != nil {
		return nil, fmt.Errorf("cannot access %s: %w", root, err)
	}

	if !info.IsDir() {
		return nil, fmt.Errorf("%s is not a directory", root)
	}

	var sb strings.Builder
	stats := &TreeStats{}

	sb.WriteString(root)
	sb.WriteString("\n")

	buildTreeRecursive(root, "", stats, &sb)

	return &TreeResult{
		Tree:  sb.String(),
		Stats: *stats,
	}, nil
}

func buildTreeRecursive(path, prefix string, stats *TreeStats, sb *strings.Builder) {
	entries, err := os.ReadDir(path)
	if err != nil {
		sb.WriteString(fmt.Sprintf("%s[error: %v]\n", prefix, err))
		return
	}

	filtered := filterTreeEntries(entries)
	sortTreeEntries(filtered)

	for i, entry := range filtered {
		isLast := i == len(filtered)-1
		connector := treeMiddleItem
		if isLast {
			connector = treeLastItem
		}

		name := entry.Name()

		if entry.IsDir() {
			stats.Dirs++
			sb.WriteString(fmt.Sprintf("%s%s%s\n", prefix, connector, name))

			newPrefix := prefix + treeEmptySpace
			if !isLast {
				newPrefix = prefix + treePipe
			}
			buildTreeRecursive(filepath.Join(path, name), newPrefix, stats, sb)
		} else {
			stats.Files++
			size := getTreeFileSize(entry)
			sb.WriteString(fmt.Sprintf("%s%s%s %s\n", prefix, connector, name, size))
		}
	}
}

func filterTreeEntries(entries []fs.DirEntry) []fs.DirEntry {
	var filtered []fs.DirEntry
	for _, entry := range entries {
		if entry.Name()[0] != '.' {
			filtered = append(filtered, entry)
		}
	}
	return filtered
}

func sortTreeEntries(entries []fs.DirEntry) {
	sort.Slice(entries, func(i, j int) bool {
		iDir, jDir := entries[i].IsDir(), entries[j].IsDir()
		if iDir != jDir {
			return iDir
		}
		return entries[i].Name() < entries[j].Name()
	})
}

func getTreeFileSize(entry fs.DirEntry) string {
	info, err := entry.Info()
	if err != nil {
		return "[?]"
	}
	return formatTreeSize(info.Size())
}

func formatTreeSize(bytes int64) string {
	const (
		KB = 1024
		MB = KB * 1024
		GB = MB * 1024
	)

	switch {
	case bytes >= GB:
		return fmt.Sprintf("[%.1fG]", float64(bytes)/GB)
	case bytes >= MB:
		return fmt.Sprintf("[%.1fM]", float64(bytes)/MB)
	case bytes >= KB:
		return fmt.Sprintf("[%.1fK]", float64(bytes)/KB)
	default:
		return fmt.Sprintf("[%dB]", bytes)
	}
}

type ListFilesInput struct {
	Path       string `json:"path,omitempty" jsonschema_description:"Optional relative path to list files from. Defaults to current directory if not provided."`
	ShowHidden bool   `json:"show_hidden,omitempty" jsonschema_description:"Whether to show hidden files (starting with dot). Defaults to false."`
}

func ListFiles(input json.RawMessage) (string, error) {
	var listFilesInput ListFilesInput
	err := json.Unmarshal(input, &listFilesInput)
	if err != nil {
		return "", fmt.Errorf("failed to parse input: %w", err)
	}

	root := "."
	if listFilesInput.Path != "" {
		root = listFilesInput.Path
	}

	result, err := BuildTree(root)
	if err != nil {
		return "", err
	}

	return result.Tree, nil
}

var ListFilesDefinition = ToolDefinition{
	Name:        "ListFiles",
	Description: "List files and directories at a given path. If no path is provided, list files in the current directory.",
	InputSchema: GenerateSchema[ListFilesInput](),
	Function:    ListFiles,
}
