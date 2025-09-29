// Copyright © 2025 Colden Cullen
// SPDX-License-Identifier: MIT

// Package driver provides an interface for defining a driver to perform a bonk pipeline's work.
package driver

import (
	"fmt"

	"go.uber.org/multierr"

	context "context"

	"go.bonk.build/pkg/executor/observable"
	"go.bonk.build/pkg/executor/plugin"
	"go.bonk.build/pkg/executor/scheduler"
	"go.bonk.build/pkg/executor/statecheck"
	"go.bonk.build/pkg/task"
)

func Run(ctx context.Context, options ...Option) error {
	option := MakeDefaultOptions()

	for _, opt := range options {
		opt(&option)
	}

	pcm := plugin.NewPluginClientManager()
	err := pcm.StartPlugins(ctx, option.Plugins...)
	if err != nil {
		return fmt.Errorf("failed to initialize plugins: %w", err)
	}

	for name, exec := range option.Executors {
		multierr.AppendInto(&err, pcm.RegisterExecutor(name, exec))
	}
	if err != nil {
		return fmt.Errorf("failed to register executors: %w", err)
	}

	// This is the root of the executable tree
	var exec task.Executor = pcm

	// Wrap the pcm in common executors
	exec = statecheck.New(exec)

	if len(option.Observers) > 0 {
		obs := observable.New(exec)

		for _, observer := range option.Observers {
			multierr.AppendInto(&err, obs.Listen(observer))
		}
		if err != nil {
			return fmt.Errorf("failed to register observers: %w", err)
		}

		exec = obs
	}

	exec = scheduler.New(exec)

	for session, tasks := range option.Sessions {
		multierr.AppendInto(&err, exec.OpenSession(ctx, session))

		for _, tsk := range tasks {
			res := task.Result{}
			multierr.AppendInto(&err, exec.Execute(ctx, tsk, &res))
		}
	}

	if err != nil {
		return fmt.Errorf("failed to register tasks with scheduler: %w", err)
	}

	for session := range option.Sessions {
		exec.CloseSession(ctx, session.ID())
	}

	return nil
}
