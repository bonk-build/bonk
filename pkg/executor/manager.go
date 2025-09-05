// Copyright © 2025 Colden Cullen
// SPDX-License-Identifier: MIT

package executor // import "go.bonk.build/pkg/executor"

import (
	"context"
	"errors"
	"fmt"
	"strings"

	"go.uber.org/multierr"

	"github.com/google/uuid"

	"go.bonk.build/pkg/task"
)

// ExecutorManager is a tree of Executors.
type ExecutorManager struct {
	executors map[string]*ExecutorManager

	executor Executor
}

// Note that ExecutorManager is itself an executor.
var (
	_ Executor       = (*ExecutorManager)(nil)
	_ SessionManager = (*ExecutorManager)(nil)
)
var ErrNoExecutorFound = errors.New("no executor found")

const ExecPathSep = "."

func NewExecutorManager() ExecutorManager {
	return ExecutorManager{
		executors: make(map[string]*ExecutorManager),
	}
}

func (bm *ExecutorManager) RegisterExecutor(name string, impl Executor) error {
	// At leaf, just register the executor
	if name == "" {
		if bm.executor != nil {
			return fmt.Errorf("duplicate executor name: %s", name)
		}

		bm.executor = impl

		return nil
	}

	before, after, _ := strings.Cut(name, ExecPathSep)
	child, ok := bm.executors[before]

	if !ok {
		bm.executors[before] = &ExecutorManager{
			executors: make(map[string]*ExecutorManager),
		}
		child = bm.executors[before]
	}

	return child.RegisterExecutor(after, impl)
}

func (bm *ExecutorManager) UnregisterExecutor(name string) {
	// At leaf, just disengage the
	if name == "" {
		bm.executor = nil

		return
	}

	before, after, _ := strings.Cut(name, ExecPathSep)
	child, ok := bm.executors[before]

	if ok {
		child.UnregisterExecutor(after)

		// If the child node is now empty, remove it
		if child.executor == nil && len(child.executors) == 0 {
			delete(bm.executors, before)
		}
	}
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
	err := ErrNoExecutorFound

	before, after, _ := strings.Cut(tsk.Executor(), ExecPathSep)
	child, ok := bm.executors[before]

	shouldExec := !ok
	if ok {
		copyForChild := tsk
		copyForChild.ID.Executor = after
		err = child.Execute(ctx, copyForChild, result)

		// If the child didn't find an executor, try this node's
		if errors.Is(err, ErrNoExecutorFound) {
			shouldExec = true
		}
	}

	if shouldExec && bm.executor != nil {
		err = bm.executor.Execute(ctx, tsk, result)
		if err != nil {
			return fmt.Errorf("failed to execute task: %w", err)
		}

		return nil
	}

	return err //nolint:wrapcheck
}

func (bm *ExecutorManager) GetNumExecutors() int {
	return len(bm.executors)
}

func (bm *ExecutorManager) ForEachExecutor(fun func(name string, exec Executor)) {
	var processor func(name string, appendName bool, child *ExecutorManager)
	processor = func(name string, appendName bool, child *ExecutorManager) {
		if child.executor != nil {
			fun(name, child.executor)
		}
		for childName, childExec := range child.executors {
			var pathParts []string
			if appendName {
				pathParts = []string{name, childName}
			} else {
				pathParts = []string{childName}
			}

			processor(strings.Join(pathParts, ExecPathSep), true, childExec)
		}
	}
	processor("", false, bm)
}

func (bm *ExecutorManager) Shutdown() {
	bm.executor = nil
	bm.executors = make(map[string]*ExecutorManager)
}
