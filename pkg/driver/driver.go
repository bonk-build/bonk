// Copyright © 2025 Colden Cullen
// SPDX-License-Identifier: MIT

package driver

import (
	"context"
	"fmt"

	"go.uber.org/multierr"

	"go.bonk.build/pkg/executor/plugin"
	"go.bonk.build/pkg/scheduler"
	"go.bonk.build/pkg/task"
)

type Driver interface {
	plugin.PluginClientManager
	scheduler.Scheduler

	NewLocalSession(
		ctx context.Context,
		path string,
	) (task.LocalSession, error)

	Shutdown(ctx context.Context)
}

type DriverOption = func(context.Context, Driver) error

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

func WithTask[Params any](
	executor, name string,
	args Params,
	inputs ...string,
) SessionOption {
	return func(ctx context.Context, drv Driver, session task.Session) error {
		return drv.AddTask(
			ctx,
			task.New(
				session,
				executor,
				name,
				args,
				inputs...,
			).Box(),
		)
	}
}
