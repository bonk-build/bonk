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
	ID TaskId

	Inputs []string
	Params cue.Value

	State *state
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

func (t *Task) SaveState(outputs []string) error {
	root, err := t.ID.OpenRoot()
	if err != nil {
		return err
	}
	t.State, err = NewState(t.ID.executor, t.Params, root, t.Inputs, outputs)
	if err != nil {
		return err
	}

	return t.State.Save(root)
}

func (t *Task) DetectStateMismatches() []string {
	root, err := t.ID.OpenRoot()
	if err != nil {
		return []string{"<missing>"}
	}
	if t.State == nil {
		t.State, err = LoadState(root)
		if err != nil {
			return []string{"<load failed>"}
		}
	}

	return t.State.DetectMismatches(t.ID.executor, t.Params, root, t.Inputs)
}
