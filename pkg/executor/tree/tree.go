// Copyright © 2025 Colden Cullen
// SPDX-License-Identifier: MIT

// Package tree provides [ExecutorTree], which is meant to route tasks to child executors.
package tree

import (
	"context"
	"errors"
	"fmt"
	"strings"

	"go.uber.org/multierr"

	"go.bonk.build/pkg/executor"
	"go.bonk.build/pkg/task"
)

// ExecutorTree is a tree of Executors. It is meant to route tasks to child executors,
// and can be used for branching executor trees.
type ExecutorTree struct {
	children map[string]executor.Executor
}

// Note that ExecutorTree is itself an executor.
var _ executor.Executor = (*ExecutorTree)(nil)

var (
	ErrDuplicateExecutor = errors.New("duplicate executor name")
	ErrNoExecutorFound   = errors.New("no executor found")
)

func New() ExecutorTree {
	return ExecutorTree{
		children: make(map[string]executor.Executor),
	}
}

func (et *ExecutorTree) RegisterExecutor(name string, exec executor.Executor) error {
	before, after, needsManager := strings.Cut(name, task.TaskIDSep)
	child, hasChild := et.children[before]

	switch {
	// Needs & has manager, just recurse
	case needsManager && hasChild:
		childManager, ok := child.(*ExecutorTree)
		if !ok {
			// If there's a child that isn't a manager, replace it with a manager and recurse.
			childManager = &ExecutorTree{
				make(map[string]executor.Executor, 1),
			}
			et.children[before] = childManager
			// Re-register the old child as a nameless.
			err := childManager.RegisterExecutor("", child)
			if err != nil {
				return err
			}
		}

		return childManager.RegisterExecutor(after, exec)

	// Needs & doesn't have manager, add manager and retry
	case needsManager && !hasChild:
		et.children[before] = &ExecutorTree{
			children: make(map[string]executor.Executor, 1),
		}

		return et.RegisterExecutor(name, exec)

	// Doesn't need more manager tree but already has a child, error
	case !needsManager && hasChild:
		if childManager, ok := child.(*ExecutorTree); ok {
			return childManager.RegisterExecutor(after, exec)
		}

		return fmt.Errorf("%w: %s", ErrDuplicateExecutor, before)

	// Best case, doesn't need more tree, just register
	case !needsManager && !hasChild:
		et.children[before] = exec

		return nil

	default:
		panic("unreachable")
	}
}

func (et *ExecutorTree) UnregisterExecutors(names ...string) {
	var unregisterImpl func(manager *ExecutorTree, name string)
	unregisterImpl = func(manager *ExecutorTree, name string) {
		before, after, hasChild := strings.Cut(name, task.TaskIDSep)
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
	et.ForEachExecutor(func(_ string, exec executor.Executor) {
		multierr.AppendInto(&err, exec.OpenSession(ctx, session))
	})

	return err
}

func (et *ExecutorTree) CloseSession(ctx context.Context, sessionId task.SessionID) {
	et.ForEachExecutor(func(_ string, exec executor.Executor) {
		exec.CloseSession(ctx, sessionId)
	})
}

func (et *ExecutorTree) Execute(
	ctx context.Context,
	session task.Session,
	tsk *task.Task,
	result *task.Result,
) error {
	exec := tsk.Executor
	before, after, _ := strings.Cut(exec, task.TaskIDSep)

	// Check for the original key and the wildcard key.
	for _, searchKey := range []string{before, "*"} {
		if child, ok := et.children[searchKey]; ok {
			tsk.Executor = after
			err := child.Execute(ctx, session, tsk, result)
			tsk.Executor = exec

			return err
		}
	}

	return fmt.Errorf("%w: %s", ErrNoExecutorFound, tsk.Executor)
}

func (et *ExecutorTree) GetNumExecutors() int {
	result := 0
	et.ForEachExecutor(func(string, executor.Executor) {
		result++
	})

	return result
}

func (et *ExecutorTree) ForEachExecutor(fun func(name string, exec executor.Executor)) {
	var forEachImpl func(name string, appendName bool, child executor.Executor)
	forEachImpl = func(name string, appendName bool, child executor.Executor) {
		if childManager, ok := child.(*ExecutorTree); ok {
			for childName, childExec := range childManager.children {
				var pathParts []string
				if appendName && childName != "" {
					pathParts = []string{name, childName}
				} else {
					pathParts = []string{childName}
				}

				forEachImpl(strings.Join(pathParts, task.TaskIDSep), true, childExec)
			}
		} else {
			fun(name, child)
		}
	}

	forEachImpl("", false, et)
}
