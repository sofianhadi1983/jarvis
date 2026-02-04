package files

import (
	"bufio"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"sync"
	"time"
)

type FileEntry struct {
	Path    string
	Name    string
	IsDir   bool
	Size    int64
	ModTime time.Time
}

type Scanner struct {
	mu         sync.RWMutex
	cache      []FileEntry
	cacheTime  time.Time
	cacheTTL   time.Duration
	workingDir string
}

func NewScanner() *Scanner {
	wd, _ := os.Getwd()
	return &Scanner{
		cacheTTL:   5 * time.Second,
		workingDir: wd,
	}
}

var ignoredDirs = map[string]bool{
	".git":         true,
	"node_modules": true,
	"__pycache__":  true,
	".venv":        true,
	"venv":         true,
	".idea":        true,
	".vscode":      true,
	"vendor":       true,
	"dist":         true,
	"build":        true,
	".next":        true,
	"target":       true,
}

var ignoredFiles = map[string]bool{
	".DS_Store": true,
	"Thumbs.db": true,
}

var imageExtensions = map[string]bool{
	".jpg":  true,
	".jpeg": true,
	".png":  true,
	".gif":  true,
	".webp": true,
	".svg":  true,
	".ico":  true,
	".bmp":  true,
}

var binaryExtensions = map[string]bool{
	".exe": true, ".dll": true, ".so": true, ".dylib": true,
	".zip": true, ".tar": true, ".gz": true, ".rar": true, ".7z": true,
	".pdf": true, ".doc": true, ".docx": true, ".xls": true, ".xlsx": true,
	".mp3": true, ".mp4": true, ".avi": true, ".mov": true, ".mkv": true,
	".ttf": true, ".otf": true, ".woff": true, ".woff2": true,
	".pyc": true, ".class": true, ".o": true, ".a": true,
}

func (s *Scanner) Scan() ([]FileEntry, error) {
	s.mu.RLock()
	if time.Since(s.cacheTime) < s.cacheTTL && s.cache != nil {
		entries := make([]FileEntry, len(s.cache))
		copy(entries, s.cache)
		s.mu.RUnlock()
		return entries, nil
	}
	s.mu.RUnlock()

	s.mu.Lock()
	defer s.mu.Unlock()

	if time.Since(s.cacheTime) < s.cacheTTL && s.cache != nil {
		entries := make([]FileEntry, len(s.cache))
		copy(entries, s.cache)
		return entries, nil
	}

	var entries []FileEntry
	maxDepth := 4

	err := filepath.WalkDir(s.workingDir, func(path string, d os.DirEntry, err error) error {
		if err != nil {
			return nil
		}

		relPath, err := filepath.Rel(s.workingDir, path)
		if err != nil {
			return nil
		}

		if relPath == "." {
			return nil
		}

		depth := strings.Count(relPath, string(filepath.Separator))
		if depth > maxDepth {
			if d.IsDir() {
				return filepath.SkipDir
			}
			return nil
		}

		name := d.Name()

		if d.IsDir() {
			if ignoredDirs[name] || strings.HasPrefix(name, ".") {
				return filepath.SkipDir
			}
		}

		if !d.IsDir() {
			if ignoredFiles[name] {
				return nil
			}
			if strings.HasPrefix(name, ".") && name != ".env" && name != ".gitignore" {
				return nil
			}
		}

		info, err := d.Info()
		if err != nil {
			return nil
		}

		entries = append(entries, FileEntry{
			Path:    relPath,
			Name:    name,
			IsDir:   d.IsDir(),
			Size:    info.Size(),
			ModTime: info.ModTime(),
		})

		return nil
	})

	if err != nil {
		return nil, err
	}

	sort.Slice(entries, func(i, j int) bool {
		if entries[i].IsDir != entries[j].IsDir {
			return entries[i].IsDir
		}
		return strings.ToLower(entries[i].Path) < strings.ToLower(entries[j].Path)
	})

	s.cache = entries
	s.cacheTime = time.Now()

	result := make([]FileEntry, len(entries))
	copy(result, entries)
	return result, nil
}

func (s *Scanner) Filter(entries []FileEntry, query string) []FileEntry {
	if query == "" {
		return entries
	}

	query = strings.ToLower(query)
	var filtered []FileEntry

	isPathQuery := strings.Contains(query, "/") || strings.Contains(query, string(filepath.Separator))

	for _, entry := range entries {
		lowerPath := strings.ToLower(entry.Path)

		if isPathQuery {
			if strings.HasPrefix(lowerPath, query) || strings.HasPrefix(lowerPath+"/", query) {
				filtered = append(filtered, entry)
			}
		} else {
			if strings.Contains(lowerPath, query) || strings.Contains(strings.ToLower(entry.Name), query) {
				filtered = append(filtered, entry)
			}
		}
	}

	return filtered
}

func (s *Scanner) InvalidateCache() {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.cache = nil
	s.cacheTime = time.Time{}
}

func IsImageFile(path string) bool {
	ext := strings.ToLower(filepath.Ext(path))
	return imageExtensions[ext]
}

func IsBinaryFile(path string) bool {
	ext := strings.ToLower(filepath.Ext(path))
	return binaryExtensions[ext]
}

func IsTextFile(path string) bool {
	if IsBinaryFile(path) {
		return false
	}
	if IsImageFile(path) {
		return false
	}

	f, err := os.Open(path)
	if err != nil {
		return false
	}
	defer f.Close()

	buf := make([]byte, 512)
	n, err := f.Read(buf)
	if err != nil && n == 0 {
		return false
	}

	for i := 0; i < n; i++ {
		if buf[i] == 0 {
			return false
		}
	}

	return true
}

func ReadFileContent(path string, maxSize int64) (string, error) {
	info, err := os.Stat(path)
	if err != nil {
		return "", err
	}

	if info.Size() > maxSize {
		f, err := os.Open(path)
		if err != nil {
			return "", err
		}
		defer f.Close()

		reader := bufio.NewReader(f)
		buf := make([]byte, maxSize)
		n, _ := reader.Read(buf)
		return string(buf[:n]) + "\n... (truncated)", nil
	}

	data, err := os.ReadFile(path)
	if err != nil {
		return "", err
	}

	return string(data), nil
}

func GetFileIcon(entry FileEntry) string {
	if entry.IsDir {
		return "[D]"
	}
	if IsImageFile(entry.Name) {
		return "[I]"
	}
	return "[F]"
}
