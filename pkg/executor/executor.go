// Copyright © 2025 Colden Cullen
// SPDX-License-Identifier: MIT

package executor // import "go.bonk.build/pkg/executor"

import (
	"context"

	"cuelang.org/go/cue"

	"github.com/google/uuid"

	"go.bonk.build/pkg/task"
)

// Executors may optionally implement this interface to be alerted when session statuses change.
type SessionManager interface {
	OpenSession(ctx context.Context, session task.Session) error
	CloseSession(ctx context.Context, sessionId uuid.UUID)
}

type Executor interface {
	Name() string
	Execute(ctx context.Context, tsk task.Task, result *task.Result) error
}

type TypedExecutor[Params any] interface {
	Name() string
	Execute(ctx context.Context, tsk task.TypedTask[Params], result *task.Result) error
}

type wrappedExecutor struct {
	name         string
	openSession  func(ctx context.Context, session task.Session) error
	closeSession func(ctx context.Context, sessionId uuid.UUID)
	execute      func(ctx context.Context, tsk task.Task, result *task.Result) error
}

var (
	_ Executor       = (*wrappedExecutor)(nil)
	_ SessionManager = (*wrappedExecutor)(nil)
)

func WrapTypedExecutor[Params any](
	cuectx *cue.Context,
	impl TypedExecutor[Params],
) Executor {
	result := wrappedExecutor{
		name: impl.Name(),
		execute: func(ctx context.Context, tsk task.Task, result *task.Result) error {
			return impl.Execute(ctx, task.Wrap[Params](cuectx, tsk), result)
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

func (wrapped wrappedExecutor) OpenSession(ctx context.Context, session task.Session) error {
	if wrapped.openSession != nil {
		return wrapped.openSession(ctx, session)
	}

	return nil
}

func (wrapped wrappedExecutor) CloseSession(ctx context.Context, sessionId uuid.UUID) {
	if wrapped.closeSession != nil {
		wrapped.closeSession(ctx, sessionId)
	}
}

func (wrapped wrappedExecutor) Execute(
	ctx context.Context,
	tsk task.Task,
	result *task.Result,
) error {
	return wrapped.execute(ctx, tsk, result)
}
