// Copyright © 2025 Colden Cullen
// SPDX-License-Identifier: MIT

package tree_test

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"go.bonk.build/pkg/executor"
	"go.bonk.build/pkg/executor/mockexec"
	"go.bonk.build/pkg/executor/tree"
	"go.bonk.build/pkg/task"
)

func Test_Add(t *testing.T) {
	t.Parallel()

	const execName = "testing.child.abc"

	exec := mockexec.New(t)

	manager := tree.New()

	err := manager.RegisterExecutor(execName, exec)
	require.NoError(t, err)

	require.Equal(t, 1, manager.GetNumExecutors())

	var foundName string
	calls := 0
	manager.ForEachExecutor(func(name string, _ executor.Executor) {
		foundName = name
		calls++
	})
	require.Equal(t, 1, calls)
	require.Equal(t, execName, foundName)
}

func Test_Add_Duplicate(t *testing.T) {
	t.Parallel()

	exec := mockexec.New(t)

	manager := tree.New()

	err := manager.RegisterExecutor("testing.child", exec)
	require.NoError(t, err)

	err = manager.RegisterExecutor("testing.child", exec)
	require.ErrorIs(t, err, tree.ErrDuplicateExecutor)

	err = manager.RegisterExecutor("testing.child.abc", exec)
	require.ErrorIs(t, err, tree.ErrDuplicateExecutor)

	require.Equal(t, 1, manager.GetNumExecutors())
}

func Test_Call(t *testing.T) {
	t.Parallel()

	const execName = "testing.child.abc"
	var result task.Result
	tsk := task.Task{
		Executor: execName,
	}

	exec := mockexec.New(t)
	exec.EXPECT().Execute(t.Context(), &tsk, &result)

	manager := tree.New()

	err := manager.RegisterExecutor(execName, exec)
	require.NoError(t, err)

	err = manager.Execute(t.Context(), &tsk, &result)
	require.NoError(t, err)
	require.Equal(t, execName, tsk.Executor)
}

func Test_Call_Fail(t *testing.T) {
	t.Parallel()

	const execName = "testing.child.abc"
	var result task.Result
	tsk := task.Task{
		Executor: execName,
	}

	exec := mockexec.New(t)
	exec.EXPECT().Execute(t.Context(), &tsk, &result).Times(0)

	manager := tree.New()

	err := manager.RegisterExecutor("something.else", exec)
	require.NoError(t, err)

	err = manager.Execute(t.Context(), &tsk, &result)
	require.Error(t, err)
	require.ErrorIs(t, err, tree.ErrNoExecutorFound)
}

func Test_Remove(t *testing.T) {
	t.Parallel()

	const execName = "testing.child.abc"

	exec := mockexec.New(t)
	manager := tree.New()

	err := manager.RegisterExecutor(execName, exec)
	require.NoError(t, err)

	manager.UnregisterExecutors(execName)

	require.Equal(t, 0, manager.GetNumExecutors())

	calls := 0
	manager.ForEachExecutor(func(string, executor.Executor) {
		calls++
	})
	require.Equal(t, 0, calls)
}

func Test_Add_Overlap(t *testing.T) {
	t.Parallel()

	execNames := []string{
		"testing.child.abc",
		"testing.sibling",
	}

	manager := tree.New()

	for _, execName := range execNames {
		exec := mockexec.New(t)

		err := manager.RegisterExecutor(execName, exec)
		require.NoError(t, err)
	}

	calls := 0
	manager.ForEachExecutor(func(string, executor.Executor) {
		calls++
	})
	require.Equal(t, 2, calls)
}

func Test_OpenCloseSession(t *testing.T) {
	t.Parallel()

	const execName = "testing.child.abc"

	session := task.NewTestSession()

	exec := mockexec.New(t)
	exec.EXPECT().OpenSession(t.Context(), session).Times(1)
	exec.EXPECT().CloseSession(t.Context(), session.ID()).Times(1)

	manager := tree.New()

	err := manager.RegisterExecutor(execName, exec)
	require.NoError(t, err)

	err = manager.OpenSession(t.Context(), session)
	require.NoError(t, err)
	defer manager.CloseSession(t.Context(), session.ID())
}

func Test_OpenCloseSession_Error(t *testing.T) {
	t.Parallel()

	const execName = "testing.child.abc"

	session := task.NewTestSession()

	exec := mockexec.New(t)
	exec.EXPECT().OpenSession(t.Context(), session).Times(1).Return(assert.AnError)
	exec.EXPECT().CloseSession(t.Context(), session.ID()).Times(1)

	manager := tree.New()

	err := manager.RegisterExecutor(execName, exec)
	require.NoError(t, err)

	err = manager.OpenSession(t.Context(), session)
	require.ErrorIs(t, err, assert.AnError)
	defer manager.CloseSession(t.Context(), session.ID())
}
