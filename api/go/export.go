// Copyright © 2025 Colden Cullen
// SPDX-License-Identifier: MIT

package bonk

import "go.bonk.build/pkg/task"

type (
	Task[Params any]     = task.Task[Params]
	GenericTask          = task.GenericTask
	Executor[Params any] = task.Executor[Params]
	GenericExecutor      = task.GenericExecutor
	Result               = task.Result

	NoopSessionManager = task.NoopSessionManager
)

func BoxExecutor[Params any](
	impl Executor[Params],
) GenericExecutor {
	return task.BoxExecutor(impl)
}
