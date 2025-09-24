// Copyright © 2025 Colden Cullen
// SPDX-License-Identifier: MIT

package bubbletea

import (
	"strings"

	"github.com/charmbracelet/lipgloss/v2"
	"github.com/charmbracelet/lipgloss/v2/tree"
	"github.com/davecgh/go-spew/spew"
	"github.com/elliotchance/orderedmap/v3"

	"go.bonk.build/pkg/executor/observer"
)

// taskNode is responsible for rendering task state to the terminal.
type taskNode struct {
	name   string
	status observer.TaskStatus
	err    error

	children taskNodeChildren
}

var _ tree.Node = (*taskNode)(nil)

func makeTaskNode(name string) *taskNode {
	return &taskNode{
		name: name,
		children: taskNodeChildren{
			OrderedMap: orderedmap.NewOrderedMap[string, *taskNode](),
		},
	}
}

// Children implements tree.Node.
func (t *taskNode) Children() tree.Children {
	return t.children
}

// Hidden implements tree.Node.
func (t *taskNode) Hidden() bool {
	return false
}

// SetHidden implements tree.Node.
func (t *taskNode) SetHidden(bool) {
}

// Value implements tree.Node.
func (t *taskNode) Value() string {
	result := strings.Builder{}
	result.WriteString(t.name)

	if t.err != nil {
		result.WriteString(": ")
		result.WriteString(t.err.Error())
	}

	result.WriteString(
		strings.Repeat(" ", 12), //nolint:mnd // this is just to clear the buffer after the item
	)

	return result.String()
}

// SetValue implements tree.Node.
func (t *taskNode) SetValue(value any) {
	if status, ok := value.(observer.TaskStatusMsg); ok {
		t.status = status.Status
		t.err = status.Error
	} else {
		panic("unimplemented " + spew.Sdump(value))
	}
}

// String implements tree.Node.
func (t *taskNode) String() string {
	panic("unimplemented")
}

type taskNodeChildren struct {
	*orderedmap.OrderedMap[string, *taskNode]
}

var _ tree.Children = taskNodeChildren{}

// At implements tree.Children.
func (t taskNodeChildren) At(index int) tree.Node {
	cur := t.Front()
	for range index {
		cur = cur.Next()
	}

	return cur.Value
}

// Length implements tree.Children.
func (t taskNodeChildren) Length() int {
	return t.Len()
}

func taskNodeStyle(style StatusStyles) tree.StyleFunc {
	return func(children tree.Children, i int) lipgloss.Style {
		item, ok := children.At(i).(*taskNode)
		if !ok {
			panic("unsupported node")
		}

		return style[item.status]
	}
}
