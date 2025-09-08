// Copyright © 2025 Colden Cullen
// SPDX-License-Identifier: MIT

package task // import "go.bonk.build/pkg/task"

import (
	"context"
	"fmt"
	"path"

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

type Task[Params any] struct {
	ID      TaskId  `json:"id"`
	Session Session `json:"-"`

	Inputs []string `json:"inputs,omitempty"`
	Args   Params   `json:"args"`

	OutputFs afero.Fs `json:"-"`
}

type GenericTask = Task[any]

func New[Params any](
	session Session,
	executor, name string,
	args Params,
	inputs ...string,
) *Task[Params] {
	tskId := TaskId{
		Executor: executor,
		Name:     name,
	}

	return &Task[Params]{
		ID:      tskId,
		Session: session,
		Inputs:  inputs,
		Args:    args,

		OutputFs: afero.NewBasePathFs(session.FS(), tskId.GetOutDirectory()),
	}
}

// Executor is the interface required to execute tasks.
type Executor[Params any] interface {
	Name() string
	Execute(ctx context.Context, tsk Task[Params], result *Result) error
}

type GenericExecutor = Executor[any]
