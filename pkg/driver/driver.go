// Copyright © 2025 Colden Cullen
// SPDX-License-Identifier: MIT

// Package driver provides an interface for defining a driver to perform a bonk pipeline's work.
package driver

import (
	"fmt"

	"go.uber.org/multierr"

	context "context"

	"go.bonk.build/pkg/executor"
	"go.bonk.build/pkg/executor/observable"
	"go.bonk.build/pkg/executor/plugin"
	"go.bonk.build/pkg/executor/scheduler"
	"go.bonk.build/pkg/executor/statecheck"
	"go.bonk.build/pkg/task"
)

func Run(ctx context.Context, result *task.Result, options Options) error {
	pcm := plugin.NewPluginClientManager()
	err := pcm.StartPlugins(ctx, options.Plugins...)
	if err != nil {
		return fmt.Errorf("failed to initialize plugins: %w", err)
	}

	for name, exec := range options.Executors {
		multierr.AppendInto(&err, pcm.RegisterExecutor(name, exec))
	}
	if err != nil {
		return fmt.Errorf("failed to register executors: %w", err)
	}

	// This is the root of the executable tree
	var exec executor.Executor = pcm

	// Wrap the pcm in common executors
	exec = statecheck.New(exec)

	if len(options.Observers) > 0 {
		obs := observable.New(exec)

		for _, observer := range options.Observers {
			multierr.AppendInto(&err, obs.Listen(observer))
		}
		if err != nil {
			return fmt.Errorf("failed to register observers: %w", err)
		}

		exec = obs
	}

	exec = scheduler.New(exec, options.Concurrency)

	for session, tasks := range options.Sessions {
		multierr.AppendInto(&err, exec.OpenSession(ctx, session))

		for _, tsk := range tasks {
			multierr.AppendInto(&err, exec.Execute(ctx, session, tsk, result))
		}
	}

	if err != nil {
		return fmt.Errorf("failed to register tasks with scheduler: %w", err)
	}

	for session := range options.Sessions {
		exec.CloseSession(ctx, session.ID())
	}

	return nil
}
