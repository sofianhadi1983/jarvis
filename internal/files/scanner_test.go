package files

import (
	"os"
	"path/filepath"
	"testing"
)

func TestNewScanner(t *testing.T) {
	scanner := NewScanner()
	if scanner == nil {
		t.Fatal("NewScanner() returned nil")
	}
}

func TestScanner_Scan(t *testing.T) {
	// Create temp directory with test files
	tmpDir := t.TempDir()

	// Create test files
	testFiles := []string{
		"file1.go",
		"file2.txt",
		"subdir/file3.js",
	}

	for _, f := range testFiles {
		path := filepath.Join(tmpDir, f)
		dir := filepath.Dir(path)
		if err := os.MkdirAll(dir, 0755); err != nil {
			t.Fatalf("Failed to create dir: %v", err)
		}
		if err := os.WriteFile(path, []byte("test content"), 0644); err != nil {
			t.Fatalf("Failed to create file: %v", err)
		}
	}

	// Change to temp dir for scanning
	oldWd, _ := os.Getwd()
	os.Chdir(tmpDir)
	defer os.Chdir(oldWd)

	scanner := NewScanner()
	entries, err := scanner.Scan()
	if err != nil {
		t.Fatalf("Scan() error: %v", err)
	}

	if len(entries) < 3 {
		t.Errorf("Expected at least 3 entries, got %d", len(entries))
	}
}

func TestScanner_Filter(t *testing.T) {
	entries := []FileEntry{
		{Path: "src/main.go", Name: "main.go"},
		{Path: "src/utils.go", Name: "utils.go"},
		{Path: "tests/main_test.go", Name: "main_test.go"},
		{Path: "README.md", Name: "README.md"},
	}

	scanner := NewScanner()

	tests := []struct {
		name     string
		query    string
		expected int
	}{
		{"empty query", "", 4},
		{"filter by path prefix", "src/", 2},
		{"filter by filename", "main", 2},
		{"filter by extension", ".go", 3},
		{"no match", "xyz", 0},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			filtered := scanner.Filter(entries, tt.query)
			if len(filtered) != tt.expected {
				t.Errorf("Filter(%q) returned %d entries, want %d", tt.query, len(filtered), tt.expected)
			}
		})
	}
}

func TestIsImageFile(t *testing.T) {
	tests := []struct {
		path     string
		expected bool
	}{
		{"photo.jpg", true},
		{"photo.jpeg", true},
		{"photo.png", true},
		{"photo.gif", true},
		{"photo.webp", true},
		{"photo.PNG", true},
		{"code.go", false},
		{"doc.pdf", false},
		{"", false},
	}

	for _, tt := range tests {
		t.Run(tt.path, func(t *testing.T) {
			result := IsImageFile(tt.path)
			if result != tt.expected {
				t.Errorf("IsImageFile(%q) = %v, want %v", tt.path, result, tt.expected)
			}
		})
	}
}

func TestIsBinaryFile(t *testing.T) {
	tests := []struct {
		path     string
		expected bool
	}{
		{"app.exe", true},
		{"lib.dll", true},
		{"lib.so", true},
		{"archive.zip", true},
		{"doc.pdf", true},
		{"code.go", false},
		{"style.css", false},
	}

	for _, tt := range tests {
		t.Run(tt.path, func(t *testing.T) {
			result := IsBinaryFile(tt.path)
			if result != tt.expected {
				t.Errorf("IsBinaryFile(%q) = %v, want %v", tt.path, result, tt.expected)
			}
		})
	}
}

func TestIsTextFile(t *testing.T) {
	tmpDir := t.TempDir()

	// Create a text file
	textPath := filepath.Join(tmpDir, "test.txt")
	if err := os.WriteFile(textPath, []byte("Hello, World!"), 0644); err != nil {
		t.Fatalf("Failed to create text file: %v", err)
	}

	// Create a binary file
	binaryPath := filepath.Join(tmpDir, "test.bin")
	if err := os.WriteFile(binaryPath, []byte{0x00, 0x01, 0x02, 0x03}, 0644); err != nil {
		t.Fatalf("Failed to create binary file: %v", err)
	}

	t.Run("text file", func(t *testing.T) {
		if !IsTextFile(textPath) {
			t.Error("Expected text file to return true")
		}
	})

	t.Run("binary file", func(t *testing.T) {
		if IsTextFile(binaryPath) {
			t.Error("Expected binary file to return false")
		}
	})

	t.Run("nonexistent file", func(t *testing.T) {
		if IsTextFile("/nonexistent/path") {
			t.Error("Expected nonexistent file to return false")
		}
	})
}

func TestReadFileContent(t *testing.T) {
	tmpDir := t.TempDir()

	t.Run("small file", func(t *testing.T) {
		path := filepath.Join(tmpDir, "small.txt")
		content := "Hello, World!"
		if err := os.WriteFile(path, []byte(content), 0644); err != nil {
			t.Fatalf("Failed to create file: %v", err)
		}

		result, err := ReadFileContent(path, 1024)
		if err != nil {
			t.Fatalf("ReadFileContent() error: %v", err)
		}
		if result != content {
			t.Errorf("ReadFileContent() = %q, want %q", result, content)
		}
	})

	t.Run("large file truncated", func(t *testing.T) {
		path := filepath.Join(tmpDir, "large.txt")
		// Create a file larger than maxSize
		largeContent := make([]byte, 200)
		for i := range largeContent {
			largeContent[i] = 'a'
		}
		if err := os.WriteFile(path, largeContent, 0644); err != nil {
			t.Fatalf("Failed to create file: %v", err)
		}

		result, err := ReadFileContent(path, 100)
		if err != nil {
			t.Fatalf("ReadFileContent() error: %v", err)
		}
		if len(result) > 120 { // 100 + truncation message
			t.Errorf("ReadFileContent() should truncate, got length %d", len(result))
		}
	})
}

func TestGetFileIcon(t *testing.T) {
	tests := []struct {
		entry    FileEntry
		expected string
	}{
		{FileEntry{Name: "folder", IsDir: true}, "[D]"},
		{FileEntry{Name: "photo.png"}, "[I]"},
		{FileEntry{Name: "main.go"}, "[F]"},
		{FileEntry{Name: "app.js"}, "[F]"},
		{FileEntry{Name: "script.py"}, "[F]"},
		{FileEntry{Name: "README.md"}, "[F]"},
		{FileEntry{Name: "unknown.xyz"}, "[F]"},
	}

	for _, tt := range tests {
		t.Run(tt.entry.Name, func(t *testing.T) {
			result := GetFileIcon(tt.entry)
			if result != tt.expected {
				t.Errorf("GetFileIcon(%q) = %q, want %q", tt.entry.Name, result, tt.expected)
			}
		})
	}
}
