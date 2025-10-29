// Copyright © 2025 Colden Cullen
// SPDX-License-Identifier: MIT

package scheduler_test

import (
	"context"
	"strconv"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"

	"go.bonk.build/pkg/executor/mockexec"
	"go.bonk.build/pkg/executor/scheduler"
	"go.bonk.build/pkg/task"
)

func TestFollowups(t *testing.T) {
	t.Parallel()

	const numFollowups = 3

	exec := mockexec.NewMockExecutor(t)
	session := task.NewTestSession()

	sched := scheduler.New(exec, scheduler.NoConcurrencyLimit)

	exec.EXPECT().OpenSession(t.Context(), session).Return(nil)
	exec.EXPECT().CloseSession(t.Context(), session.ID())

	err := sched.OpenSession(t.Context(), session)
	require.NoError(t, err)
	defer sched.CloseSession(t.Context(), session.ID())

	res := task.Result{}
	tsk := task.New(
		task.NewID("testing"),
		"none",
		nil,
	)

	exec.EXPECT().
		Execute(mock.IsType(t.Context()), session, tsk, mock.IsType(&res)).
		Return(nil).
		Run(func(ctx context.Context, _ task.Session, t *task.Task, r *task.Result) {
			for idx := range numFollowups {
				r.AddFollowupTasks(task.New(
					task.NewID("child", strconv.Itoa(idx)),
					"none",
					nil,
				))
			}
		})
	for idx := range numFollowups {
		exec.EXPECT().
			Execute(
				mock.IsType(t.Context()),
				session,
				task.TaskIDMatches(tsk.ID.GetChild("child", strconv.Itoa(idx))),
				mock.IsType(&res),
			).
			Return(nil)
	}

	err = sched.Execute(t.Context(), session, tsk, &res)
	require.NoError(t, err)
}

func TestErrNoFollowups(t *testing.T) {
	t.Parallel()

	exec := mockexec.NewMockExecutor(t)
	session := task.NewTestSession()

	sched := scheduler.New(exec, scheduler.NoConcurrencyLimit)

	exec.EXPECT().OpenSession(t.Context(), session).Return(nil)
	exec.EXPECT().CloseSession(t.Context(), session.ID())

	err := sched.OpenSession(t.Context(), session)
	require.NoError(t, err)
	defer sched.CloseSession(t.Context(), session.ID())

	res := task.Result{}
	tsk := task.New(
		task.NewID("testing"),
		"none",
		nil,
	)

	exec.EXPECT().
		Execute(mock.IsType(t.Context()), session, tsk, mock.IsType(&res)).
		Return(assert.AnError).
		Run(func(ctx context.Context, _ task.Session, t *task.Task, r *task.Result) {
			for idx := range 3 {
				r.AddFollowupTasks(task.New(
					task.NewID("child", strconv.Itoa(idx)),
					"none",
					nil,
				))
			}
		})

	err = sched.Execute(t.Context(), session, tsk, &res)
	require.ErrorIs(t, err, assert.AnError)
}

func TestFollowupsErrs(t *testing.T) {
	t.Parallel()

	exec := mockexec.NewMockExecutor(t)
	session := task.NewTestSession()

	sched := scheduler.New(exec, scheduler.NoConcurrencyLimit)

	exec.EXPECT().OpenSession(t.Context(), session).Return(nil)
	exec.EXPECT().CloseSession(t.Context(), session.ID())

	err := sched.OpenSession(t.Context(), session)
	require.NoError(t, err)
	defer sched.CloseSession(t.Context(), session.ID())

	res := task.Result{}
	tsk := task.New(
		task.NewID("testing"),
		"none",
		nil,
	)

	exec.EXPECT().
		Execute(mock.IsType(t.Context()), session, tsk, mock.IsType(&res)).
		Return(nil).
		Run(func(ctx context.Context, _ task.Session, t *task.Task, r *task.Result) {
			for idx := range 3 {
				r.AddFollowupTasks(task.New(
					task.NewID("child", strconv.Itoa(idx)),
					"none",
					nil,
				))
			}
		})
	exec.EXPECT().
		Execute(mock.IsType(t.Context()), session, mock.IsType(tsk), mock.IsType(&res)).
		Return(assert.AnError)

	err = sched.Execute(t.Context(), session, tsk, &res)
	require.ErrorIs(t, err, assert.AnError)
}
