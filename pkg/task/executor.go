// Copyright © 2025 Colden Cullen
// SPDX-License-Identifier: MIT

package task

import "context"

//go:generate go tool mockgen -destination executor_mock.go -package task -copyright_file ../../license-header.txt -typed -write_package_comment=false . Executor

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

// CloseSession implements Executor.
func (n NoopSessionManager) CloseSession(context.Context, SessionID) {}
