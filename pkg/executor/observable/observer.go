// Copyright © 2025 Colden Cullen
// SPDX-License-Identifier: MIT

// Package observable provides an executor which alerts followers to task status
package observable

import (
	"context"
	"errors"
	"fmt"
	"sync"

	"go.bonk.build/pkg/task"
)

type Observer = func(TaskStatusMsg)

type Observable interface {
	task.Executor

	Listen(f Observer) error
}

var ErrUnopenedSession = errors.New("task being executed for unopened session")

func New(exec task.Executor) Observable {
	return &observ{
		exec:      exec,
		sessions:  make(map[task.SessionID]*observSession, 1),
		listeners: make([]Observer, 0),
	}
}

type observSession struct {
	waiter sync.WaitGroup
}

type observ struct {
	exec     task.Executor
	sessions map[task.SessionID]*observSession

	listeners []Observer
}

func (obs *observ) Execute(ctx context.Context, tsk *task.Task, result *task.Result) error {
	session, ok := obs.sessions[tsk.Session.ID()]
	if !ok {
		return fmt.Errorf("%w: %s", ErrUnopenedSession, tsk.Session.ID())
	}

	obs.trigger(session, TaskRunningMsg(tsk.ID))

	err := obs.exec.Execute(ctx, tsk, result)

	obs.trigger(session, TaskFinishedMsg(tsk.ID, err))

	return err //nolint:wrapcheck
}

// OpenSession implements Observable.
func (obs *observ) OpenSession(ctx context.Context, session task.Session) error {
	obs.sessions[session.ID()] = &observSession{}

	return obs.exec.OpenSession(ctx, session) //nolint:wrapcheck
}

// CloseSession implements Observable.
func (obs *observ) CloseSession(ctx context.Context, sessionID task.SessionID) {
	// Block until outstanding tasks are done
	obs.sessions[sessionID].waiter.Wait()
	// Remove the session from the map
	delete(obs.sessions, sessionID)

	obs.exec.CloseSession(ctx, sessionID)
}

// Listen implements Observable.
func (obs *observ) Listen(f Observer) error {
	obs.listeners = append(obs.listeners, f)

	return nil
}

func (obs *observ) trigger(session *observSession, tsm TaskStatusMsg) {
	for _, listener := range obs.listeners {
		session.waiter.Go(func() {
			listener(tsm)
		})
	}
}
