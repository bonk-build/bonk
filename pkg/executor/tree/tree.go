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

// ExecutorTree is a tree of Executors.
type ExecutorTree struct {
	children map[string]task.GenericExecutor
}

// Note that ExecutorTree is itself an executor.
var (
	_ task.GenericExecutor = (*ExecutorTree)(nil)
)
var ErrNoExecutorFound = errors.New("no executor found")

const ExecPathSep = "."

func New() ExecutorTree {
	return ExecutorTree{
		children: make(map[string]task.GenericExecutor),
	}
}

func (et *ExecutorTree) RegisterExecutor(name string, exec task.GenericExecutor) error {
	var registerImpl func(manager *ExecutorTree, name string, impl task.GenericExecutor) error
	registerImpl = func(manager *ExecutorTree, name string, impl task.GenericExecutor) error {
		before, after, needsManager := strings.Cut(name, ExecPathSep)
		child, hasChild := manager.children[before]

		switch {
		// Needs & has manager, just recurse
		case needsManager && hasChild:
			childManager, ok := child.(*ExecutorTree)
			if !ok {
				return fmt.Errorf("duplicate executor name: %s", before)
			}

			return registerImpl(childManager, after, impl)

		// Needs & doesn't have manager, add manager and retry
		case needsManager && !hasChild:
			manager.children[before] = &ExecutorTree{
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

	return registerImpl(et, name, exec)
}

func (et *ExecutorTree) UnregisterExecutors(names ...string) {
	var unregisterImpl func(manager *ExecutorTree, name string)
	unregisterImpl = func(manager *ExecutorTree, name string) {
		before, after, hasChild := strings.Cut(name, ExecPathSep)
		child, ok := manager.children[before]

		switch {
		case !ok:
			return

		case hasChild:
			if childManager, ok := child.(*ExecutorTree); ok {
				unregisterImpl(childManager, after)
			}

		default:
			delete(manager.children, name)
		}
	}

	for _, name := range names {
		unregisterImpl(et, name)
	}
}

func (et *ExecutorTree) OpenSession(ctx context.Context, session task.Session) error {
	var err error
	et.ForEachExecutor(func(_ string, exec task.GenericExecutor) {
		multierr.AppendInto(&err, exec.OpenSession(ctx, session))
	})

	return err
}

func (et *ExecutorTree) CloseSession(ctx context.Context, sessionId task.SessionId) {
	et.ForEachExecutor(func(_ string, exec task.GenericExecutor) {
		exec.CloseSession(ctx, sessionId)
	})
}

func (et *ExecutorTree) Execute(
	ctx context.Context,
	tsk *task.GenericTask,
	result *task.Result,
) error {
	exec := tsk.ID.Executor
	before, after, _ := strings.Cut(exec, ExecPathSep)
	child, ok := et.children[before]

	if ok {
		tsk.ID.Executor = after
		err := child.Execute(ctx, tsk, result)
		tsk.ID.Executor = exec

		return err //nolint:wrapcheck
	} else {
		return fmt.Errorf("%w: %s", ErrNoExecutorFound, tsk.ID.Executor)
	}
}

func (et *ExecutorTree) GetNumExecutors() int {
	result := 0
	et.ForEachExecutor(func(string, task.GenericExecutor) {
		result++
	})

	return result
}

func (et *ExecutorTree) ForEachExecutor(fun func(name string, exec task.GenericExecutor)) {
	var forEachImpl func(name string, appendName bool, child task.GenericExecutor)
	forEachImpl = func(name string, appendName bool, child task.GenericExecutor) {
		if childManager, ok := child.(*ExecutorTree); ok {
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

	forEachImpl("", false, et)
}

func (et *ExecutorTree) Shutdown() {
	et.children = make(map[string]task.GenericExecutor)
}
