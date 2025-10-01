// Copyright © 2025 Colden Cullen
// SPDX-License-Identifier: MIT

// Package executor provides the [Executor] interface, which is an abstraction for "object which may execute a task."
//
// Subpackages provide useful executors for building task-executing heirarchies.
// Each package exports just a few objects conforming to the [Executor] interface,
// and optionally a few helpers.
package executor

import (
	"context"

	"go.bonk.build/pkg/task"
)

// Executor is the interface required to execute tasks.
type Executor interface {
	// Execute is given a task to execute and expected to populate result with the outcome.
	Execute(ctx context.Context, tsk *task.Task, result *task.Result) error
	// OpenSession is called before any tasks are executed, and can be used to do things such as
	// initializing caches, etc.
	OpenSession(ctx context.Context, session task.Session) error
	// CloseSession shuts down and frees all resources created over the course of a session.
	// After this call, no outstanding goroutines should be running.
	CloseSession(ctx context.Context, sessionID task.SessionID)
}

// NoopSessionManager can be embedded if session management isn't necessary.
type NoopSessionManager struct{}

// OpenSession implements Executor.
func (n NoopSessionManager) OpenSession(context.Context, task.Session) error { return nil }

// CloseSession implements Executor.
func (n NoopSessionManager) CloseSession(context.Context, task.SessionID) {}
