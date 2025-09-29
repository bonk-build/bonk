// Copyright © 2025 Colden Cullen
// SPDX-License-Identifier: MIT

package task

import "context"

// Executor is the interface required to execute tasks.
type Executor interface {
	Execute(ctx context.Context, tsk *Task, result *Result) error
	OpenSession(ctx context.Context, session Session) error
	CloseSession(ctx context.Context, sessionID SessionID)
}

// NoopSessionManager can be embedded if session management isn't necessary.
type NoopSessionManager struct{}

// OpenSession implements Executor.
func (n NoopSessionManager) OpenSession(context.Context, Session) error { return nil }

// CloseSession should shutdown and free all resources created over the course of a session.
// After this call, no outstanding goroutines should be running.
func (n NoopSessionManager) CloseSession(context.Context, SessionID) {}
