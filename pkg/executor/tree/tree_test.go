// Copyright © 2025 Colden Cullen
// SPDX-License-Identifier: MIT

package tree_test

import (
	"testing"

	"go.uber.org/mock/gomock"

	"github.com/stretchr/testify/require"

	"go.bonk.build/pkg/executor/tree"
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

	exec := task.NewMockExecutor[any](mock)

	manager := tree.New()

	err := manager.RegisterExecutor(execName, exec)
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
		Executor: execName,
	}

	exec := task.NewMockExecutor[any](mock)
	exec.EXPECT().Execute(t.Context(), &tsk, &result)

	manager := tree.New()

	err := manager.RegisterExecutor(execName, exec)
	require.NoError(t, err)

	err = manager.Execute(t.Context(), &tsk, &result)
	require.NoError(t, err)
	require.Equal(t, execName, tsk.Executor)
}

func Test_Remove(t *testing.T) {
	t.Parallel()
	testSetup(t)
	const execName = "testing.child.abc"

	exec := task.NewMockExecutor[any](mock)
	manager := tree.New()

	err := manager.RegisterExecutor(execName, exec)
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

	manager := tree.New()

	for _, execName := range execNames {
		exec := task.NewMockExecutor[any](mock)

		err := manager.RegisterExecutor(execName, exec)
		require.NoError(t, err)
	}

	calls := 0
	manager.ForEachExecutor(func(name string, exec task.GenericExecutor) {
		calls++
	})
	require.Equal(t, 2, calls)
}
