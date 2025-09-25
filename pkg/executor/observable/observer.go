// Copyright © 2025 Colden Cullen
// SPDX-License-Identifier: MIT

// Package observable provides an executor which alerts followers to task status
package observable

import (
	"context"
	"log/slog"

	"atomicgo.dev/event"

	"go.bonk.build/pkg/task"
)

type Observable interface {
	task.Executor

	Listen(f func(TaskStatusMsg)) error
}

func New(exec task.Executor) Observable {
	return observable{
		Executor: exec,
		Event:    event.New[TaskStatusMsg](),
	}
}

type observable struct {
	task.Executor
	*event.Event[TaskStatusMsg]
}

func (obs observable) Execute(ctx context.Context, tsk *task.Task, result *task.Result) error {
	triggerErr := obs.Trigger(TaskRunningMsg(tsk.ID))
	if triggerErr != nil {
		slog.WarnContext(ctx, "failed to trigger task status message")
	}

	err := obs.Executor.Execute(ctx, tsk, result)

	triggerErr = obs.Trigger(TaskFinishedMsg(tsk.ID, err))
	if triggerErr != nil {
		slog.WarnContext(ctx, "failed to trigger task status message")
	}

	return err //nolint:wrapcheck
}
