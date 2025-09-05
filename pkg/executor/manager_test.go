// Copyright © 2025 Colden Cullen
// SPDX-License-Identifier: MIT

package executor_test

import (
	"testing"

	"go.uber.org/mock/gomock"

	"github.com/stretchr/testify/require"

	"go.bonk.build/pkg/executor"
	"go.bonk.build/pkg/task"
)

var mock *gomock.Controller

func testSetup(t *testing.T) {
	t.Helper()

	mock = gomock.NewController(t)
}

func Test_Add(t *testing.T) {
	t.Parallel()
	testSetup(t)
	const execName = "testing.child.abc"

	exec := NewMockExecutor(mock)
	manager := executor.NewExecutorManager()

	err := manager.RegisterExecutor(execName, exec)
	require.NoError(t, err)

	require.Equal(t, 1, manager.GetNumExecutors())

	var foundName string
	calls := 0
	manager.ForEachExecutor(func(name string, exec executor.Executor) {
		foundName = name
		calls++
	})
	require.Equal(t, 1, calls)
	require.Equal(t, execName, foundName)
}

func Test_Call(t *testing.T) {
	t.Parallel()
	testSetup(t)
	const execName = "testing.child.abc"
	var result task.Result

	exec := NewMockExecutor(mock)
	exec.EXPECT().Execute(t.Context(), gomock.Any(), &result)

	manager := executor.NewExecutorManager()

	err := manager.RegisterExecutor(execName, exec)
	require.NoError(t, err)

	err = manager.Execute(t.Context(), task.Task{
		ID: task.TaskId{
			Executor: execName,
		},
	}, &result)
	require.NoError(t, err)
}

func Test_Remove(t *testing.T) {
	t.Parallel()
	testSetup(t)
	const execName = "testing.child.abc"

	exec := NewMockExecutor(mock)
	manager := executor.NewExecutorManager()

	err := manager.RegisterExecutor(execName, exec)
	require.NoError(t, err)

	manager.UnregisterExecutor(execName)

	require.Equal(t, 0, manager.GetNumExecutors())

	calls := 0
	manager.ForEachExecutor(func(name string, exec executor.Executor) {
		calls++
	})
	require.Equal(t, 0, calls)
}

func Test_Add_Overlap(t *testing.T) {
	t.Parallel()
	testSetup(t)

	execNames := []string{
		"testing.child.abc",
		"testing",
	}

	exec := NewMockExecutor(mock)
	manager := executor.NewExecutorManager()

	for _, execName := range execNames {
		err := manager.RegisterExecutor(execName, exec)
		require.NoError(t, err)
	}

	calls := 0
	manager.ForEachExecutor(func(name string, exec executor.Executor) {
		calls++
	})
	require.Equal(t, 2, calls)
}

func Test_Call_Overlap(t *testing.T) {
	t.Parallel()
	testSetup(t)

	var result task.Result

	execNames := []string{
		"testing.child.abc",
		"testing",
	}

	abc := NewMockExecutor(mock)
	abc.EXPECT().Execute(t.Context(), gomock.Any(), &result).Times(1)

	testing := NewMockExecutor(mock)
	testing.EXPECT().Execute(t.Context(), gomock.Any(), &result).Times(2)

	manager := executor.NewExecutorManager()

	err := manager.RegisterExecutor(execNames[0], abc)
	require.NoError(t, err)
	err = manager.RegisterExecutor(execNames[1], testing)
	require.NoError(t, err)

	tsk := task.Task{}

	tsk.ID.Executor = execNames[0]
	err = manager.Execute(t.Context(), tsk, &result)
	require.NoError(t, err)

	tsk.ID.Executor = "testing.child"
	err = manager.Execute(t.Context(), tsk, &result)
	require.NoError(t, err)

	tsk.ID.Executor = "testing"
	err = manager.Execute(t.Context(), tsk, &result)
	require.NoError(t, err)

	tsk.ID.Executor = ""
	err = manager.Execute(t.Context(), tsk, &result)
	require.ErrorIs(t, err, executor.ErrNoExecutorFound)
}
