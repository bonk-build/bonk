// Copyright © 2025 Colden Cullen
// SPDX-License-Identifier: MIT

package bubbletea

import (
	"strings"

	"charm.land/lipgloss/v2/tree"

	tea "charm.land/bubbletea/v2"

	"go.bonk.build/pkg/executor/observable"
)

type taskTree struct {
	tree.Tree

	view tea.View
}

var _ tea.Model = (*taskTree)(nil)

func newTaskTree() taskTree {
	return taskTree{
		Tree: *tree.New().
			Enumerator(tree.RoundedEnumerator).
			ItemStyleFunc(taskNodeStyle(StatusStyleClear)),

		view: tea.NewView(nil),
	}
}

// Init implements tea.Model.
func (t *taskTree) Init() tea.Cmd {
	return nil
}

// Update implements tea.Model.
func (t *taskTree) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	var cmds []tea.Cmd

	if msg, ok := msg.(observable.TaskStatusMsg); ok {
		curName, childPath, hasChildren := msg.TaskID.Cut()

		var cur *taskNode

		// Search the top level list for the node
		treeChildren := t.Children()
		for idx := range treeChildren.Length() {
			taskNode, ok := treeChildren.At(idx).(*taskNode)
			if !ok {
				panic("unexpected child!")
			}
			if taskNode.name == curName {
				cur = taskNode

				break
			}
		}
		if cur == nil {
			cur = makeTaskNode(curName)
			t.Child(cur)
		}

		// Now find the sub task inside of that
		for hasChildren {
			curName, childPath, hasChildren = strings.Cut(childPath, ".")

			newChild, ok := cur.children.Get(curName)

			// If there isn't already a child node, add & initialize it
			if !ok {
				newChild = makeTaskNode(curName)
				cur.children.Set(curName, newChild)
			}

			cur = newChild
		}

		if cur == nil {
			panic("invalid!")
		}

		// Now update cur
		cur.SetValue(msg)
	}

	return t, tea.Batch(cmds...)
}

// View implements tea.ViewModel.
func (t *taskTree) View() tea.View {
	t.view.SetContent(t)
	return t.view
}
