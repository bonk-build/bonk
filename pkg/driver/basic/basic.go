// Copyright © 2025 Colden Cullen
// SPDX-License-Identifier: MIT

// basic provides a driver using default implementations of the basic bonk services.
package basic

import (
	"context"
	"errors"

	"go.uber.org/multierr"

	"go.bonk.build/pkg/executor/plugin"
	"go.bonk.build/pkg/scheduler"
	"go.bonk.build/pkg/task"
)

type BasicDriver struct {
	plugin.PluginClientManager
	scheduler.Scheduler

	openSessions []task.Session
}

type BasicDriverOption = func(context.Context, *BasicDriver) error

func WithPlugins(plugins ...string) BasicDriverOption {
	return func(ctx context.Context, drv *BasicDriver) error {
		return drv.StartPlugins(ctx, plugins...)
	}
}

func WithLocalSession(path string) BasicDriverOption {
	return func(ctx context.Context, drv *BasicDriver) error {
		_, err := drv.NewLocalSession(ctx, path)

		return err
	}
}

func WithTask[Params any](
	executor, name string,
	args Params,
	inputs ...string,
) BasicDriverOption {
	return func(ctx context.Context, drv *BasicDriver) error {
		if len(drv.openSessions) == 0 {
			return errors.New("cannot schedule task without an open session")
		}

		return drv.AddTask(
			ctx,
			task.New(
				drv.openSessions[len(drv.openSessions)-1],
				executor,
				name,
				args,
				inputs...,
			).Box(),
		)
	}
}

func New(ctx context.Context, options ...BasicDriverOption) (*BasicDriver, error) {
	result := &BasicDriver{
		PluginClientManager: *plugin.NewPluginClientManager(),
	}
	result.Scheduler = *scheduler.NewScheduler(&result.PluginClientManager, 100) //nolint:mnd

	var err error
	for _, option := range options {
		multierr.AppendInto(&err, option(ctx, result))
	}
	if err != nil {
		result.Shutdown(ctx)

		return nil, err
	}

	return result, nil
}

func (drv *BasicDriver) NewLocalSession(
	ctx context.Context,
	path string,
) (task.LocalSession, error) {
	session := task.NewLocalSession(task.NewSessionId(), path)

	err := drv.OpenSession(ctx, session)
	if err != nil {
		return nil, err //nolint:wrapcheck
	}

	drv.openSessions = append(drv.openSessions, session)

	return session, nil
}

func (drv *BasicDriver) Shutdown(ctx context.Context) {
	for _, session := range drv.openSessions {
		drv.CloseSession(ctx, session.ID())
	}

	drv.PluginClientManager.Shutdown()
}
