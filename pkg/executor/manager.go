// Copyright © 2025 Colden Cullen
// SPDX-License-Identifier: MIT

package executor // import "go.bonk.build/pkg/executor"

import (
	"context"
	"fmt"

	"go.bonk.build/pkg/task"
)

type ExecutorManager struct {
	executors map[string]Executor
}

// Note that ExecutorManager is itself an executor.
var _ Executor = (*ExecutorManager)(nil)

func NewExecutorManager() ExecutorManager {
	bm := ExecutorManager{}
	bm.executors = make(map[string]Executor)

	return bm
}

func (bm *ExecutorManager) RegisterExecutor(name string, impl Executor) error {
	_, ok := bm.executors[name]
	if ok {
		return fmt.Errorf("duplicate executor name: %s", name)
	}

	bm.executors[name] = impl

	return nil
}

func (bm *ExecutorManager) UnregisterExecutor(name string) {
	delete(bm.executors, name)
}

func (bm *ExecutorManager) Execute(ctx context.Context, tsk task.Task) (*task.TaskResult, error) {
	executorName := tsk.Executor()

	executor, ok := bm.executors[executorName]
	if !ok {
		return nil, fmt.Errorf("Executor %s not found", executorName)
	}

	result, err := executor.Execute(ctx, tsk)
	if err != nil {
		return nil, fmt.Errorf("failed to execute task: %w", err)
	}

	return result, nil
}

func (bm *ExecutorManager) Shutdown() {
	bm.executors = make(map[string]Executor)
}
