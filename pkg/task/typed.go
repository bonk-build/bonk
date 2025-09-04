// Copyright © 2025 Colden Cullen
// SPDX-License-Identifier: MIT

package task

import (
	"fmt"

	"cuelang.org/go/cue"

	"github.com/google/uuid"
)

type TypedTask[Params any] struct {
	Task

	Args Params
}

func NewTyped[Params any](
	session uuid.UUID,
	executor, name string,
	cuectx *cue.Context,
	params Params,
	inputs ...string,
) TypedTask[Params] {
	return TypedTask[Params]{
		Task: Task{
			ID: TaskId{
				Session:  session,
				Executor: executor,
				Name:     name,
			},
			Inputs: inputs,
			Params: cuectx.Encode(params),
		},
		Args: params,
	}
}

func Wrap[Params any](cuectx *cue.Context, tsk Task) TypedTask[Params] {
	result := TypedTask[Params]{
		Task: tsk,
	}

	err := result.Params.Decode(&result.Args)
	if err != nil {
		panic(fmt.Errorf("failed to decode parameters: %w", err))
	}

	return result
}
