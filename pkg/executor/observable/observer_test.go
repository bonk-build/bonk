// Copyright © 2025 Colden Cullen
// SPDX-License-Identifier: MIT

package observable_test

import (
	"context"
	"testing"
	"testing/synctest"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"go.bonk.build/pkg/executor/mockexec"
	"go.bonk.build/pkg/executor/observable"
	"go.bonk.build/pkg/task"
)

func TestPass(t *testing.T) { //nolint:paralleltest
	synctest.Test(t, func(t *testing.T) {
		exec := mockexec.New(t)
		session := task.NewTestSession()
		obs := observable.New(exec)

		cont := make(chan struct{})

		result := task.Result{}
		tskID := task.NewID("testing")
		tsk := task.New(
			tskID,
			"exec",
			nil,
		)
		exepectedStatus := observable.StatusRunning

		callCount := 0
		err := obs.Listen(func(tsm observable.TaskStatusMsg) {
			callCount++
			assert.Equal(t, tskID, tsm.TaskID)
			assert.Equal(t, exepectedStatus, tsm.Status)
			assert.NoError(t, tsm.Error)
		})
		require.NoError(t, err)

		exec.EXPECT().OpenSession(t.Context(), session)
		exec.EXPECT().CloseSession(t.Context(), session.ID())
		exec.EXPECT().
			Execute(t.Context(), session, tsk, &result).
			Times(1).
			DoAndReturn(func(context.Context, task.Session, *task.Task, *task.Result) error {
				<-cont

				return nil
			})

		err = obs.OpenSession(t.Context(), session)
		require.NoError(t, err)

		go func() {
			err := obs.Execute(t.Context(), session, tsk, &result)
			assert.NoError(t, err)
		}()

		synctest.Wait()
		assert.Equal(t, 1, callCount)

		obs.CloseSession(t.Context(), session.ID())

		exepectedStatus = observable.StatusSuccess
		cont <- struct{}{}
		synctest.Wait()
		assert.Equal(t, 2, callCount)
	})
}

func TestFail(t *testing.T) { //nolint:paralleltest
	synctest.Test(t, func(t *testing.T) {
		exec := mockexec.New(t)
		session := task.NewTestSession()
		obs := observable.New(exec)

		cont := make(chan struct{})

		result := task.Result{}
		tskID := task.NewID("testing")
		tsk := task.New(
			tskID,
			"exec",
			nil,
		)
		exepectedStatus := observable.StatusRunning

		callCount := 0
		err := obs.Listen(func(tsm observable.TaskStatusMsg) {
			callCount++
			assert.Equal(t, tskID, tsm.TaskID)
			assert.Equal(t, exepectedStatus, tsm.Status)
			if tsm.Status == observable.StatusError {
				assert.ErrorIs(t, tsm.Error, assert.AnError)
			}
		})
		require.NoError(t, err)

		exec.EXPECT().OpenSession(t.Context(), session)
		exec.EXPECT().CloseSession(t.Context(), session.ID())
		exec.EXPECT().
			Execute(t.Context(), session, tsk, &result).
			Times(1).
			DoAndReturn(func(context.Context, task.Session, *task.Task, *task.Result) error {
				<-cont

				return assert.AnError
			})

		err = obs.OpenSession(t.Context(), session)
		require.NoError(t, err)

		go func() {
			err := obs.Execute(t.Context(), session, tsk, &result)
			assert.ErrorIs(t, err, assert.AnError)
		}()

		synctest.Wait()
		assert.Equal(t, 1, callCount)

		obs.CloseSession(t.Context(), session.ID())

		exepectedStatus = observable.StatusError
		cont <- struct{}{}
		synctest.Wait()
		assert.Equal(t, 2, callCount)
	})
}

func TestUnopened(t *testing.T) {
	t.Parallel()

	session := task.NewTestSession()
	result := task.Result{}
	tskID := task.NewID("testing")
	tsk := task.New(
		tskID,
		"exec",
		nil,
	)

	obs := observable.New(nil)
	err := obs.Execute(t.Context(), session, tsk, &result)
	require.ErrorIs(t, err, observable.ErrUnopenedSession)
}
