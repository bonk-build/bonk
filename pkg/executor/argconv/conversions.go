// Copyright © 2025 Colden Cullen
// SPDX-License-Identifier: MIT

// Package argconv provides an executor for automatically unboxing task parameters.
package argconv

import (
	"context"
	"fmt"
	"reflect"

	"cuelang.org/go/cuego"

	"github.com/go-viper/mapstructure/v2"

	"go.bonk.build/pkg/executor"
	"go.bonk.build/pkg/task"
)

//go:generate go tool mockgen -destination typedexecutor_mock.go -package argconv -copyright_file ../../../license-header.txt -typed  -write_package_comment=false . TypedExecutor

// TypedExecutor is like [executor.Executor] but with unboxed arguments.
type TypedExecutor[Params any] interface {
	OpenSession(ctx context.Context, session task.Session) error
	CloseSession(ctx context.Context, sessionID task.SessionID)
	Execute(ctx context.Context, tsk *task.Task, args *Params, result *task.Result) error
}

// UnboxArgs converts a task with generic arguments to a task with typed arguments.
func UnboxArgs[Params any](tsk *task.Task) (*Params, error) {
	paramsT := reflect.TypeFor[Params]()
	argsV := reflect.ValueOf(tsk.Args)
	argsT := argsV.Type()
	var ok bool
	var result *Params

	err := fmt.Errorf("failed to convert params from %s to %s", argsT, paramsT)

	switch argsT.Kind() {
	case paramsT.Kind():
		if argsT == paramsT {
			if argsV.CanAddr() {
				result, ok = argsV.Addr().Interface().(*Params)
				if !ok {
					return nil, err
				}
			} else {
				// yolo
				if tmpResult, ok := tsk.Args.(Params); ok {
					result = &tmpResult
				} else {
					return nil, err
				}
			}
		}

	case reflect.Map:
		// If the types don't match and we're given a map, use mapstructure to pull it out.
		result = new(Params)
		err := mapstructure.Decode(tsk.Args, result)
		if err != nil {
			return nil, fmt.Errorf("failed to decode args: %w", err)
		}

		return result, nil

	default:
		// yolo
		if tmpResult, ok := tsk.Args.(Params); ok {
			result = &tmpResult
		} else {
			return nil, err
		}
	}

	// Use cuego to complete / validate the type
	cueErr := cuego.Complete(result)
	if cueErr != nil {
		return nil, cueErr //nolint:wrapcheck // Want to expose cue errors to those who many want them
	}

	return result, nil
}

type wrappedExecutor[Params any] struct {
	TypedExecutor[Params]
}

var _ executor.Executor = (*wrappedExecutor[any])(nil)

// BoxExecutor accepts a TypedExecutor and wraps it into an untyped Executor.
func BoxExecutor[Params any](
	impl TypedExecutor[Params],
) executor.Executor {
	return wrappedExecutor[Params]{
		TypedExecutor: impl,
	}
}

func (wrapped wrappedExecutor[Params]) Execute(
	ctx context.Context,
	tsk *task.Task,
	result *task.Result,
) error {
	unboxed, err := UnboxArgs[Params](tsk)
	if err != nil {
		return err
	}

	return wrapped.TypedExecutor.Execute( //nolint:wrapcheck
		ctx,
		tsk,
		unboxed,
		result,
	)
}
