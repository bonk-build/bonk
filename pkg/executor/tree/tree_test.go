// Copyright © 2025 Colden Cullen
// SPDX-License-Identifier: MIT

package tree_test

import (
	"testing"

	"go.uber.org/mock/gomock"

	"github.com/stretchr/testify/require"

	"go.bonk.build/pkg/executor/tree"
	"go.bonk.build/pkg/task"
	"go.bonk.build/test"
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

	exec := test.NewMockExecutor[any](mock)
	exec.EXPECT().Name().Return(execName)

	manager := tree.NewExecutorManager("")

	err := manager.RegisterExecutors(exec)
	require.NoError(t, err)

	require.Equal(t, 1, manager.GetNumExecutors())

	var foundName string
	calls := 0
	manager.ForEachExecutor(func(name string, exec task.GenericExecutor) {
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
	tsk := task.GenericTask{
		ID: task.TaskId{
			Executor: execName,
		},
	}

	exec := test.NewMockExecutor[any](mock)
	exec.EXPECT().Name().Return(execName)
	exec.EXPECT().Execute(t.Context(), gomock.Any(), &result)

	manager := tree.NewExecutorManager("")

	err := manager.RegisterExecutors(exec)
	require.NoError(t, err)

	err = manager.Execute(t.Context(), &tsk, &result)
	require.NoError(t, err)
	require.Equal(t, execName, tsk.ID.Executor)
}

func Test_Remove(t *testing.T) {
	t.Parallel()
	testSetup(t)
	const execName = "testing.child.abc"

	exec := test.NewMockExecutor[any](mock)
	exec.EXPECT().Name().Return(execName)
	manager := tree.NewExecutorManager("")

	err := manager.RegisterExecutors(exec)
	require.NoError(t, err)

	manager.UnregisterExecutors(execName)

	require.Equal(t, 0, manager.GetNumExecutors())

	calls := 0
	manager.ForEachExecutor(func(name string, exec task.GenericExecutor) {
		calls++
	})
	require.Equal(t, 0, calls)
}

func Test_Add_Overlap(t *testing.T) {
	t.Parallel()
	testSetup(t)

	execNames := []string{
		"testing.child.abc",
		"testing.sibling",
	}

	manager := tree.NewExecutorManager("")

	for _, execName := range execNames {
		exec := test.NewMockExecutor[any](mock)
		exec.EXPECT().Name().Return(execName)

		err := manager.RegisterExecutors(exec)
		require.NoError(t, err)
	}

	calls := 0
	manager.ForEachExecutor(func(name string, exec task.GenericExecutor) {
		calls++
	})
	require.Equal(t, 2, calls)
}
