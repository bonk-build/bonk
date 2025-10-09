// Copyright © 2025 Colden Cullen
// SPDX-License-Identifier: MIT

package task_test

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"go.bonk.build/pkg/task"
)

func TestNew(t *testing.T) {
	t.Parallel()

	tsk := task.New(
		task.NewID("root", "child"),
		"exec",
		nil,
	)

	require.NotNil(t, tsk)
	assert.Equal(t, tsk.ID, task.ID("root.child"))
	assert.Equal(t, "exec", tsk.Executor)
	assert.Nil(t, tsk.Args)
	assert.Empty(t, tsk.Inputs)
	assert.Empty(t, tsk.Dependencies)
}

func TestNewWithInputs(t *testing.T) {
	t.Parallel()

	tsk := task.New(
		task.NewID("root", "child"),
		"exec",
		nil,
		task.WithInputs(
			"InputA",
		),
	)

	require.NotNil(t, tsk)
	assert.Equal(t, tsk.ID, task.ID("root.child"))
	assert.Equal(t, "exec", tsk.Executor)
	assert.Nil(t, tsk.Args)
	assert.Len(t, tsk.Inputs, 1)
	assert.Equal(t, "InputA", tsk.Inputs[0])
	assert.Empty(t, tsk.Dependencies)
}

func TestNewWithDependencies(t *testing.T) {
	t.Parallel()

	tsk := task.New(
		task.NewID("root", "child"),
		"exec",
		nil,
		task.WithDependencies(
			task.NewID("root", "sibling"),
		),
	)

	require.NotNil(t, tsk)
	require.Equal(t, tsk.ID, task.ID("root.child"))
	require.Equal(t, "exec", tsk.Executor)
	require.Nil(t, tsk.Args)
	require.Empty(t, tsk.Inputs)
	require.Len(t, tsk.Dependencies, 1)
	require.Equal(t, task.ID("root.sibling"), tsk.Dependencies[0])
}
