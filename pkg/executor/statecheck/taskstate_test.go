// Copyright © 2025 Colden Cullen
// SPDX-License-Identifier: MIT

package statecheck_test

import (
	"testing"

	"github.com/spf13/afero"
	"github.com/stretchr/testify/require"

	"go.bonk.build/pkg/executor/statecheck"
	"go.bonk.build/pkg/task"
)

func makeTestTask(t *testing.T) (*task.Task, task.Result) {
	t.Helper()

	tsk := task.New(
		task.ID("Test.Testing"),
		"test.abc.def",
		nil,
	)
	result := task.Result{
		Outputs: []string{
			"output-file",
		},
	}

	return tsk, result
}

func TestTaskState_SaveState(t *testing.T) {
	t.Parallel()

	tsk, result := makeTestTask(t)
	session := task.NewTestSession()

	err := statecheck.SaveState(session, tsk, &result)
	require.NoError(t, err)

	exists, err := afero.Exists(task.OutputFS(session, tsk.ID), statecheck.StateFile)
	require.NoError(t, err)
	require.True(t, exists)
}

func TestTaskState_StateMismatches_Args(t *testing.T) {
	t.Parallel()

	tsk, result := makeTestTask(t)
	session := task.NewTestSession()

	err := statecheck.SaveState(session, tsk, &result)
	require.NoError(t, err)

	mismatches := statecheck.DetectStateMismatches(session, tsk)
	require.Empty(t, mismatches)

	tsk.Args = 12

	mismatches = statecheck.DetectStateMismatches(session, tsk)
	require.Len(t, mismatches, 1)
	require.Contains(t, mismatches, "arguments-checksum")
}

func TestTaskState_StateMismatches_Inputs(t *testing.T) {
	t.Parallel()

	const inputFileName = "input-file"
	const inputFileContents = "This if the first iteration of the file"

	tsk, result := makeTestTask(t)
	session := task.NewTestSession()

	err := statecheck.SaveState(session, tsk, &result)
	require.NoError(t, err)

	mismatches := statecheck.DetectStateMismatches(session, tsk)
	require.Empty(t, mismatches)

	tsk.Inputs = []string{inputFileName}

	inputFile, err := session.SourceFS().Create(inputFileName)
	require.NoError(t, err)

	written, err := inputFile.WriteString(inputFileContents)
	require.NoError(t, err)
	require.Equal(t, len(inputFileContents), written)

	mismatches = statecheck.DetectStateMismatches(session, tsk)
	require.Len(t, mismatches, 2)
	require.Contains(t, mismatches, "inputs")
	require.Contains(t, mismatches, "inputs-checksum")
}

func TestTaskState_StateMismatches_InputsChecksum(t *testing.T) {
	t.Parallel()

	const inputFileName = "input-file"
	const inputFileContents1 = "This if the first iteration of the file"
	const inputFileContents2 = "This if the first iteration of the file"

	tsk, result := makeTestTask(t)
	session := task.NewTestSession()
	tsk.Inputs = []string{inputFileName}

	inputFile, err := session.SourceFS().Create(inputFileName)
	require.NoError(t, err)

	written, err := inputFile.WriteString(inputFileContents1)
	require.NoError(t, err)
	require.Equal(t, len(inputFileContents1), written)

	err = statecheck.SaveState(session, tsk, &result)
	require.NoError(t, err)

	mismatches := statecheck.DetectStateMismatches(session, tsk)
	require.Empty(t, mismatches)

	written, err = inputFile.WriteString(inputFileContents1)
	require.NoError(t, err)
	require.Equal(t, len(inputFileContents2), written)

	mismatches = statecheck.DetectStateMismatches(session, tsk)
	require.Len(t, mismatches, 1)
	require.Contains(t, mismatches, "inputs-checksum")
}

func TestTaskState_StateMismatches_Executor(t *testing.T) {
	t.Parallel()

	tsk, result := makeTestTask(t)
	session := task.NewTestSession()

	err := statecheck.SaveState(session, tsk, &result)
	require.NoError(t, err)

	mismatches := statecheck.DetectStateMismatches(session, tsk)
	require.Empty(t, mismatches)

	tsk.Executor = "Different"

	mismatches = statecheck.DetectStateMismatches(session, tsk)
	require.Len(t, mismatches, 1)
	require.Contains(t, mismatches, "executor")
}
