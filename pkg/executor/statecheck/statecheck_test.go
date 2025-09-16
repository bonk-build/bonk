// Copyright © 2025 Colden Cullen
// SPDX-License-Identifier: MIT

package statecheck_test

import (
	"errors"
	"testing"

	"go.uber.org/mock/gomock"

	"github.com/spf13/afero"
	"github.com/stretchr/testify/require"

	"go.bonk.build/pkg/executor/statecheck"
	"go.bonk.build/pkg/task"
)

func TestStateCheck_SaveState(t *testing.T) {
	t.Parallel()

	mock := gomock.NewController(t)
	exec := task.NewMockExecutor[any](mock)
	checker := statecheck.New(exec)
	tsk, result := makeTestTask(t)

	exec.EXPECT().Execute(t.Context(), &tsk, &result).Return(nil).Times(1)

	err := checker.Execute(t.Context(), &tsk, &result)
	require.NoError(t, err)

	exists, err := afero.Exists(tsk.OutputFS(), statecheck.StateFile)
	require.NoError(t, err)
	require.True(t, exists)

	// Run again, ensure no error and that task was only executed once
	err = checker.Execute(t.Context(), &tsk, &result)
	require.NoError(t, err)
}

func TestStateCheck_ExecFailure(t *testing.T) {
	t.Parallel()

	mock := gomock.NewController(t)
	exec := task.NewMockExecutor[any](mock)
	checker := statecheck.New(exec)
	tsk, result := makeTestTask(t)

	expectederr := errors.New("failed to do thing")

	exec.EXPECT().Execute(t.Context(), &tsk, &result).Return(expectederr).Times(1)

	err := checker.Execute(t.Context(), &tsk, &result)
	require.ErrorIs(t, err, expectederr)

	exists, err := afero.Exists(tsk.OutputFS(), statecheck.StateFile)
	require.NoError(t, err)
	require.False(t, exists)
}

func TestStateCheck_StateMismatches_Args(t *testing.T) {
	t.Parallel()

	mock := gomock.NewController(t)
	exec := task.NewMockExecutor[any](mock)
	checker := statecheck.New(exec)
	tsk, result := makeTestTask(t)

	exec.EXPECT().Execute(t.Context(), &tsk, &result).Return(nil).Times(2)

	err := checker.Execute(t.Context(), &tsk, &result)
	require.NoError(t, err)

	exists, err := afero.Exists(tsk.OutputFS(), statecheck.StateFile)
	require.NoError(t, err)
	require.True(t, exists)

	tsk.Args = 12

	// Run again, ensure no error and that task was only executed once
	err = checker.Execute(t.Context(), &tsk, &result)
	require.NoError(t, err)
}
