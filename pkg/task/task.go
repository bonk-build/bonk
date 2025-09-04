// Copyright © 2025 Colden Cullen
// SPDX-License-Identifier: MIT

package task // import "go.bonk.build/pkg/task"

import (
	"fmt"
	"path"

	"cuelang.org/go/cue"

	"github.com/google/uuid"
	"github.com/spf13/afero"
)

type TaskId struct {
	Session  uuid.UUID `json:"-"`
	Name     string    `json:"name"`
	Executor string    `json:"executor"`
}

func (id *TaskId) String() string {
	return fmt.Sprintf("%s:%s", id.Name, id.Executor)
}

func (id *TaskId) GetChild(name, executor string) TaskId {
	return TaskId{
		Session:  id.Session,
		Executor: executor,
		Name:     fmt.Sprintf("%s.%s", id.Name, name),
	}
}

func (id *TaskId) GetOutputDirectory() string {
	return path.Join(".bonk", id.String())
}

func (id *TaskId) GetOutputFilesystem(project afero.Fs) (afero.Fs, error) {
	path := id.GetOutputDirectory()
	err := project.MkdirAll(path, 0o750)
	if err != nil {
		return nil, fmt.Errorf("failed to create task output dir %s: %w", path, err)
	}

	return afero.NewBasePathFs(project, path), nil
}

type Task struct {
	ID TaskId `json:"id"`

	Inputs []string  `json:"inputs,omitempty"`
	Params cue.Value `json:"params"`

	ProjectFs afero.Fs `json:"-"`
	OutputFs  afero.Fs `json:"-"`
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
