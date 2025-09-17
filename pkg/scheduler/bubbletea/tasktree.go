// Copyright © 2025 Colden Cullen
// SPDX-License-Identifier: MIT

package bubbletea

import (
	"strings"

	"github.com/charmbracelet/lipgloss/v2"
	"github.com/charmbracelet/lipgloss/v2/tree"

	tea "github.com/charmbracelet/bubbletea/v2"
)

// taskTree is responsible for rendering task state to the terminal.
type taskTree struct {
	node *tree.Tree

	children map[string]*taskTree
}

var (
	_ tea.Model     = (*taskTree)(nil)
	_ tea.ViewModel = (*taskTree)(nil)
)

func newTaskTree() *taskTree {
	return &taskTree{
		node:     tree.New(),
		children: make(map[string]*taskTree),
	}
}

// Init implements tea.Model.
func (t *taskTree) Init() tea.Cmd {
	return nil
}

// Update implements tea.Model.
func (t *taskTree) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	var cmds []tea.Cmd

	if msg, ok := msg.(TaskStatusMsg); ok {
		this, children, hasChildren := strings.Cut(msg.tskId, ".")

		child, ok := t.children[this]

		// If there isn't already a child node, add & initialize it
		if !ok {
			child = newTaskTree()
			t.node.Child(child.node.Root(this))
			cmds = append(cmds, child.Init())

			t.children[this] = child
		}

		if hasChildren {
			child.Update(TaskStatusMsg{
				tskId:  children,
				status: msg.status,
			})
		} else {
			style := StatusStyleClear[msg.status]

			parts := []any{
				style.Emoji,
				style.Padding(0, 2).Render(this), //nolint:mnd
			}

			if msg.err != nil {
				parts = append(parts, msg.err)
			}

			child.node.Root(lipgloss.Sprint(parts...))
		}
	}

	return t, tea.Batch(cmds...)
}

// View implements tea.ViewModel.
func (t *taskTree) View() string {
	// Assume the root is empty and we can ignore it, just print the children
	children := t.node.Children()
	components := make([]string, children.Length())
	for idx := range children.Length() {
		components[idx] = children.At(idx).String()
	}

	return lipgloss.JoinVertical(lipgloss.Left, components...)
}
