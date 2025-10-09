// Copyright © 2025 Colden Cullen
// SPDX-License-Identifier: MIT

package driver

import (
	"go.bonk.build/pkg/executor"
	"go.bonk.build/pkg/executor/observable"
	"go.bonk.build/pkg/task"
)

type Options struct {
	Concurrency int
	Plugins     []string
	Executors   map[string]executor.Executor
	Sessions    map[task.Session][]*task.Task
	Observers   []observable.Observer
}

func MakeDefaultOptions() Options {
	return Options{
		Plugins:   make([]string, 0, 3), //nolint:mnd
		Executors: make(map[string]executor.Executor),
		Sessions:  make(map[task.Session][]*task.Task),
		Observers: make([]observable.Observer, 0),
	}
}

func (opts Options) WithConcurrency(concurrency int) Options {
	opts.Concurrency = concurrency

	return opts
}

// WithExecutor registers the given executor.
func (opts Options) WithExecutor(name string, exec executor.Executor) Options {
	opts.Executors[name] = exec

	return opts
}

// WithPlugins loads the specified plugins.
func (opts Options) WithPlugins(plugins ...string) Options {
	opts.Plugins = append(opts.Plugins, plugins...)

	return opts
}

// SessionOption is a functor for modifying a [task.Session].
type SessionOption = func(Options, task.Session)

// WithLocalSession creates a [task.LocalSession] with the given options.
func (opts Options) WithLocalSession(path string, options ...SessionOption) Options {
	sess := task.NewLocalSession(task.NewSessionID(), path)

	for _, option := range options {
		option(opts, sess)
	}

	return opts
}

// WithTask executes a task in the session.
func WithTask(
	id task.ID,
	executor string,
	args any,
	options ...task.Option,
) SessionOption {
	return func(opts Options, session task.Session) {
		opts.Sessions[session] = append(opts.Sessions[session], task.New(
			id,
			executor,
			args,
			options...,
		))
	}
}

// WithObservers adds observers to the execution pipeline.
func (opts Options) WithObservers(observers ...observable.Observer) Options {
	opts.Observers = append(opts.Observers, observers...)

	return opts
}
