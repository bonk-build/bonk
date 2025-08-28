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
	root, err := os.OpenRoot(id.GetOutputDirectory())
	if err != nil {
		return nil, fmt.Errorf("failed to open task output root: %w", err)
	}

	return root, nil
}

func (id *TaskId) LoadStateFile() (*state, error) {
	fs, err := id.OpenRoot()
	if err != nil {
		return nil, err
	}

	return LoadState(fs)
}

type Task struct {
	ID TaskId `json:"id"`

	Inputs []string  `json:"inputs,omitempty"`
	Params cue.Value `json:"params,omitempty"`

	state *state
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

func (t *Task) SaveState(result *TaskResult) error {
	root, err := t.ID.OpenRoot()
	if err != nil {
		return err
	}
	t.state, err = NewState(t.ID.executor, t.Params, root, t.Inputs, result)
	if err != nil {
		return err
	}

	return t.state.Save(root)
}

func (t *Task) DetectStateMismatches() []string {
	root, err := t.ID.OpenRoot()
	if err != nil {
		return []string{"<missing>"}
	}
	if t.state == nil {
		t.state, err = LoadState(root)
		if err != nil {
			return []string{"<load failed>"}
		}
	}

	return t.state.DetectMismatches(t.ID.executor, t.Params, root, t.Inputs)
}

type TaskResult struct {
	Outputs       []string `json:"outputs,omitempty"`
	FollowupTasks []Task   `json:"followupTasks,omitempty"`
}
