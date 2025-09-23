// Copyright © 2025 Colden Cullen
// SPDX-License-Identifier: MIT

package task

import (
	"github.com/spf13/afero"
)

// Task represents a unit of work to be executed.
type Task struct {
	// ID describes how this task is addressed.
	ID ID `json:"id"`
	// Executor describes where to route this task for execution.
	Executor string `json:"executor"`
	// Session manages the filesystem access for this task.
	Session Session `json:"-"`

	// Inputs describes any files that may be consumed by this task (relative to [Session.SourceFS]).
	Inputs []string `json:"inputs,omitempty"`
	// Args contains any arguments that may be passed to the executor.
	Args any `json:"args"`
}

// New creates a new task with the given parameters.
func New(
	id ID,
	session Session,
	executor string,
	args any,
) *Task {
	result := &Task{
		ID:       id,
		Executor: executor,
		Session:  session,
		Args:     args,
	}

	return result
}

// WithInputs appends input specifiers to this task.
func (tsk *Task) WithInputs(inputs ...string) *Task {
	tsk.Inputs = append(tsk.Inputs, inputs...)

	return tsk
}

// OutputFS returns the output filesystem for this task.
func (tsk *Task) OutputFS() afero.Fs {
	return afero.NewBasePathFs(tsk.Session.OutputFS(), tsk.ID.String())
}
