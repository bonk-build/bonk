// Copyright © 2025 Colden Cullen
// SPDX-License-Identifier: MIT

package tree // import "go.bonk.build/pkg/executor/tree"

import (
	"context"
	"errors"
	"fmt"
	"strings"

	"go.uber.org/multierr"

	"go.bonk.build/pkg/task"
)

// ExecutorManager is a tree of Executors.
type ExecutorManager struct {
	children map[string]task.GenericExecutor
}

// Note that ExecutorManager is itself an executor.
var (
	_ task.GenericExecutor = (*ExecutorManager)(nil)
)
var ErrNoExecutorFound = errors.New("no executor found")

const ExecPathSep = "."

func NewExecutorManager() ExecutorManager {
	return ExecutorManager{
		children: make(map[string]task.GenericExecutor),
	}
}

func (bm *ExecutorManager) RegisterExecutor(name string, exec task.GenericExecutor) error {
	var registerImpl func(manager *ExecutorManager, name string, impl task.GenericExecutor) error
	registerImpl = func(manager *ExecutorManager, name string, impl task.GenericExecutor) error {
		before, after, needsManager := strings.Cut(name, ExecPathSep)
		child, hasChild := manager.children[before]

		switch {
		// Needs & has manager, just recurse
		case needsManager && hasChild:
			childManager, ok := child.(*ExecutorManager)
			if !ok {
				return fmt.Errorf("duplicate executor name: %s", before)
			}

			return registerImpl(childManager, after, impl)

		// Needs & doesn't have manager, add manager and retry
		case needsManager && !hasChild:
			manager.children[before] = &ExecutorManager{
				children: make(map[string]task.GenericExecutor, 1),
			}

			return registerImpl(manager, name, impl)

		// Doesn't need more manager tree but already has a child, error
		case !needsManager && hasChild:
			return fmt.Errorf("duplicate executor name: %s", before)

		// Best case, doesn't need more tree, just register
		case !needsManager && !hasChild:
			manager.children[before] = impl

			return nil

		default:
			panic("unreachable")
		}
	}

	return registerImpl(bm, name, exec)
}

func (bm *ExecutorManager) UnregisterExecutors(names ...string) {
	var unregisterImpl func(manager *ExecutorManager, name string)
	unregisterImpl = func(manager *ExecutorManager, name string) {
		before, after, hasChild := strings.Cut(name, ExecPathSep)
		child, ok := manager.children[before]

		switch {
		case !ok:
			return

		case hasChild:
			if childManager, ok := child.(*ExecutorManager); ok {
				unregisterImpl(childManager, after)
			}

		default:
			delete(manager.children, name)
		}
	}

	for _, name := range names {
		unregisterImpl(bm, name)
	}
}

func (bm *ExecutorManager) OpenSession(ctx context.Context, session task.Session) error {
	var err error
	bm.ForEachExecutor(func(_ string, exec task.GenericExecutor) {
		multierr.AppendInto(&err, exec.OpenSession(ctx, session))
	})

	return err
}

func (bm *ExecutorManager) CloseSession(ctx context.Context, sessionId task.SessionId) {
	bm.ForEachExecutor(func(_ string, exec task.GenericExecutor) {
		exec.CloseSession(ctx, sessionId)
	})
}

func (bm *ExecutorManager) Execute(
	ctx context.Context,
	tsk *task.GenericTask,
	result *task.Result,
) error {
	exec := tsk.ID.Executor
	before, after, _ := strings.Cut(exec, ExecPathSep)
	child, ok := bm.children[before]

	if ok {
		tsk.ID.Executor = after
		err := child.Execute(ctx, tsk, result)
		tsk.ID.Executor = exec

		return err //nolint:wrapcheck
	} else {
		return fmt.Errorf("%w: %s", ErrNoExecutorFound, tsk.ID.Executor)
	}
}

func (bm *ExecutorManager) GetNumExecutors() int {
	result := 0
	bm.ForEachExecutor(func(string, task.GenericExecutor) {
		result++
	})

	return result
}

func (bm *ExecutorManager) ForEachExecutor(fun func(name string, exec task.GenericExecutor)) {
	var forEachImpl func(name string, appendName bool, child task.GenericExecutor)
	forEachImpl = func(name string, appendName bool, child task.GenericExecutor) {
		if childManager, ok := child.(*ExecutorManager); ok {
			for childName, childExec := range childManager.children {
				var pathParts []string
				if appendName && childName != "" {
					pathParts = []string{name, childName}
				} else {
					pathParts = []string{childName}
				}

				forEachImpl(strings.Join(pathParts, ExecPathSep), true, childExec)
			}
		} else {
			fun(name, child)
		}
	}

	forEachImpl("", false, bm)
}

func (bm *ExecutorManager) Shutdown() {
	bm.children = make(map[string]task.GenericExecutor)
}
