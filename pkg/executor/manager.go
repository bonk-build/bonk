// Copyright © 2025 Colden Cullen
// SPDX-License-Identifier: MIT

package executor // import "go.bonk.build/pkg/executor"

import (
	"context"
	"fmt"

	"go.uber.org/multierr"

	"github.com/google/uuid"

	"go.bonk.build/pkg/task"
)

type ExecutorManager struct {
	executors map[string]Executor
}

// Note that ExecutorManager is itself an executor.
var (
	_ Executor       = (*ExecutorManager)(nil)
	_ SessionManager = (*ExecutorManager)(nil)
)

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

func (bm *ExecutorManager) OpenSession(ctx context.Context, sessionId uuid.UUID) error {
	var err error
	bm.ForEachExecutor(func(_ string, exec Executor) {
		if sm, ok := exec.(SessionManager); ok {
			multierr.AppendInto(&err, sm.OpenSession(ctx, sessionId))
		}
	})

	return err
}

func (bm *ExecutorManager) CloseSession(ctx context.Context, sessionId uuid.UUID) {
	bm.ForEachExecutor(func(_ string, exec Executor) {
		if sm, ok := exec.(SessionManager); ok {
			sm.CloseSession(ctx, sessionId)
		}
	})
}

func (bm *ExecutorManager) Execute(
	ctx context.Context,
	tsk task.Task,
	result *task.Result,
) error {
	executorName := tsk.Executor()

	executor, ok := bm.executors[executorName]
	if !ok {
		return fmt.Errorf("Executor %s not found", executorName)
	}

	err := executor.Execute(ctx, tsk, result)
	if err != nil {
		return fmt.Errorf("failed to execute task: %w", err)
	}

	return nil
}

func (bm *ExecutorManager) GetNumExecutors() int {
	return len(bm.executors)
}

func (bm *ExecutorManager) ForEachExecutor(fun func(name string, exec Executor)) {
	for name, exec := range bm.executors {
		fun(name, exec)
	}
}

func (bm *ExecutorManager) Shutdown() {
	bm.executors = make(map[string]Executor)
}
