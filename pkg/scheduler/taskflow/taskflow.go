// Copyright © 2025 Colden Cullen
// SPDX-License-Identifier: MIT

// taskflow provides a scheduler backed by [github.com/noneback/go-taskflow].
package taskflow

import (
	"context"
	"fmt"
	"log/slog"

	"go.uber.org/multierr"

	gotaskflow "github.com/noneback/go-taskflow"
	slogctx "github.com/veqryn/slog-context"

	"go.bonk.build/pkg/scheduler"
	"go.bonk.build/pkg/task"
)

type tfScheduler struct {
	child    task.GenericExecutor
	executor gotaskflow.Executor
	tasks    map[string]*gotaskflow.Task

	flowHasTasks bool
	rootFlow     *gotaskflow.TaskFlow
}

var _ scheduler.Scheduler = (*tfScheduler)(nil)

func New(concurrency uint) scheduler.SchedulerFactory {
	return func(_ context.Context, child task.GenericExecutor) scheduler.Scheduler {
		return &tfScheduler{
			child:    child,
			executor: gotaskflow.NewExecutor(concurrency),
			tasks:    make(map[string]*gotaskflow.Task),

			flowHasTasks: false,
			rootFlow:     gotaskflow.NewTaskFlow("bonk"),
		}
	}
}

func (s *tfScheduler) AddTask(ctx context.Context, tsk *task.GenericTask, deps ...string) error {
	ctx = slogctx.Append(ctx, "task", tsk.ID.String())
	ctx = slogctx.Append(ctx, "executor", tsk.Executor)

	taskName := tsk.ID.String()
	s.flowHasTasks = true
	newTask := s.rootFlow.NewTask(taskName, func() {
		var result task.Result
		err := s.child.Execute(ctx, tsk, &result)
		if err != nil {
			slog.DebugContext(ctx, "error executing task", "error", err)

			return
		}

		var followupErr error
		for _, followup := range result.FollowupTasks {
			multierr.AppendInto(&followupErr, s.AddTask(ctx, &followup))
		}
		if followupErr != nil {
			slog.ErrorContext(ctx, "failed to schedule followup tasks", "error", followupErr)
		}
	})

	for _, dep := range deps {
		depTask, ok := s.tasks[dep]
		if !ok {
			return fmt.Errorf("could not find dependency task %s", dep)
		}

		newTask.Succeed(depTask)
	}

	s.tasks[taskName] = newTask

	return nil
}

func (s *tfScheduler) Run() {
	for s.flowHasTasks {
		existingFlow := s.rootFlow

		s.flowHasTasks = false
		s.rootFlow = gotaskflow.NewTaskFlow("bonk")

		s.executor.Run(existingFlow).Wait()
	}
}
