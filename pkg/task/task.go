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
	Name     string `json:"name"`
	Executor string `json:"executor"`
}

func (id *TaskId) String() string {
	return fmt.Sprintf("%s:%s", id.Name, id.Executor)
}

func (id *TaskId) GetChild(name, executor string) TaskId {
	return TaskId{
		Executor: executor,
		Name:     fmt.Sprintf("%s.%s", id.Name, name),
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

func New(executor, name string, params cue.Value, inputs ...string) Task {
	return Task{
		ID: TaskId{
			Executor: executor,
			Name:     name,
		},
		Inputs: inputs,
		Params: params,
	}
}

func (t *Task) Executor() string {
	return t.ID.Executor
}

type TaskResult struct {
	Outputs       []string `json:"outputs,omitempty"`
	FollowupTasks []Task   `json:"followupTasks,omitempty"`
}
