// Copyright © 2025 Colden Cullen
// SPDX-License-Identifier: MIT

package task

import (
	"strings"
)

// TaskIDSep is the string placed between parts of a hierarchical [ID].
const TaskIDSep = "."

// ID represents is a way to address an individual task.
type ID string

// NewID creates a new TaskID from a series of parts.
func NewID(parts ...string) ID {
	return ID(strings.Join(parts, TaskIDSep))
}

func (id ID) String() string {
	return string(id)
}

// GetChild returns a new TaskID which is a child of the current one.
func (id ID) GetChild(names ...string) ID {
	parts := append([]string{id.String()}, names...)

	return ID(strings.Join(parts, TaskIDSep))
}

// Cut is a helper for calling [strings.Cut].
func (id ID) Cut() (string, string, bool) {
	return strings.Cut(id.String(), TaskIDSep)
}
