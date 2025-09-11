// Copyright © 2025 Colden Cullen
// SPDX-License-Identifier: MIT

package task

import (
	"context"
	"fmt"
	"reflect"

	"github.com/go-viper/mapstructure/v2"
)

// Box converts a task with typed arguments to a task with generic arguments.
func (t *Task[Params]) Box() *GenericTask {
	return &GenericTask{
		ID:      t.ID,
		Session: t.Session,
		Inputs:  t.Inputs,
		Args:    t.Args,
	}
}

// Unbox converts a task with generic arguments to a task with typed arguments.
func Unbox[Params any](tsk *GenericTask) (*Task[Params], error) {
	paramsT := reflect.TypeFor[Params]()
	argsT := reflect.TypeOf(tsk.Args)

	result := &Task[Params]{
		ID:      tsk.ID,
		Session: tsk.Session,
		Inputs:  tsk.Inputs,
	}

	var convSuccess bool
	switch argsT.Kind() {
	case paramsT.Kind():
		result.Args, convSuccess = tsk.Args.(Params)
		if !convSuccess {
			return nil, fmt.Errorf("failed to convert params from %s to %s", paramsT, argsT)
		}

	case reflect.Map:
		// If the types don't match and we're given a map, use mapstructure to pull it out.
		err := mapstructure.Decode(tsk.Args, &result.Args)
		if err != nil {
			return nil, fmt.Errorf("failed to decode args: %w", err)
		}

	default:
		result.Args, convSuccess = tsk.Args.(Params)
		if !convSuccess {
			return nil, fmt.Errorf("failed to convert params from %s to %s", paramsT, argsT)
		}
	}

	return result, nil
}

type wrappedExecutor[Params any] struct {
	Executor[Params]
}

var _ GenericExecutor = (*wrappedExecutor[any])(nil)

// BoxExecutor accepts a TypedExecutor and wraps it into an untyped Executor.
func BoxExecutor[Params any](
	impl Executor[Params],
) GenericExecutor {
	return wrappedExecutor[Params]{
		Executor: impl,
	}
}

func (wrapped wrappedExecutor[Params]) Execute(
	ctx context.Context,
	tsk *GenericTask,
	result *Result,
) error {
	unboxed, err := Unbox[Params](tsk)
	if err != nil {
		return err
	}

	return wrapped.Executor.Execute( //nolint:wrapcheck
		ctx,
		unboxed,
		result,
	)
}
