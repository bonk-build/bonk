// Copyright © 2025 Colden Cullen
// SPDX-License-Identifier: MIT

package scheduler // import "go.bonk.build/pkg/scheduler"

import (
	"context"

	"go.bonk.build/pkg/task"
)

type Scheduler interface {
	AddTask(ctx context.Context, tsk *task.GenericTask, deps ...string) error
	Run()
}
