// Copyright © 2025 Colden Cullen
// SPDX-License-Identifier: MIT

// Package router provides [Router], which is meant to route tasks to child executors.
package router

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"sync"

	"go.uber.org/multierr"

	"go.bonk.build/pkg/executor"
	"go.bonk.build/pkg/task"
)

// Router is a tree of Executors. It is meant to route tasks to child executors,
// and can be used for branching executor trees.
type Router struct {
	children map[string]executor.Executor
	mu       sync.RWMutex
}

const Wildcard = "*"

var (
	// Note that Router is itself an Executor.
	_ executor.Executor = (*Router)(nil)

	ErrDuplicateExecutor = errors.New("duplicate executor name")
	ErrNoExecutorFound   = errors.New("no executor found")
)

func New() Router {
	return Router{
		children: make(map[string]executor.Executor),
	}
}

func (r *Router) RegisterExecutor(name string, exec executor.Executor) error {
	r.mu.Lock()
	defer r.mu.Unlock()

	before, after, needsSubRouter := strings.Cut(name, task.TaskIDSep)
	child, hasChild := r.children[before]

	switch {
	// Needs & has subrouter, just recurse
	case needsSubRouter && hasChild:
		childRouter, ok := child.(*Router)
		if !ok {
			// If there's a child that isn't a router, replace it with a router and recurse.
			childRouter = &Router{
				children: make(map[string]executor.Executor, 1),
			}
			r.children[before] = childRouter
			// Re-register the old child as a nameless.
			err := childRouter.RegisterExecutor("", child)
			if err != nil {
				panic(err)
			}
		}

		return childRouter.RegisterExecutor(after, exec)

	// Needs & doesn't have router, add router and rerecurse
	case needsSubRouter && !hasChild:
		childRouter := &Router{
			children: make(map[string]executor.Executor, 1),
		}
		r.children[before] = childRouter

		return childRouter.RegisterExecutor(after, exec)

	// Doesn't need more routers but already has a child, error
	case !needsSubRouter && hasChild:
		if childRouter, ok := child.(*Router); ok {
			return childRouter.RegisterExecutor(after, exec)
		}

		return fmt.Errorf("%w: %s", ErrDuplicateExecutor, before)

	// Best case, doesn't need more routers, just register
	case !needsSubRouter && !hasChild:
		r.children[before] = exec

		return nil

	default:
		panic("unreachable")
	}
}

func (r *Router) UnregisterExecutors(names ...string) {
	r.mu.Lock()
	defer r.mu.Unlock()

	for _, name := range names {
		before, after, _ := strings.Cut(name, task.TaskIDSep)
		child, ok := r.children[before]
		if !ok {
			return
		}

		if child, ok := child.(*Router); ok {
			child.UnregisterExecutors(after)

			// If children remain, return so as to not remove
			if len(child.children) > 0 {
				return
			}
		}

		// Remove the child
		delete(r.children, name)
	}
}

func (r *Router) OpenSession(ctx context.Context, session task.Session) error {
	r.mu.RLock()
	defer r.mu.RUnlock()

	var err error
	r.ForEachExecutor(func(_ string, exec executor.Executor) {
		multierr.AppendInto(&err, exec.OpenSession(ctx, session))
	})

	return err
}

func (r *Router) CloseSession(ctx context.Context, sessionId task.SessionID) {
	r.mu.RLock()
	defer r.mu.RUnlock()

	r.ForEachExecutor(func(_ string, exec executor.Executor) {
		exec.CloseSession(ctx, sessionId)
	})
}

func (r *Router) Execute(
	ctx context.Context,
	session task.Session,
	tsk *task.Task,
	result *task.Result,
) error {
	r.mu.RLock()
	defer r.mu.RUnlock()

	exec := tsk.Executor
	before, after, _ := strings.Cut(exec, task.TaskIDSep)

	// Check for the original key and the wildcard key.
	for _, searchKey := range []string{before, "*"} {
		if child, ok := r.children[searchKey]; ok {
			tsk.Executor = after
			err := child.Execute(ctx, session, tsk, result)
			tsk.Executor = exec

			return err
		}
	}

	if fallback, ok := r.children[""]; ok {
		return fallback.Execute(ctx, session, tsk, result)
	}

	return fmt.Errorf("%w: %s", ErrNoExecutorFound, tsk.Executor)
}

func (r *Router) GetNumExecutors() int {
	result := 0
	r.ForEachExecutor(func(string, executor.Executor) {
		result++
	})

	return result
}

func (r *Router) ForEachExecutor(fun func(name string, exec executor.Executor)) {
	r.mu.RLock()
	defer r.mu.RUnlock()

	for name, exec := range r.children {
		forEachExecutorImpl(name, exec, fun)
	}
}

func forEachExecutorImpl(
	workingName string,
	exec executor.Executor,
	fun func(string, executor.Executor),
) {
	switch exec := exec.(type) {
	case *Router:
		for name, child := range exec.children {
			if name == "" {
				name = workingName
			} else {
				name = strings.Join([]string{workingName, name}, task.TaskIDSep)
			}

			forEachExecutorImpl(name, child, fun)
		}

	default:
		fun(workingName, exec)
	}
}
