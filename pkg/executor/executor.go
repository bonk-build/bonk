// Copyright © 2025 Colden Cullen
// SPDX-License-Identifier: MIT

package executor // import "go.bonk.build/pkg/executor"

import (
	"context"

	"cuelang.org/go/cue"

	"go.bonk.build/pkg/task"
)

type Executor interface {
	Execute(ctx context.Context, tsk task.Task, result *task.TaskResult) error
}

type TypedExecutor[Params any] interface {
	Execute(ctx context.Context, tsk task.TypedTask[Params], result *task.TaskResult) error
}

type wrappedExecutor struct {
	thunk func(ctx context.Context, tsk task.Task, result *task.TaskResult) error
}

var _ Executor = (*wrappedExecutor)(nil)

func WrapTypedExecutor[Params any](
	cuectx cue.Context,
	impl TypedExecutor[Params],
	result *task.TaskResult,
) Executor {
	return wrappedExecutor{
		thunk: func(ctx context.Context, tsk task.Task, result *task.TaskResult) error {
			return impl.Execute(ctx, task.Wrap[Params](cuectx, tsk), result)
		},
	}
}

func (wrapped wrappedExecutor) Execute(
	ctx context.Context,
	tsk task.Task,
	result *task.TaskResult,
) error {
	return wrapped.thunk(ctx, tsk, result)
}
