// Copyright © 2025 Colden Cullen
// SPDX-License-Identifier: MIT

package bubbletea

import (
	"reflect"
	"sync/atomic"

	"github.com/charmbracelet/lipgloss/v2"
	"github.com/davecgh/go-spew/spew"

	tea "github.com/charmbracelet/bubbletea/v2"

	"go.bonk.build/pkg/executor/observable"
)

// teaModel is responsible for handling task invocation and status tracking.
type teaModel struct {
	tree taskTree

	tasks atomic.Int64

	quitting  bool
	debugDump bool
}

var (
	_ tea.Model     = (*teaModel)(nil)
	_ tea.ViewModel = (*teaModel)(nil)
)

// Init implements tea.Model.
func (t *teaModel) Init() tea.Cmd {
	cmds := make([]tea.Cmd, 0, 1)

	t.tree = newTaskTree()
	cmds = append(cmds, t.tree.Init())

	return tea.Batch(cmds...)
}

// Update implements tea.Model.
func (t *teaModel) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	var (
		cmd  tea.Cmd
		cmds []tea.Cmd
	)

	switch msg := msg.(type) {
	case tea.KeyMsg:
		if msg.Key().Mod.Contains(tea.ModCtrl) && msg.Key().Code == 'c' {
			t.quitting = true
			cmds = append(cmds, tea.Quit)
		}

	case TaskScheduleMsg:
		t.tasks.Add(1)
		cmds = append(cmds, msg.GetExecCmd())

	case observable.TaskStatusMsg:
		if msg.Status != observable.StatusRunning {
			remaining := t.tasks.Add(-1)
			if remaining == 0 {
				t.quitting = true
				cmds = append(cmds, tea.Quit)
			}
		}

	default:
		if t.debugDump && reflect.TypeOf(msg).Name() != "printLineMessage" {
			cmds = append(cmds, tea.Println(spew.Sdump(msg)))
		}
	}

	// Pass the events down to the tree
	_, cmd = t.tree.Update(msg)
	cmds = append(cmds, cmd)

	return t, tea.Batch(cmds...)
}

// View implements tea.ViewModel.
func (t *teaModel) View() string {
	component := make([]string, 0, 1)

	component = append(component, t.tree.View())

	// Add final newline
	if t.quitting {
		component = append(component, "")
	}

	return lipgloss.JoinVertical(lipgloss.Left, component...)
}
