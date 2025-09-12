// Copyright © 2025 Colden Cullen
// SPDX-License-Identifier: MIT

package task

import "context"

//go:generate go tool mockgen -destination executor_mock.go -package task -copyright_file ../../license-header.txt -typed . Executor

// Executor is the interface required to execute tasks.
type Executor[Params any] interface {
	Execute(ctx context.Context, tsk *Task[Params], result *Result) error
	OpenSession(ctx context.Context, session Session) error
	CloseSession(ctx context.Context, sessionId SessionId)
}

type GenericExecutor = Executor[any]

// Can be embedded if session management isn't necessary.
type NoopSessionManager struct{}

func (NoopSessionManager) OpenSession(ctx context.Context, session Session) error { return nil }
func (NoopSessionManager) CloseSession(ctx context.Context, sessionId SessionId)  {}
