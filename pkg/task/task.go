// Copyright © 2025 Colden Cullen
// SPDX-License-Identifier: MIT

package task // import "go.bonk.build/pkg/task"

import (
	"github.com/spf13/afero"
)

type Task[Params any] struct {
	ID       TaskID  `json:"id"`
	Executor string  `json:"executor"`
	Session  Session `json:"-"`

	Inputs []string `json:"inputs,omitempty"`
	Args   Params   `json:"args"`
}

type GenericTask = Task[any]

func New[Params any](
	id string,
	session Session,
	executor string,
	args Params,
) *Task[Params] {
	result := &Task[Params]{
		ID:       TaskID(id),
		Executor: executor,
		Session:  session,
		Args:     args,
	}

	return result
}

func (tsk *Task[Params]) WithInputs(inputs ...string) *Task[Params] {
	tsk.Inputs = append(tsk.Inputs, inputs...)

	return tsk
}

func (tsk *Task[Params]) OutputFS() afero.Fs {
	return afero.NewBasePathFs(tsk.Session.OutputFS(), tsk.ID.String())
}
