// Copyright © 2025 Colden Cullen
// SPDX-License-Identifier: MIT

package scheduler_test

import (
	"context"
	"strconv"
	"testing"

	"go.uber.org/mock/gomock"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"go.bonk.build/pkg/executor/scheduler"
	"go.bonk.build/pkg/task"
)

func TestFollowups(t *testing.T) {
	t.Parallel()

	const numFollowups = 3

	mock := gomock.NewController(t)
	exec := task.NewMockExecutor(mock)
	session := task.NewTestSession()

	sched := scheduler.New(exec)

	exec.EXPECT().OpenSession(t.Context(), session).Times(1)
	exec.EXPECT().CloseSession(t.Context(), session.ID()).Times(1)

	err := sched.OpenSession(t.Context(), session)
	require.NoError(t, err)
	defer sched.CloseSession(t.Context(), session.ID())

	res := task.Result{}
	tsk := task.New(
		task.NewID("testing"),
		session,
		"none",
		nil,
	)

	exec.EXPECT().
		Execute(gomock.Any(), tsk, &res).
		Times(1).
		Return(nil).
		Do(func(ctx context.Context, t *task.Task, r *task.Result) {
			for idx := range numFollowups {
				r.FollowupTasks = append(r.FollowupTasks, *task.New(
					task.NewID("child", strconv.Itoa(idx)),
					t.Session,
					"none",
					nil,
				))
			}
		})
	exec.EXPECT().
		Execute(gomock.Any(), gomock.Any(), gomock.Any()).
		Times(numFollowups)

	err = sched.Execute(t.Context(), tsk, &res)
	require.NoError(t, err)
}

func TestErrNoFollowups(t *testing.T) {
	t.Parallel()

	mock := gomock.NewController(t)
	exec := task.NewMockExecutor(mock)
	session := task.NewTestSession()

	sched := scheduler.New(exec)

	exec.EXPECT().OpenSession(t.Context(), session).Times(1)
	exec.EXPECT().CloseSession(t.Context(), session.ID()).Times(1)

	err := sched.OpenSession(t.Context(), session)
	require.NoError(t, err)
	defer sched.CloseSession(t.Context(), session.ID())

	res := task.Result{}
	tsk := task.New(
		task.NewID("testing"),
		session,
		"none",
		nil,
	)

	exec.EXPECT().
		Execute(gomock.Any(), tsk, &res).
		Times(1).
		Return(assert.AnError).
		Do(func(ctx context.Context, t *task.Task, r *task.Result) {
			for idx := range 3 {
				r.FollowupTasks = append(r.FollowupTasks, *task.New(
					task.NewID("child", strconv.Itoa(idx)),
					t.Session,
					"none",
					nil,
				))
			}
		})
	exec.EXPECT().
		Execute(gomock.Any(), gomock.Any(), gomock.Any()).
		Times(0)

	err = sched.Execute(t.Context(), tsk, &res)
	require.ErrorIs(t, err, assert.AnError)
}
