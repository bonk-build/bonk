// Copyright © 2025 Colden Cullen
// SPDX-License-Identifier: MIT

package task

import "context"

//go:generate go tool mockgen -destination executor_mock.go -package task -copyright_file ../../license-header.txt -typed -write_package_comment=false . Executor

// Executor is the interface required to execute tasks.
type Executor interface {
	Execute(ctx context.Context, tsk *Task, result *Result) error
	OpenSession(ctx context.Context, session Session) error
	CloseSession(ctx context.Context, sessionId SessionID)
}

// NoopSessionManager can be embedded if session management isn't necessary.
type NoopSessionManager struct{}

func (NoopSessionManager) OpenSession(ctx context.Context, session Session) error { return nil }
func (NoopSessionManager) CloseSession(ctx context.Context, sessionId SessionID)  {}
