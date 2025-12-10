// Copyright © 2025 Colden Cullen
// SPDX-License-Identifier: MIT

package bubbletea

import (
	"reflect"

	"charm.land/lipgloss/v2"

	"github.com/davecgh/go-spew/spew"

	tea "charm.land/bubbletea/v2"

	"go.bonk.build/pkg/executor/observable"
)

// teaModel is responsible for handling task invocation and status tracking.
type teaModel struct {
	tree taskTree
	view tea.View

	debugDump bool
}

var _ tea.Model = (*teaModel)(nil)

// Init implements tea.Model.
func (t *teaModel) Init() tea.Cmd {
	cmds := make([]tea.Cmd, 0, 1)

	t.tree = newTaskTree()
	cmds = append(cmds, t.tree.Init())

	t.view = tea.NewView(nil)

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
			cmds = append(cmds, tea.Quit)
		}

	case observable.TaskStatusMsg:
		// noop

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
func (t *teaModel) View() tea.View {
	component := make([]string, 0, 2) //nolint:mnd

	component = append(component, t.tree.String())

	// Append empty string to get a blank line at the bottom
	component = append(component, "")

	t.view.SetContent(lipgloss.JoinVertical(lipgloss.Left, component...))
	return t.view
}
