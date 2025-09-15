// Copyright © 2025 Colden Cullen
// SPDX-License-Identifier: MIT

package driver

import (
	"errors"
	"testing"

	"go.uber.org/mock/gomock"
	"go.uber.org/multierr"

	"github.com/stretchr/testify/require"

	"go.bonk.build/pkg/task"
)

func TestWithExecutor(t *testing.T) {
	t.Parallel()

	const execName = "executor"

	mock := gomock.NewController(t)
	drv := NewMockDriver(mock)
	exec := task.NewMockExecutor[any](mock)

	drv.EXPECT().RegisterExecutor(execName, gomock.Any()).Times(1)

	err := WithExecutor(execName, exec)(t.Context(), drv)
	require.NoError(t, err)
}

func TestWithExecutor_Fail(t *testing.T) {
	t.Parallel()

	const execName = "executor"
	expectedErr := errors.New("failed to register executor")

	mock := gomock.NewController(t)
	drv := NewMockDriver(mock)
	exec := task.NewMockExecutor[any](mock)

	drv.EXPECT().RegisterExecutor(execName, gomock.Any()).Return(expectedErr).Times(1)

	err := WithExecutor(execName, exec)(t.Context(), drv)
	require.ErrorIs(t, err, expectedErr)
}

func TestWithPlugins(t *testing.T) {
	t.Parallel()

	mock := gomock.NewController(t)
	drv := NewMockDriver(mock)

	plugins := []string{
		"plugin a",
		"plugin b",
	}

	drv.EXPECT().StartPlugins(t.Context(), "plugin a", "plugin b").Times(1)

	err := WithPlugins(plugins...)(t.Context(), drv)
	require.NoError(t, err)
}

func TestWithLocalSession(t *testing.T) {
	t.Parallel()

	mock := gomock.NewController(t)
	drv := NewMockDriver(mock)

	drv.EXPECT().NewLocalSession(t.Context(), ".").Times(1)
	drv.EXPECT().AddTask(t.Context(), gomock.Any()).Times(2)

	err := WithLocalSession(".",
		WithTask("exec 0", "task 0", []string{}),
		WithTask("exec 1", "task 1", map[string]string{}),
	)(t.Context(), drv)
	require.NoError(t, err)
}

func TestWithLocalSession_Fail1(t *testing.T) {
	t.Parallel()

	mock := gomock.NewController(t)
	drv := NewMockDriver(mock)

	expectedErr := errors.New("failed to open session")

	drv.EXPECT().NewLocalSession(t.Context(), ".").Times(1).Return(nil, expectedErr)

	err := WithLocalSession(".")(t.Context(), drv)
	require.ErrorIs(t, err, expectedErr)
}

func TestWithLocalSession_Fail2(t *testing.T) {
	t.Parallel()

	mock := gomock.NewController(t)
	drv := NewMockDriver(mock)

	expectedErr := errors.New("failed to create task")

	drv.EXPECT().NewLocalSession(t.Context(), ".").Times(1)
	drv.EXPECT().AddTask(t.Context(), gomock.Any()).Return(expectedErr).Times(2)

	err := WithLocalSession(".",
		WithTask("exec 0", "task 0", []string{}),
		WithTask("exec 1", "task 1", map[string]string{}),
	)(t.Context(), drv)
	require.Error(t, err)
	require.True(t, multierr.Every(err, expectedErr))
}
