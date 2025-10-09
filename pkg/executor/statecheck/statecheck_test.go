// Copyright © 2025 Colden Cullen
// SPDX-License-Identifier: MIT

package statecheck_test

import (
	"testing"

	"github.com/spf13/afero"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"go.bonk.build/pkg/executor/mockexec"
	"go.bonk.build/pkg/executor/statecheck"
	"go.bonk.build/pkg/task"
)

func TestStateCheck_SaveState(t *testing.T) {
	t.Parallel()

	exec := mockexec.New(t)
	checker := statecheck.New(exec)
	tsk, result := makeTestTask(t)
	session := task.NewTestSession()

	exec.EXPECT().Execute(t.Context(), session, tsk, &result).Return(nil).Times(1)

	err := checker.Execute(t.Context(), session, tsk, &result)
	require.NoError(t, err)

	exists, err := afero.Exists(task.OutputFS(session, tsk.ID), statecheck.StateFile)
	require.NoError(t, err)
	require.True(t, exists)

	// Run again, ensure no error and that task was only executed once
	err = checker.Execute(t.Context(), session, tsk, &result)
	require.NoError(t, err)
}

func TestStateCheck_ExecFailure(t *testing.T) {
	t.Parallel()

	exec := mockexec.New(t)
	checker := statecheck.New(exec)
	tsk, result := makeTestTask(t)
	session := task.NewTestSession()

	exec.EXPECT().Execute(t.Context(), session, tsk, &result).Return(assert.AnError).Times(1)

	err := checker.Execute(t.Context(), session, tsk, &result)
	require.ErrorIs(t, err, assert.AnError)

	exists, err := afero.Exists(task.OutputFS(session, tsk.ID), statecheck.StateFile)
	require.NoError(t, err)
	require.False(t, exists)
}

func TestStateCheck_StateMismatches_Args(t *testing.T) {
	t.Parallel()

	exec := mockexec.New(t)
	checker := statecheck.New(exec)
	tsk, result := makeTestTask(t)
	session := task.NewTestSession()

	exec.EXPECT().Execute(t.Context(), session, tsk, &result).Return(nil).Times(2)

	err := checker.Execute(t.Context(), session, tsk, &result)
	require.NoError(t, err)

	exists, err := afero.Exists(task.OutputFS(session, tsk.ID), statecheck.StateFile)
	require.NoError(t, err)
	require.True(t, exists)

	tsk.Args = 12

	// Run again, ensure no error and that task was only executed once
	err = checker.Execute(t.Context(), session, tsk, &result)
	require.NoError(t, err)
}
