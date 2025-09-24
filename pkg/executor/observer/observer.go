// Copyright © 2025 Colden Cullen
// SPDX-License-Identifier: MIT

// Package observer provides an executor which alerts followers to task status
package observer

import (
	"context"
	"log/slog"

	"atomicgo.dev/event"

	"go.bonk.build/pkg/task"
)

type Observer interface {
	task.Executor

	Listen(f func(TaskStatusMsg)) error
}

func New(exec task.Executor) Observer {
	return observer{
		Executor: exec,
		Event:    event.New[TaskStatusMsg](),
	}
}

type observer struct {
	task.Executor
	*event.Event[TaskStatusMsg]
}

func (obs observer) Execute(ctx context.Context, tsk *task.Task, result *task.Result) error {
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
