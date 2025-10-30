// Copyright © 2025 Colden Cullen
// SPDX-License-Identifier: MIT

package router_test

import (
	"maps"
	"slices"
	"sync"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"

	"go.bonk.build/pkg/executor"
	"go.bonk.build/pkg/executor/mockexec"
	"go.bonk.build/pkg/executor/router"
	"go.bonk.build/pkg/task"
)

func Test_Add(t *testing.T) {
	t.Parallel()

	executors := map[string]*mockexec.MockExecutor{
		"testing.child.abc": mockexec.NewMockExecutor(t),
		"testing.child":     mockexec.NewMockExecutor(t),
		"testing.sibling":   mockexec.NewMockExecutor(t),
		"unrelated":         mockexec.NewMockExecutor(t),
		"super.*":           mockexec.NewMockExecutor(t),
	}

	execErrs := map[string]error{
		"testing.child": router.ErrDuplicateExecutor,
	}

	taskRoutings := map[string]string{
		"testing.child.abc": "testing.child.abc",
		"testing.child":     "testing.child",
		"testing.child.def": "testing.child",
		"super.testing":     "super.*",
	}

	rtr := router.New()
	session := task.NewTestSession()

	// Validate expected successful registrations
	waiter := sync.WaitGroup{}
	for name, exec := range executors {
		exec.EXPECT().OpenSession(t.Context(), session).Return(nil)
		exec.EXPECT().CloseSession(t.Context(), session.ID())

		waiter.Go(func() {
			err := rtr.RegisterExecutor(name, exec)
			assert.NoError(t, err)
		})
	}
	waiter.Wait()

	// Validate expected errors
	for name, expectedErr := range execErrs {
		waiter.Go(func() {
			err := rtr.RegisterExecutor(name, nil)
			require.ErrorIs(t, err, expectedErr)
		})
	}
	waiter.Wait()

	// Validate session opening
	err := rtr.OpenSession(t.Context(), session)
	require.NoError(t, err)

	// Validate resulting router
	assert.Equal(t, len(executors), rtr.GetNumExecutors())
	found := make([]string, 0, len(executors))
	rtr.ForEachExecutor(func(name string, _ executor.Executor) {
		found = append(found, name)
	})
	assert.ElementsMatch(t, slices.Collect(maps.Keys(executors)), found)

	// Validate task delivery
	for sent, receive := range taskRoutings {
		tsk := task.New(
			task.NewID("testing"),
			sent,
			nil,
		)

		exec := executors[receive]
		exec.EXPECT().Execute(t.Context(), session, tsk, (*task.Result)(nil)).Return(nil)

		err := rtr.Execute(t.Context(), session, tsk, nil)
		require.NoError(t, err)
	}

	// Validate session closing
	rtr.CloseSession(t.Context(), session.ID())

	// Validate unregistration
	numExecs := rtr.GetNumExecutors()
	for name := range executors {
		rtr.UnregisterExecutors(name)
		numExecs--

		assert.Equal(t, numExecs, rtr.GetNumExecutors())
	}
}

func Test_Call(t *testing.T) {
	t.Parallel()

	const execName = "testing.child.abc"
	session := task.NewTestSession()
	var result task.Result
	tsk := task.Task{
		Executor: execName,
	}

	exec := mockexec.NewMockExecutor(t)
	exec.EXPECT().Execute(t.Context(), session, &tsk, &result).Return(nil)

	rtr := router.New()

	err := rtr.RegisterExecutor(execName, exec)
	require.NoError(t, err)

	err = rtr.Execute(t.Context(), session, &tsk, &result)
	require.NoError(t, err)
	require.Equal(t, execName, tsk.Executor)
}

func Test_Call_Wildcard(t *testing.T) {
	t.Parallel()

	session := task.NewTestSession()
	var result task.Result
	tsk := task.Task{
		Executor: "testing.child.abc",
	}

	exec := mockexec.NewMockExecutor(t)
	exec.EXPECT().Execute(t.Context(), session, &tsk, &result).Return(nil)

	rtr := router.New()

	err := rtr.RegisterExecutor("testing.*.abc", exec)
	require.NoError(t, err)

	err = rtr.Execute(t.Context(), session, &tsk, &result)
	require.NoError(t, err)
}

func Test_Call_Fail(t *testing.T) {
	t.Parallel()

	const execName = "testing.child.abc"
	session := task.NewTestSession()
	var result task.Result
	tsk := task.Task{
		Executor: execName,
	}

	exec := mockexec.NewMockExecutor(t)

	rtr := router.New()

	err := rtr.RegisterExecutor("something.else", exec)
	require.NoError(t, err)

	err = rtr.Execute(t.Context(), session, &tsk, &result)
	require.Error(t, err)
	require.ErrorIs(t, err, router.ErrNoExecutorFound)
}

func Test_Call_Overlap(t *testing.T) {
	t.Parallel()

	execNames := []string{
		"testing.child.abc",
		"testing.sibling",
	}

	rtr := router.New()

	for _, execName := range execNames {
		exec := mockexec.NewMockExecutor(t)

		err := rtr.RegisterExecutor(execName, exec)
		require.NoError(t, err)
	}

	exec := mockexec.NewMockExecutor(t)
	exec.EXPECT().Execute(t.Context(), nil, mock.Anything, (*task.Result)(nil)).Return(nil)

	err := rtr.RegisterExecutor("testing.child", exec)
	require.NoError(t, err)

	err = rtr.Execute(t.Context(), nil, &task.Task{
		Executor: "testing.child",
	}, nil)
	require.NoError(t, err)
}

func Test_OpenCloseSession_Error(t *testing.T) {
	t.Parallel()

	const execName = "testing.child.abc"

	session := task.NewTestSession()

	exec := mockexec.NewMockExecutor(t)
	exec.EXPECT().OpenSession(t.Context(), session).Return(assert.AnError)
	exec.EXPECT().CloseSession(t.Context(), session.ID())

	rtr := router.New()

	err := rtr.RegisterExecutor(execName, exec)
	require.NoError(t, err)

	err = rtr.OpenSession(t.Context(), session)
	require.ErrorIs(t, err, assert.AnError)
	defer rtr.CloseSession(t.Context(), session.ID())
}
