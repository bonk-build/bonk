// Copyright © 2025 Colden Cullen
// SPDX-License-Identifier: MIT

package driver

import (
	"go.bonk.build/pkg/executor/argconv"
	"go.bonk.build/pkg/executor/observable"
	"go.bonk.build/pkg/task"
)

type Options struct {
	Concurrency uint
	Plugins     []string
	Executors   map[string]task.Executor
	Sessions    map[task.Session][]*task.Task
	Observers   []observable.Observer
}

func MakeDefaultOptions() Options {
	return Options{
		Plugins:   make([]string, 0, 3), //nolint:mnd
		Executors: make(map[string]task.Executor),
		Sessions:  make(map[task.Session][]*task.Task),
		Observers: make([]observable.Observer, 0),
	}
}

// Option is a functor for modifying a [Driver].
type Option = func(*Options)

func WithConcurrency(concurrency uint) Option {
	return func(opts *Options) {
		opts.Concurrency = concurrency
	}
}

// WithGenericExecutor registers the given generic executor.
func WithGenericExecutor(name string, exec task.Executor) Option {
	return func(opts *Options) {
		opts.Executors[name] = exec
	}
}

// WithExecutor registers the given executor.
func WithExecutor[Params any](name string, exec argconv.TypedExecutor[Params]) Option {
	return WithGenericExecutor(name, argconv.BoxExecutor(exec))
}

// WithPlugins loads the specified plugins.
func WithPlugins(plugins ...string) Option {
	return func(opts *Options) {
		opts.Plugins = append(opts.Plugins, plugins...)
	}
}

// SessionOption is a functor for modifying a [task.Session].
type SessionOption = func(*Options, task.Session)

// WithLocalSession creates a [task.LocalSession] with the given options.
func WithLocalSession(path string, options ...SessionOption) Option {
	return func(opts *Options) {
		sess := task.NewLocalSession(task.NewSessionID(), path)

		for _, option := range options {
			option(opts, sess)
		}
	}
}

// WithTask executes a task in the session.
func WithTask(
	id task.ID,
	executor string,
	args any,
	options ...task.Option,
) SessionOption {
	return func(opts *Options, session task.Session) {
		opts.Sessions[session] = append(opts.Sessions[session], task.New(
			id,
			session,
			executor,
			args,
			options...,
		))
	}
}

// WithObservers adds observers to the execution pipeline.
func WithObservers(observers ...observable.Observer) Option {
	return func(opts *Options) {
		opts.Observers = append(opts.Observers, observers...)
	}
}
