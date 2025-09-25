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
	"go.bonk.build/pkg/executor/statecheck"
	"go.bonk.build/pkg/scheduler/taskflow"
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

	// Wrap the pcm in common executors
	exec := statecheck.New(pcm)
	exec = observable.New(exec)

	sched := taskflow.New(option.Concurrency)(ctx, exec)

	for session, tasks := range option.Sessions {
		multierr.AppendInto(&err, exec.OpenSession(ctx, session))

		for _, tsk := range tasks {
			multierr.AppendInto(&err, sched.AddTask(ctx, tsk))
		}
	}

	if err != nil {
		return fmt.Errorf("failed to register tasks with scheduler: %w", err)
	}

	sched.Run()

	for session := range option.Sessions {
		exec.CloseSession(ctx, session.ID())
	}

	return nil
}
