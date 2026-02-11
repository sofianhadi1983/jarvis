package todo

import (
	"fmt"
	"strings"
)

type TodoStatus string

const (
	StatusPending    TodoStatus = "pending"
	StatusInProgress TodoStatus = "in_progress"
	StatusCompleted  TodoStatus = "completed"
)

type TodoItem struct {
	Content    string     `json:"content" jsonschema_description:"Brief task description"`
	Status     TodoStatus `json:"status" jsonschema_description:"pending, in_progress, or completed"`
	ActiveForm string     `json:"activeForm" jsonschema_description:"Present-tense action (e.g. 'Reading files...')"`
}

type TodoWriteInput struct {
	Todos []TodoItem `json:"todos" jsonschema_description:"Complete replacement list of todos (max 20)"`
}

type Manager struct {
	items    []TodoItem
	onUpdate func(activeForm string)
}

func NewManager(onUpdate func(activeForm string)) *Manager {
	return &Manager{
		onUpdate: onUpdate,
	}
}

func (m *Manager) Replace(items []TodoItem) (string, error) {
	if len(items) > 20 {
		return "", fmt.Errorf("too many todos: %d (max 20)", len(items))
	}

	inProgressCount := 0
	for _, item := range items {
		if strings.TrimSpace(item.Content) == "" {
			return "", fmt.Errorf("todo item content cannot be empty")
		}
		switch item.Status {
		case StatusPending, StatusInProgress, StatusCompleted:
		default:
			return "", fmt.Errorf("invalid status %q: must be pending, in_progress, or completed", item.Status)
		}
		if item.Status == StatusInProgress {
			inProgressCount++
		}
	}
	if inProgressCount > 1 {
		return "", fmt.Errorf("only 1 item can be in_progress at a time (found %d)", inProgressCount)
	}

	m.items = make([]TodoItem, len(items))
	copy(m.items, items)

	activeForm := ""
	for _, item := range m.items {
		if item.Status == StatusInProgress {
			activeForm = item.ActiveForm
			break
		}
	}

	if m.onUpdate != nil {
		m.onUpdate(activeForm)
	}

	return m.formatSummary(), nil
}

func (m *Manager) Reset() {
	m.items = nil
}

func (m *Manager) HasItems() bool {
	return len(m.items) > 0
}

func (m *Manager) formatSummary() string {
	var b strings.Builder
	for i, item := range m.items {
		if i > 0 {
			b.WriteString("\n")
		}
		switch item.Status {
		case StatusCompleted:
			b.WriteString("[x] ")
		case StatusInProgress:
			b.WriteString("[>] ")
		default:
			b.WriteString("[ ] ")
		}
		b.WriteString(item.Content)
	}
	return b.String()
}
