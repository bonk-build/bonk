// Copyright © 2025 Colden Cullen
// SPDX-License-Identifier: MIT

package scheduler // import "go.bonk.build/pkg/scheduler"

import (
	"context"
	"fmt"
	"log/slog"

	"go.uber.org/multierr"

	gotaskflow "github.com/noneback/go-taskflow"
	slogctx "github.com/veqryn/slog-context"

	"go.bonk.build/pkg/task"
)

type Scheduler interface {
	AddTask(ctx context.Context, tsk *task.GenericTask, deps ...string) error
	Run()
}

type scheduler struct {
	child    task.GenericExecutor
	executor gotaskflow.Executor
	tasks    map[string]*gotaskflow.Task

	flowHasTasks bool
	rootFlow     *gotaskflow.TaskFlow
}

var _ Scheduler = (*scheduler)(nil)

func NewScheduler(child task.GenericExecutor, concurrency uint) Scheduler {
	return &scheduler{
		child:    child,
		executor: gotaskflow.NewExecutor(concurrency),
		tasks:    make(map[string]*gotaskflow.Task),

		flowHasTasks: false,
		rootFlow:     gotaskflow.NewTaskFlow("bonk"),
	}
}

func (s *scheduler) AddTask(ctx context.Context, tsk *task.GenericTask, deps ...string) error {
	ctx = slogctx.Append(ctx, "task", tsk.ID.Name)
	ctx = slogctx.Append(ctx, "executor", tsk.ID.Executor)

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

func (s *scheduler) Run() {
	for s.flowHasTasks {
		existingFlow := s.rootFlow

		s.flowHasTasks = false
		s.rootFlow = gotaskflow.NewTaskFlow("bonk")

		s.executor.Run(existingFlow).Wait()
	}
}
