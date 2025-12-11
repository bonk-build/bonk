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

	// Inputs describes any files that may be consumed by this task (relative to [Session.SourceFS]).
	Inputs []FileReference `json:"inputs,omitempty"`
	// Dependencies contains a list of tasks which must be completed before this task can run.
	Dependencies []ID `json:"dependencies,omitempty"`
	// Args contains any arguments that may be passed to the executor.
	Args any `json:"args"`
}

type Option func(*Task)

// New creates a new task with the given parameters.
func New(
	id ID,
	executor string,
	args any,
	options ...Option,
) *Task {
	result := &Task{
		ID:       id,
		Executor: executor,
		Args:     args,
	}

	for _, opt := range options {
		opt(result)
	}

	return result
}

// WithInputs appends input specifiers to this task.
func WithInputs(inputs ...FileReference) Option {
	return func(tsk *Task) {
		tsk.Inputs = append(tsk.Inputs, inputs...)
	}
}

// WithDependencies appends input specifiers to this task.
func WithDependencies(dependencies ...ID) Option {
	return func(tsk *Task) {
		tsk.Dependencies = append(tsk.Dependencies, dependencies...)
	}
}

func (tsk *Task) OpenInputs(session Session) ([]afero.File, error) {
	result := make([]afero.File, len(tsk.Inputs))
	for idx, file := range tsk.Inputs {
		var err error
		switch file.FileSystem {
		case FsSource:
			result[idx], err = session.SourceFS().Open(file.Path)

		case FsOutput:
			result[idx], err = OutputFS(session, tsk.ID).Open(file.Path)
		}

		if err != nil {
			return nil, err
		}
	}
	return result, nil
}
