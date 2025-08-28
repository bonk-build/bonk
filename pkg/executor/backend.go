// Copyright © 2025 Colden Cullen
// SPDX-License-Identifier: MIT

package executor // import "go.bonk.build/pkg/executor"

import (
	"context"

	"go.bonk.build/pkg/task"
)

type Executor interface {
	Execute(ctx context.Context, tsk task.Task) (*task.TaskResult, error)
}
