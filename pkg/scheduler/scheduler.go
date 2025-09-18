// Copyright © 2025 Colden Cullen
// SPDX-License-Identifier: MIT

package scheduler

import (
	"context"

	"go.bonk.build/pkg/task"
)

type Scheduler interface {
	AddTask(ctx context.Context, tsk *task.Task, deps ...string) error
	Run()
}

type SchedulerFactory func(context.Context, task.Executor) Scheduler
