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

type wrappedExecutor struct {
	name         string
	openSession  func(ctx context.Context, session Session) error
	closeSession func(ctx context.Context, sessionId SessionId)
	execute      func(ctx context.Context, tsk Task, result *Result) error
}

var (
	_ Executor       = (*wrappedExecutor)(nil)
	_ SessionManager = (*wrappedExecutor)(nil)
)

// WrapTypedExecutor accepts a TypedExecutor and wraps it into an untyped Executor.
func WrapTypedExecutor[Params any](
	cuectx *cue.Context,
	impl TypedExecutor[Params],
) Executor {
	result := wrappedExecutor{
		name: impl.Name(),
		execute: func(ctx context.Context, tsk Task, result *Result) error {
			return impl.Execute(ctx, Wrap[Params](cuectx, tsk), result)
		},
	}

	if sm, ok := impl.(SessionManager); ok {
		result.openSession = sm.OpenSession
		result.closeSession = sm.CloseSession
	}

	return result
}

func (wrapped wrappedExecutor) Name() string {
	return wrapped.name
}

func (wrapped wrappedExecutor) OpenSession(ctx context.Context, session Session) error {
	if wrapped.openSession != nil {
		return wrapped.openSession(ctx, session)
	}

	return nil
}

func (wrapped wrappedExecutor) CloseSession(ctx context.Context, sessionId SessionId) {
	if wrapped.closeSession != nil {
		wrapped.closeSession(ctx, sessionId)
	}
}

func (wrapped wrappedExecutor) Execute(
	ctx context.Context,
	tsk Task,
	result *Result,
) error {
	return wrapped.execute(ctx, tsk, result)
}
