// Copyright © 2025 Colden Cullen
// SPDX-License-Identifier: MIT

package task

import (
	"github.com/spf13/afero"
)

type Task struct {
	ID       TaskID  `json:"id"`
	Executor string  `json:"executor"`
	Session  Session `json:"-"`

	Inputs []string `json:"inputs,omitempty"`
	Args   any      `json:"args"`
}

func New(
	id TaskID,
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

func (tsk *Task) WithInputs(inputs ...string) *Task {
	tsk.Inputs = append(tsk.Inputs, inputs...)

	return tsk
}

func (tsk *Task) OutputFS() afero.Fs {
	return afero.NewBasePathFs(tsk.Session.OutputFS(), tsk.ID.String())
}
