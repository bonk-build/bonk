// Copyright © 2025 Colden Cullen
// SPDX-License-Identifier: MIT

package task

import (
	"fmt"
	"strings"
)

// TaskIDSep is the string placed between parts of a hierarchical [TaskID].
const TaskIDSep = "."

// TaskID represents is a way to address an individual task.
type TaskID string

// NewID creates a new TaskID from a series of parts.
func NewID(parts ...string) TaskID {
	return TaskID(strings.Join(parts, TaskIDSep))
}

func (id TaskID) String() string {
	return string(id)
}

// GetChild returns a new TaskID which is a child of the current one.
func (id TaskID) GetChild(name string) TaskID {
	return TaskID(fmt.Sprintf("%s.%s", id, name))
}

// Cut is a helper for calling [strings.Cut].
func (id TaskID) Cut() (string, string, bool) {
	return strings.Cut(id.String(), TaskIDSep)
}
