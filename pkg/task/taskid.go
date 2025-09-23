// Copyright © 2025 Colden Cullen
// SPDX-License-Identifier: MIT

package task

import (
	"fmt"
	"strings"
)

const TaskIDSep = "."

type TaskID string

// NewID creates a new TaskID from a series of parts.
func NewID(parts ...string) TaskID {
	return TaskID(strings.Join(parts, TaskIDSep))
}

func (id TaskID) String() string {
	return string(id)
}

func (id TaskID) GetChild(name string) TaskID {
	return TaskID(fmt.Sprintf("%s.%s", id, name))
}

func (id TaskID) Cut() (string, string, bool) {
	return strings.Cut(id.String(), TaskIDSep)
}
