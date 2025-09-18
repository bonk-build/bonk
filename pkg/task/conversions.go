// Copyright © 2025 Colden Cullen
// SPDX-License-Identifier: MIT

package task

import (
	"context"
	"fmt"
	"reflect"

	"cuelang.org/go/cuego"

	"github.com/go-viper/mapstructure/v2"
)

// Box converts a task with typed arguments to a task with generic arguments.
func (t *Task[Params]) Box() *GenericTask {
	if generic, ok := any(t).(*GenericTask); ok {
		return generic
	}

	return &GenericTask{
		ID:       t.ID,
		Executor: t.Executor,
		Session:  t.Session,
		Inputs:   t.Inputs,
		Args:     t.Args,
	}
}

// Unbox converts a task with generic arguments to a task with typed arguments.
func Unbox[Params any](tsk *GenericTask) (*Task[Params], error) {
	paramsT := reflect.TypeFor[Params]()
	argsT := reflect.TypeOf(tsk.Args)

	result := &Task[Params]{
		ID:       tsk.ID,
		Executor: tsk.Executor,
		Session:  tsk.Session,
		Inputs:   tsk.Inputs,
	}

	var convSuccess bool
	switch argsT.Kind() {
	case paramsT.Kind():
		result.Args, convSuccess = tsk.Args.(Params)
		if !convSuccess {
			return nil, fmt.Errorf("failed to convert params from %s to %s", argsT, paramsT)
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
			return nil, fmt.Errorf("failed to convert params from %s to %s", argsT, paramsT)
		}
	}

	// Use cuego to complete / validate the type
	err := cuego.Complete(&result.Args)
	if err != nil {
		return nil, err //nolint:wrapcheck // Want to expose cue errors to those who many want them
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
	if generic, ok := impl.(GenericExecutor); ok {
		return generic
	}

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
