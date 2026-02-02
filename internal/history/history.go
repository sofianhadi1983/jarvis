package history

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"time"

	"github.com/google/uuid"
)

// Entry represents a single history entry
type Entry struct {
	Prompt    string    `json:"prompt"`
	Timestamp time.Time `json:"timestamp"`
}

// Session represents a chat session with its history
type Session struct {
	ID        string    `json:"id"`
	CreatedAt time.Time `json:"created_at"`
	UpdatedAt time.Time `json:"updated_at"`
	Entries   []Entry   `json:"entries"`
}

// Manager handles history storage and retrieval
type Manager struct {
	session     *Session
	historyDir  string
	historyFile string
	index       int // Current position in history for navigation
}

// NewManager creates a new history manager
func NewManager() (*Manager, error) {
	homeDir, err := os.UserHomeDir()
	if err != nil {
		return nil, fmt.Errorf("failed to get home directory: %w", err)
	}

	historyDir := filepath.Join(homeDir, ".jarvis", "history")
	if err := os.MkdirAll(historyDir, 0755); err != nil {
		return nil, fmt.Errorf("failed to create history directory: %w", err)
	}

	// Create new session
	sessionID := uuid.New().String()[:8]
	session := &Session{
		ID:        sessionID,
		CreatedAt: time.Now(),
		UpdatedAt: time.Now(),
		Entries:   []Entry{},
	}

	historyFile := filepath.Join(historyDir, fmt.Sprintf("session_%s.json", sessionID))

	manager := &Manager{
		session:     session,
		historyDir:  historyDir,
		historyFile: historyFile,
		index:       -1,
	}

	return manager, nil
}

// LoadSession loads an existing session by ID
func (m *Manager) LoadSession(sessionID string) error {
	historyFile := filepath.Join(m.historyDir, fmt.Sprintf("session_%s.json", sessionID))

	data, err := os.ReadFile(historyFile)
	if err != nil {
		return fmt.Errorf("failed to read session file: %w", err)
	}

	var session Session
	if err := json.Unmarshal(data, &session); err != nil {
		return fmt.Errorf("failed to parse session file: %w", err)
	}

	m.session = &session
	m.historyFile = historyFile
	m.index = len(session.Entries)

	return nil
}

// Add adds a new entry to history
func (m *Manager) Add(prompt string) {
	entry := Entry{
		Prompt:    prompt,
		Timestamp: time.Now(),
	}
	m.session.Entries = append(m.session.Entries, entry)
	m.session.UpdatedAt = time.Now()
	m.index = len(m.session.Entries)
	m.Save()
}

// Save persists the session to disk
func (m *Manager) Save() error {
	data, err := json.MarshalIndent(m.session, "", "  ")
	if err != nil {
		return fmt.Errorf("failed to marshal session: %w", err)
	}

	if err := os.WriteFile(m.historyFile, data, 0644); err != nil {
		return fmt.Errorf("failed to write session file: %w", err)
	}

	return nil
}

// Previous returns the previous entry in history (for up arrow)
func (m *Manager) Previous() (string, bool) {
	if len(m.session.Entries) == 0 {
		return "", false
	}

	if m.index > 0 {
		m.index--
	}

	if m.index >= 0 && m.index < len(m.session.Entries) {
		return m.session.Entries[m.index].Prompt, true
	}

	return "", false
}

// Next returns the next entry in history (for down arrow)
func (m *Manager) Next() (string, bool) {
	if len(m.session.Entries) == 0 {
		return "", false
	}

	if m.index < len(m.session.Entries) {
		m.index++
	}

	if m.index >= len(m.session.Entries) {
		// Past the end, return empty (current input)
		return "", true
	}

	return m.session.Entries[m.index].Prompt, true
}

// ResetIndex resets the history navigation index
func (m *Manager) ResetIndex() {
	m.index = len(m.session.Entries)
}

// Clear clears all history for the current session
func (m *Manager) Clear() {
	m.session.Entries = []Entry{}
	m.session.UpdatedAt = time.Now()
	m.index = 0
	m.Save()
}

// GetSessionID returns the current session ID
func (m *Manager) GetSessionID() string {
	return m.session.ID
}

// GetEntryCount returns the number of entries in history
func (m *Manager) GetEntryCount() int {
	return len(m.session.Entries)
}

// ListSessions returns all available session IDs
func (m *Manager) ListSessions() ([]string, error) {
	files, err := os.ReadDir(m.historyDir)
	if err != nil {
		return nil, err
	}

	var sessions []string
	for _, file := range files {
		if !file.IsDir() && filepath.Ext(file.Name()) == ".json" {
			// Extract session ID from filename
			name := file.Name()
			if len(name) > 13 { // "session_" + ".json"
				sessionID := name[8 : len(name)-5]
				sessions = append(sessions, sessionID)
			}
		}
	}

	return sessions, nil
}

// DeleteSession deletes a session file
func (m *Manager) DeleteSession(sessionID string) error {
	historyFile := filepath.Join(m.historyDir, fmt.Sprintf("session_%s.json", sessionID))
	return os.Remove(historyFile)
}

// ClearAllHistory deletes all history files
func (m *Manager) ClearAllHistory() error {
	files, err := os.ReadDir(m.historyDir)
	if err != nil {
		return err
	}

	for _, file := range files {
		if !file.IsDir() && filepath.Ext(file.Name()) == ".json" {
			path := filepath.Join(m.historyDir, file.Name())
			os.Remove(path)
		}
	}

	// Clear current session
	m.Clear()

	return nil
}
