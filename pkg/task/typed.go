// Copyright © 2025 Colden Cullen
// SPDX-License-Identifier: MIT

package task

import (
	"context"
	"fmt"

	"cuelang.org/go/cue"
)

type TypedTask[Params any] struct {
	Task

	Args Params
}

func NewTyped[Params any](
	session Session,
	executor, name string,
	cuectx *cue.Context,
	params Params,
	inputs ...string,
) TypedTask[Params] {
	return TypedTask[Params]{
		Task: Task{
			ID: TaskId{
				Executor: executor,
				Name:     name,
			},
			Session: session,
			Inputs:  inputs,
			Params:  cuectx.Encode(params),
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

// TypedExecutor is like Executor, but for tasks that have a specific type of parameters to expect.
type TypedExecutor[Params any] interface {
	Name() string
	Execute(ctx context.Context, tsk TypedTask[Params], result *Result) error
}

type wrappedExecutor[Params any] struct {
	TypedExecutor[Params]

	cuectx *cue.Context
}

var (
	_ Executor       = (*wrappedExecutor[any])(nil)
	_ SessionManager = (*wrappedExecutor[any])(nil)
)

// WrapTypedExecutor accepts a TypedExecutor and wraps it into an untyped Executor.
func WrapTypedExecutor[Params any](
	cuectx *cue.Context,
	impl TypedExecutor[Params],
) Executor {
	return wrappedExecutor[Params]{
		TypedExecutor: impl,
		cuectx:        cuectx,
	}
}

func (wrapped wrappedExecutor[Params]) OpenSession(ctx context.Context, session Session) error {
	if ssm, ok := wrapped.TypedExecutor.(SessionManager); ok {
		return ssm.OpenSession(ctx, session) //nolint:wrapcheck
	}

	return nil
}

func (wrapped wrappedExecutor[Params]) CloseSession(ctx context.Context, sessionId SessionId) {
	if ssm, ok := wrapped.TypedExecutor.(SessionManager); ok {
		ssm.CloseSession(ctx, sessionId)
	}
}

func (wrapped wrappedExecutor[Params]) Execute(
	ctx context.Context,
	tsk Task,
	result *Result,
) error {
	return wrapped.TypedExecutor.Execute( //nolint:wrapcheck
		ctx,
		Wrap[Params](wrapped.cuectx, tsk),
		result,
	)
}
