// Copyright © 2025 Colden Cullen
// SPDX-License-Identifier: MIT

package driver

import (
	"context"
	"fmt"

	"go.uber.org/multierr"

	"go.bonk.build/pkg/executor/argconv"
	"go.bonk.build/pkg/task"
)

// Option is a functor for modifying a [Driver].
type Option = func(context.Context, Driver) error

// WithGenericExecutor registers the given generic executor.
func WithGenericExecutor(name string, exec task.Executor) Option {
	return func(_ context.Context, drv Driver) error {
		return drv.RegisterExecutor(name, exec)
	}
}

// WithExecutor registers the given executor.
func WithExecutor[Params any](name string, exec argconv.TypedExecutor[Params]) Option {
	return WithGenericExecutor(name, argconv.BoxExecutor(exec))
}

// WithPlugins loads the specified plugins.
func WithPlugins(plugins ...string) Option {
	return func(ctx context.Context, drv Driver) error {
		return drv.StartPlugins(ctx, plugins...)
	}
}

// SessionOption is a functor for modifying a [task.Session].
type SessionOption = func(context.Context, Driver, task.Session) error

// WithLocalSession creates a [task.LocalSession] with the given options.
func WithLocalSession(path string, options ...SessionOption) Option {
	return func(ctx context.Context, drv Driver) error {
		sess, err := drv.NewLocalSession(ctx, path)
		if err != nil {
			return err //nolint:wrapcheck
		}

		for _, option := range options {
			multierr.AppendInto(&err, option(ctx, drv, sess))
		}

		if err != nil {
			return fmt.Errorf("failed to initialize session with options: %w", err)
		}

		return nil
	}
}

// TaskOption is a functor for modifying a [task.Task].
type TaskOption = func(context.Context, *task.Task)

// WithTask executes a task in the session.
func WithTask(
	id task.ID,
	executor string,
	args any,
	options ...TaskOption,
) SessionOption {
	return func(ctx context.Context, drv Driver, session task.Session) error {
		tsk := task.New(
			id,
			session,
			executor,
			args,
		)
		for _, opt := range options {
			opt(ctx, tsk)
		}

		return drv.AddTask(ctx, tsk)
	}
}

// WithInputs appends the given input specifiers to the task.
func WithInputs(inputs ...string) TaskOption {
	return func(_ context.Context, tsk *task.Task) {
		tsk.WithInputs(inputs...)
	}
}
