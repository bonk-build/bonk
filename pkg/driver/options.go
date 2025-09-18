// Copyright © 2025 Colden Cullen
// SPDX-License-Identifier: MIT

package driver

import (
	"context"
	"fmt"

	"go.uber.org/multierr"

	"go.bonk.build/pkg/task"
)

type DriverOption = func(context.Context, Driver) error

func WithExecutor[Params any](name string, exec task.Executor[Params]) DriverOption {
	return func(ctx context.Context, drv Driver) error {
		return drv.RegisterExecutor(name, task.BoxExecutor(exec))
	}
}

func WithPlugins(plugins ...string) DriverOption {
	return func(ctx context.Context, drv Driver) error {
		return drv.StartPlugins(ctx, plugins...)
	}
}

type SessionOption = func(context.Context, Driver, task.Session) error

func WithLocalSession(path string, options ...SessionOption) DriverOption {
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

type TaskOption = func(context.Context, *task.GenericTask)

func WithTask[Params any](
	id string,
	executor string,
	args Params,
	options ...TaskOption,
) SessionOption {
	return func(ctx context.Context, drv Driver, session task.Session) error {
		tsk := task.New(
			id,
			session,
			executor,
			args,
		).Box()
		for _, opt := range options {
			opt(ctx, tsk)
		}

		return drv.AddTask(ctx, tsk)
	}
}

func WithInputs(inputs ...string) TaskOption {
	return func(_ context.Context, tsk *task.GenericTask) {
		tsk.WithInputs(inputs...)
	}
}
