// Copyright © 2025 Colden Cullen
// SPDX-License-Identifier: MIT

package task // import "go.bonk.build/pkg/task"

import (
	"fmt"
	"os"
	"path"

	"cuelang.org/go/cue"
)

type TaskId struct {
	id       string
	executor string
}

func (id *TaskId) String() string {
	return fmt.Sprintf("%s:%s", id.id, id.executor)
}

func (id *TaskId) GetChild(name, executor string) TaskId {
	return TaskId{
		executor: executor,
		id:       fmt.Sprintf("%s:%s", id.id, name),
	}
}

func (id *TaskId) GetOutputDirectory() string {
	return path.Join(".bonk", id.String())
}

func (id *TaskId) OpenRoot() (*os.Root, error) {
	path := id.GetOutputDirectory()
	err := os.MkdirAll(path, 0o750)
	if err != nil {
		return nil, fmt.Errorf("failed to create task output dir %s: %w", path, err)
	}
	root, err := os.OpenRoot(path)
	if err != nil {
		return nil, fmt.Errorf("failed to open task output root: %w", err)
	}

	return root, nil
}

type Task struct {
	ID TaskId `json:"id"`

	Inputs []string  `json:"inputs,omitempty"`
	Params cue.Value `json:"params,omitempty"`
}

func New(executor, id string, params cue.Value, inputs ...string) Task {
	return Task{
		ID: TaskId{
			executor: executor,
			id:       id,
		},
		Inputs: inputs,
		Params: params,
	}
}

func (t *Task) Executor() string {
	return t.ID.executor
}

func (t *Task) GetOutputDirectory() string {
	return t.ID.GetOutputDirectory()
}

type TaskResult struct {
	Outputs       []string `json:"outputs,omitempty"`
	FollowupTasks []Task   `json:"followupTasks,omitempty"`
}
