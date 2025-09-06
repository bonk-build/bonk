// Copyright © 2025 Colden Cullen
// SPDX-License-Identifier: MIT

package scheduler // import "go.bonk.build/pkg/scheduler"

import (
	"context"
	"fmt"
	"log/slog"

	"go.uber.org/multierr"

	gotaskflow "github.com/noneback/go-taskflow"

	"go.bonk.build/pkg/task"
)

type TaskSender interface {
	Execute(ctx context.Context, tsk task.Task, result *task.Result) error
}

type Scheduler struct {
	executorManager TaskSender
	executor        gotaskflow.Executor
	tasks           map[string]*gotaskflow.Task

	flowHasTasks bool
	rootFlow     *gotaskflow.TaskFlow
}

func NewScheduler(executorManager TaskSender, concurrency uint) *Scheduler {
	return &Scheduler{
		executorManager: executorManager,
		executor:        gotaskflow.NewExecutor(concurrency),
		tasks:           make(map[string]*gotaskflow.Task),

		flowHasTasks: false,
		rootFlow:     gotaskflow.NewTaskFlow("bonk"),
	}
}

func (s *Scheduler) AddTask(tsk task.Task, deps ...string) error {
	taskName := tsk.ID.String()
	s.flowHasTasks = true
	newTask := s.rootFlow.NewTask(taskName, func() {
		mismatches := DetectStateMismatches(&tsk)
		if mismatches == nil {
			slog.Debug("states match, skipping task")

			return
		}

		slog.Debug("state mismatch, running task", "mismatches", mismatches)

		var result task.Result
		err := s.executorManager.Execute(context.Background(), tsk, &result)
		if err != nil {
			slog.Error("error executing task", "task", taskName, "executor", tsk.Executor(), "error", err)

			return
		}

		slog.Info("task succeeded, saving state", "task", taskName)

		var followupErr error
		for _, followup := range result.FollowupTasks {
			multierr.AppendInto(&followupErr, s.AddTask(followup))
		}
		if followupErr != nil {
			slog.Error("failed to schedule followup tasks", "task", taskName, "error", followupErr)
		}

		err = SaveState(&tsk, &result)
		if err != nil {
			slog.Error("failed to save task state", "task", taskName, "error", err)

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
