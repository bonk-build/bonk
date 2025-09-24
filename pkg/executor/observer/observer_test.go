// Copyright © 2025 Colden Cullen
// SPDX-License-Identifier: MIT

package observer_test

import (
	"context"
	"sync"
	"testing"
	"testing/synctest"

	"go.uber.org/mock/gomock"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"go.bonk.build/pkg/executor/observer"
	"go.bonk.build/pkg/task"
)

func TestPass(t *testing.T) {
	t.Parallel()

	synctest.Test(t, func(t *testing.T) {
		mock := gomock.NewController(t)
		exec := task.NewMockExecutor(mock)
		session := task.NewTestSession()
		obs := observer.New(exec)

		cont := make(chan struct{})

		result := task.Result{}
		tskID := task.NewID("testing")
		tsk := task.New(
			tskID,
			session,
			"exec",
			nil,
		)
		exepectedStatus := observer.StatusRunning

		callCount := 0
		err := obs.Listen(func(tsm observer.TaskStatusMsg) {
			callCount++
			assert.Equal(t, tskID, tsm.TaskID)
			assert.Equal(t, exepectedStatus, tsm.Status)
			assert.NoError(t, tsm.Error)
		})
		require.NoError(t, err)

		exec.EXPECT().
			Execute(t.Context(), tsk, &result).
			Times(1).
			DoAndReturn(func(context.Context, *task.Task, *task.Result) error {
				<-cont

				return nil
			})

		execWaiter := sync.WaitGroup{}
		execWaiter.Go(func() {
			err := obs.Execute(t.Context(), tsk, &result)
			require.NoError(t, err)
		})

		synctest.Wait()
		assert.Equal(t, 1, callCount)

		exepectedStatus = observer.StatusSuccess
		cont <- struct{}{}
		execWaiter.Wait()
		assert.Equal(t, 2, callCount)
	})
}

func TestFail(t *testing.T) {
	t.Parallel()

	synctest.Test(t, func(t *testing.T) {
		mock := gomock.NewController(t)
		exec := task.NewMockExecutor(mock)
		session := task.NewTestSession()
		obs := observer.New(exec)

		cont := make(chan struct{})

		result := task.Result{}
		tskID := task.NewID("testing")
		tsk := task.New(
			tskID,
			session,
			"exec",
			nil,
		)
		exepectedStatus := observer.StatusRunning

		callCount := 0
		err := obs.Listen(func(tsm observer.TaskStatusMsg) {
			callCount++
			assert.Equal(t, tskID, tsm.TaskID)
			assert.Equal(t, exepectedStatus, tsm.Status)
			if tsm.Status == observer.StatusError {
				assert.ErrorIs(t, tsm.Error, assert.AnError)
			}
		})
		require.NoError(t, err)

		exec.EXPECT().
			Execute(t.Context(), tsk, &result).
			Times(1).
			DoAndReturn(func(context.Context, *task.Task, *task.Result) error {
				<-cont

				return assert.AnError
			})

		execWaiter := sync.WaitGroup{}
		execWaiter.Go(func() {
			err := obs.Execute(t.Context(), tsk, &result)
			assert.ErrorIs(t, err, assert.AnError)
		})

		synctest.Wait()
		assert.Equal(t, 1, callCount)

		exepectedStatus = observer.StatusError
		cont <- struct{}{}
		execWaiter.Wait()
		assert.Equal(t, 2, callCount)
	})
}
