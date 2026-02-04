package references

import (
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"strings"

	"jarvis/internal/files"
	"jarvis/internal/image"
)

type FileType int

const (
	FileTypeImage FileType = iota
	FileTypeText
	FileTypeBinary
)

const MaxTextFileSize = 100 * 1024

type FileReference struct {
	Path      string
	AbsPath   string
	Type      FileType
	Content   string
	ImageData *image.ImageInput
	Error     string
}

type ParsedInput struct {
	Text   string
	Files  []*FileReference
	Errors []string
}

var fileRefPattern = regexp.MustCompile(`@((?:https?://[^\s]+)|(?:[^\s@]+))`)

func ParseInput(input string) *ParsedInput {
	result := &ParsedInput{
		Text:   input,
		Files:  []*FileReference{},
		Errors: []string{},
	}

	matches := fileRefPattern.FindAllStringSubmatch(input, -1)
	if len(matches) == 0 {
		return result
	}

	processed := make(map[string]bool)

	for _, match := range matches {
		if len(match) < 2 {
			continue
		}

		fullMatch := match[0]
		ref := match[1]

		if processed[ref] {
			continue
		}
		processed[ref] = true

		result.Text = strings.Replace(result.Text, fullMatch, "", 1)

		fileRef := LoadFileReference(ref)
		if fileRef.Error != "" {
			result.Errors = append(result.Errors, fileRef.Error)
		}
		result.Files = append(result.Files, fileRef)
	}

	result.Text = strings.TrimSpace(result.Text)
	result.Text = regexp.MustCompile(`\s+`).ReplaceAllString(result.Text, " ")

	return result
}

func LoadFileReference(path string) *FileReference {
	ref := &FileReference{
		Path: path,
	}

	if image.IsURL(path) {
		ref.Type = FileTypeImage
		ref.ImageData = image.NewURLImage(path)
		ref.AbsPath = path
		return ref
	}

	absPath, err := resolvePath(path)
	if err != nil {
		ref.Error = fmt.Sprintf("cannot resolve path %s: %v", path, err)
		return ref
	}
	ref.AbsPath = absPath

	info, err := os.Stat(absPath)
	if err != nil {
		if os.IsNotExist(err) {
			ref.Error = fmt.Sprintf("file not found: %s", path)
		} else {
			ref.Error = fmt.Sprintf("cannot access %s: %v", path, err)
		}
		return ref
	}

	if info.IsDir() {
		ref.Error = fmt.Sprintf("cannot reference directory: %s", path)
		return ref
	}

	if files.IsImageFile(absPath) {
		ref.Type = FileTypeImage
		imgInput, err := image.LoadImage(absPath)
		if err != nil {
			ref.Error = fmt.Sprintf("failed to load image %s: %v", path, err)
			return ref
		}
		ref.ImageData = imgInput
	} else if files.IsBinaryFile(absPath) {
		ref.Type = FileTypeBinary
		ref.Error = fmt.Sprintf("binary file not supported: %s", path)
	} else if files.IsTextFile(absPath) {
		ref.Type = FileTypeText
		content, err := files.ReadFileContent(absPath, MaxTextFileSize)
		if err != nil {
			ref.Error = fmt.Sprintf("failed to read %s: %v", path, err)
			return ref
		}
		ref.Content = content
	} else {
		ref.Type = FileTypeText
		content, err := files.ReadFileContent(absPath, MaxTextFileSize)
		if err != nil {
			ref.Error = fmt.Sprintf("cannot read %s: %v", path, err)
			return ref
		}
		ref.Content = content
	}

	return ref
}

func resolvePath(path string) (string, error) {
	if strings.HasPrefix(path, "~/") {
		home, err := os.UserHomeDir()
		if err != nil {
			return "", err
		}
		path = filepath.Join(home, path[2:])
	}

	if !filepath.IsAbs(path) {
		wd, err := os.Getwd()
		if err != nil {
			return "", err
		}
		path = filepath.Join(wd, path)
	}

	return filepath.Clean(path), nil
}

func (p *ParsedInput) HasFiles() bool {
	return len(p.Files) > 0
}

func (p *ParsedInput) HasErrors() bool {
	return len(p.Errors) > 0
}

func (p *ParsedInput) GetImages() []*image.ImageInput {
	var images []*image.ImageInput
	for _, f := range p.Files {
		if f.Type == FileTypeImage && f.ImageData != nil && f.Error == "" {
			images = append(images, f.ImageData)
		}
	}
	return images
}

func (p *ParsedInput) GetTextContents() []string {
	var contents []string
	for _, f := range p.Files {
		if f.Type == FileTypeText && f.Content != "" && f.Error == "" {
			formatted := fmt.Sprintf("<file path=\"%s\">\n%s\n</file>", f.Path, f.Content)
			contents = append(contents, formatted)
		}
	}
	return contents
}

func (p *ParsedInput) FileIndicators() []string {
	var indicators []string
	for _, f := range p.Files {
		if f.Error != "" {
			continue
		}
		switch f.Type {
		case FileTypeImage:
			indicators = append(indicators, fmt.Sprintf("[image: %s]", f.Path))
		case FileTypeText:
			indicators = append(indicators, fmt.Sprintf("[file: %s]", f.Path))
		}
	}
	return indicators
}

func (p *ParsedInput) CombineTextContent() string {
	textContents := p.GetTextContents()
	if len(textContents) == 0 {
		return p.Text
	}

	if p.Text == "" {
		return strings.Join(textContents, "\n\n")
	}

	return p.Text + "\n\n" + strings.Join(textContents, "\n\n")
}
