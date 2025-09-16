// Copyright © 2025 Colden Cullen
// SPDX-License-Identifier: MIT

// basic provides a driver using default implementations of the basic bonk services.
package basic

import (
	"context"

	"go.uber.org/multierr"

	"go.bonk.build/pkg/driver"
	"go.bonk.build/pkg/executor/plugin"
	"go.bonk.build/pkg/executor/statecheck"
	"go.bonk.build/pkg/scheduler"
	"go.bonk.build/pkg/task"
)

type basicDriver struct {
	plugin.PluginClientManager
	scheduler.Scheduler

	openSessions []task.Session
}

var _ driver.Driver = (*basicDriver)(nil)

func New(ctx context.Context, options ...driver.DriverOption) (driver.Driver, error) {
	result := &basicDriver{
		PluginClientManager: plugin.NewPluginClientManager(),
	}
	result.Scheduler = scheduler.NewScheduler(
		statecheck.New(result.PluginClientManager),
		100, //nolint:mnd
	)

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

func (drv *basicDriver) NewLocalSession(
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

func (drv *basicDriver) Shutdown(ctx context.Context) {
	for _, session := range drv.openSessions {
		drv.CloseSession(ctx, session.ID())
	}

	drv.PluginClientManager.Shutdown(ctx)
}
