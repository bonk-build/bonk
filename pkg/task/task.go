// Copyright © 2025 Colden Cullen
// SPDX-License-Identifier: MIT

package task // import "go.bonk.build/pkg/task"

import (
	"context"
	"fmt"
	"path"

	"cuelang.org/go/cue"

	"github.com/spf13/afero"
)

type TaskId struct {
	Name     string `json:"name"`
	Executor string `json:"executor"`
}

func (id *TaskId) String() string {
	return id.Name
}

func (id *TaskId) GetChild(name, executor string) TaskId {
	return TaskId{
		Executor: executor,
		Name:     fmt.Sprintf("%s.%s", id.Name, name),
	}
}

func (id *TaskId) GetOutDirectory() string {
	return path.Join(".bonk", id.Name)
}

type Task struct {
	ID      TaskId  `json:"id"`
	Session Session `json:"-"`

	Inputs []string  `json:"inputs,omitempty"`
	Params cue.Value `json:"params"`

	OutputFs afero.Fs `json:"-"`
}

func New(session Session, executor, name string, params cue.Value, inputs ...string) Task {
	tskId := TaskId{
		Executor: executor,
		Name:     name,
	}

	return Task{
		ID:      tskId,
		Session: session,
		Inputs:  inputs,
		Params:  params,

		OutputFs: afero.NewBasePathFs(session.FS(), tskId.GetOutDirectory()),
	}
}

func (t *Task) Executor() string {
	return t.ID.Executor
}

// Executor is the interface required to execute tasks.
type Executor interface {
	Name() string
	Execute(ctx context.Context, tsk Task, result *Result) error
}
