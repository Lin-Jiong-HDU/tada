package tui

import (
	"testing"

	"github.com/Lin-Jiong-HDU/tada/internal/core/queue"
	tea "github.com/charmbracelet/bubbletea"
)

func TestNewModel(t *testing.T) {
	tasks := []*queue.Task{
		{ID: "1", Status: queue.TaskStatusPending},
	}

	mdl := NewModel(tasks)

	if mdl == nil {
		t.Fatal("Expected non-nil model")
	}

	m, ok := mdl.(model)
	if !ok {
		t.Fatal("Expected model type")
	}

	if len(m.tasks) != 1 {
		t.Errorf("Expected 1 task, got %d", len(m.tasks))
	}

	if m.cursor != 0 {
		t.Errorf("Expected cursor at 0, got %d", m.cursor)
	}
}

func TestModel_Init(t *testing.T) {
	tasks := []*queue.Task{}
	mdl := NewModel(tasks)

	m, ok := mdl.(model)
	if !ok {
		t.Fatal("Expected model type")
	}

	cmd := m.Init()
	if cmd != nil {
		t.Error("Expected nil command from Init")
	}
}

func TestModel_Update_UpKey(t *testing.T) {
	tasks := []*queue.Task{
		{ID: "1", Status: queue.TaskStatusPending},
		{ID: "2", Status: queue.TaskStatusPending},
	}
	mdl := NewModel(tasks)
	m := mdl.(model)
	m.cursor = 1

	// Move up
	msg := tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'k'}}
	newModel, cmd := m.Update(msg)

	m2 := newModel.(model)
	if m2.cursor != 0 {
		t.Errorf("Expected cursor at 0, got %d", m2.cursor)
	}
	if cmd != nil {
		t.Error("Expected nil command")
	}
}

func TestModel_Update_DownKey(t *testing.T) {
	tasks := []*queue.Task{
		{ID: "1", Status: queue.TaskStatusPending},
		{ID: "2", Status: queue.TaskStatusPending},
	}
	mdl := NewModel(tasks)
	m := mdl.(model)

	// Move down
	msg := tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'j'}}
	newModel, cmd := m.Update(msg)

	m2 := newModel.(model)
	if m2.cursor != 1 {
		t.Errorf("Expected cursor at 1, got %d", m2.cursor)
	}
	if cmd != nil {
		t.Error("Expected nil command")
	}
}

func TestModel_Update_AuthorizeKey(t *testing.T) {
	tasks := []*queue.Task{
		{ID: "1", Status: queue.TaskStatusPending},
	}
	mdl := NewModel(tasks)
	m := mdl.(model)

	msg := tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'a'}}
	newModel, cmd := m.Update(msg)

	if cmd == nil {
		t.Error("Expected command from authorize")
	}

	// Execute the command to get the result message
	resultMsg := cmd()
	if _, ok := resultMsg.(AuthorizeResultMsg); !ok {
		t.Error("Expected AuthorizeResultMsg from command")
	}

	// Update model with the result
	newModel2, _ := newModel.(model).Update(resultMsg)
	m2 := newModel2.(model)

	// Tasks execute immediately on approval, so status should be executing
	if m2.tasks[0].Status != queue.TaskStatusExecuting {
		t.Errorf("Expected status executing, got %s", m2.tasks[0].Status)
	}
}

func TestModel_Update_QuitKey(t *testing.T) {
	tasks := []*queue.Task{}
	mdl := NewModel(tasks)
	m := mdl.(model)

	msg := tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'q'}}
	_, cmd := m.Update(msg)

	if cmd == nil {
		t.Error("Expected quit command")
	}

	_, ok := cmd().(tea.QuitMsg)
	if !ok {
		t.Error("Expected QuitMsg")
	}
}
