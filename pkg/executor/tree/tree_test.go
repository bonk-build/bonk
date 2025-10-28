// Copyright © 2025 Colden Cullen
// SPDX-License-Identifier: MIT

package tree_test

import (
	"testing"

	"go.uber.org/mock/gomock"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"go.bonk.build/pkg/executor"
	"go.bonk.build/pkg/executor/mockexec"
	"go.bonk.build/pkg/executor/tree"
	"go.bonk.build/pkg/task"
)

func Test_Add(t *testing.T) {
	t.Parallel()

	execNames := [...]string{
		"testing.child.abc",
		"testing.child",
		"testing.sibling",
		"unrelated",
		"super.*",
	}

	execErrs := map[string]error{
		"testing.child": tree.ErrDuplicateExecutor,
	}

	exec := mockexec.New(t)
	manager := tree.New()
	session := task.NewTestSession()

	// Validate expected successful registrations
	for _, name := range execNames {
		err := manager.RegisterExecutor(name, exec)
		require.NoError(t, err)
	}

	// Validate expected errors
	for name, expectedErr := range execErrs {
		err := manager.RegisterExecutor(name, exec)
		require.ErrorIs(t, err, expectedErr)
	}

	// Validate session opening
	exec.EXPECT().OpenSession(t.Context(), session).Times(len(execNames))
	err := manager.OpenSession(t.Context(), session)
	require.NoError(t, err)

	// Validate resulting tree
	assert.Equal(t, len(execNames), manager.GetNumExecutors())
	found := make([]string, 0, len(execNames))
	manager.ForEachExecutor(func(name string, _ executor.Executor) {
		found = append(found, name)
	})
	assert.ElementsMatch(t, execNames, found)

	// Validate session closing
	exec.EXPECT().CloseSession(t.Context(), session.ID()).Times(len(execNames))
	manager.CloseSession(t.Context(), session.ID())

	// Validate unregistration
	manager.UnregisterExecutors(execNames[:]...)
	assert.Equal(t, 0, manager.GetNumExecutors())
}

func Test_Call(t *testing.T) {
	t.Parallel()

	const execName = "testing.child.abc"
	session := task.NewTestSession()
	var result task.Result
	tsk := task.Task{
		Executor: execName,
	}

	exec := mockexec.New(t)
	exec.EXPECT().Execute(t.Context(), session, &tsk, &result)

	manager := tree.New()

	err := manager.RegisterExecutor(execName, exec)
	require.NoError(t, err)

	err = manager.Execute(t.Context(), session, &tsk, &result)
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

	exec := mockexec.New(t)
	exec.EXPECT().Execute(t.Context(), session, &tsk, &result)

	manager := tree.New()

	err := manager.RegisterExecutor("testing.*.abc", exec)
	require.NoError(t, err)

	err = manager.Execute(t.Context(), session, &tsk, &result)
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

	exec := mockexec.New(t)
	exec.EXPECT().Execute(t.Context(), session, &tsk, &result).Times(0)

	manager := tree.New()

	err := manager.RegisterExecutor("something.else", exec)
	require.NoError(t, err)

	err = manager.Execute(t.Context(), session, &tsk, &result)
	require.Error(t, err)
	require.ErrorIs(t, err, tree.ErrNoExecutorFound)
}

func Test_Call_Overlap(t *testing.T) {
	t.Parallel()

	execNames := []string{
		"testing.child.abc",
		"testing.sibling",
	}

	manager := tree.New()

	for _, execName := range execNames {
		exec := mockexec.New(t)
		exec.EXPECT().Execute(t.Context(), nil, gomock.Any(), nil).Times(0)

		err := manager.RegisterExecutor(execName, exec)
		require.NoError(t, err)
	}

	exec := mockexec.New(t)
	exec.EXPECT().Execute(t.Context(), nil, gomock.Any(), nil).Times(1)

	err := manager.RegisterExecutor("testing.child", exec)
	require.NoError(t, err)

	err = manager.Execute(t.Context(), nil, &task.Task{
		Executor: "testing.child",
	}, nil)
	require.NoError(t, err)
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
