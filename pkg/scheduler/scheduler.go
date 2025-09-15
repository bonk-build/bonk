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

type Scheduler struct {
	executorManager task.GenericExecutor
	executor        gotaskflow.Executor
	tasks           map[string]*gotaskflow.Task

	flowHasTasks bool
	rootFlow     *gotaskflow.TaskFlow
}

func NewScheduler(executorManager task.GenericExecutor, concurrency uint) *Scheduler {
	return &Scheduler{
		executorManager: executorManager,
		executor:        gotaskflow.NewExecutor(concurrency),
		tasks:           make(map[string]*gotaskflow.Task),

		flowHasTasks: false,
		rootFlow:     gotaskflow.NewTaskFlow("bonk"),
	}
}

func (s *Scheduler) AddTask(ctx context.Context, tsk *task.GenericTask, deps ...string) error {
	ctx = slogctx.Append(ctx, "task", tsk.ID.Name)
	ctx = slogctx.Append(ctx, "executor", tsk.ID.Executor)

	taskName := tsk.ID.String()
	s.flowHasTasks = true
	newTask := s.rootFlow.NewTask(taskName, func() {
		mismatches := DetectStateMismatches(tsk)
		if mismatches == nil {
			slog.DebugContext(ctx, "states match, skipping task")

			return
		}

		slog.DebugContext(ctx, "state mismatch, running task", "mismatches", mismatches)

		var result task.Result
		err := s.executorManager.Execute(ctx, tsk, &result)
		if err != nil {
			slog.DebugContext(ctx, "error executing task", "error", err)

			return
		}

		slog.Info("task succeeded, saving state")

		var followupErr error
		for _, followup := range result.FollowupTasks {
			multierr.AppendInto(&followupErr, s.AddTask(ctx, &followup))
		}
		if followupErr != nil {
			slog.ErrorContext(ctx, "failed to schedule followup tasks", "error", followupErr)
		}

		err = SaveState(tsk, &result)
		if err != nil {
			slog.ErrorContext(ctx, "failed to save task state", "error", err)

			return
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

func (s *Scheduler) Run() {
	for s.flowHasTasks {
		existingFlow := s.rootFlow

		s.flowHasTasks = false
		s.rootFlow = gotaskflow.NewTaskFlow("bonk")

		s.executor.Run(existingFlow).Wait()
	}
}
