// Copyright © 2025 Colden Cullen
// SPDX-License-Identifier: MIT

// Package observable provides an executor which alerts followers to task status
package observable

import (
	"context"
	"errors"
	"fmt"
	"sync"

	"go.bonk.build/pkg/executor"
	"go.bonk.build/pkg/task"
)

type Observer = func(TaskStatusMsg)

type Observable interface {
	executor.Executor

	Listen(f Observer) error
}

var ErrUnopenedSession = errors.New("task being executed for unopened session")

func New(exec executor.Executor) Observable {
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
	exec     executor.Executor
	sessions map[task.SessionID]*observSession

	listeners []Observer
}

func (obs *observ) Execute(
	ctx context.Context,
	session task.Session,
	tsk *task.Task,
	result *task.Result,
) error {
	obsSession, ok := obs.sessions[session.ID()]
	if !ok {
		return fmt.Errorf("%w: %s", ErrUnopenedSession, session.ID())
	}

	obs.trigger(obsSession, TaskRunningMsg(tsk.ID))

	err := obs.exec.Execute(ctx, session, tsk, result)

	obs.trigger(obsSession, TaskFinishedMsg(tsk.ID, err))

	return err
}

// OpenSession implements Observable.
func (obs *observ) OpenSession(ctx context.Context, session task.Session) error {
	obs.sessions[session.ID()] = &observSession{}

	return obs.exec.OpenSession(ctx, session)
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
