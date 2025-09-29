// Copyright © 2025 Colden Cullen
// SPDX-License-Identifier: MIT

package driver

import (
	"testing"

	"go.uber.org/mock/gomock"

	"github.com/stretchr/testify/require"

	"go.bonk.build/pkg/executor/argconv"
	"go.bonk.build/pkg/executor/observable"
	"go.bonk.build/pkg/task"
)

func TestWithConcurrency(t *testing.T) {
	t.Parallel()

	const concurrency = uint(0xDEADBEEF)

	options := MakeDefaultOptions().
		WithConcurrency(concurrency)
	require.Equal(t, concurrency, options.Concurrency)
}

func TestWithExecutor(t *testing.T) {
	t.Parallel()

	const execName = "executor"

	mock := gomock.NewController(t)
	exec := task.NewMockExecutor(mock)

	options := MakeDefaultOptions().
		WithExecutor(execName, exec)

	require.Len(t, options.Executors, 1)
	require.Same(t, exec, options.Executors[execName])
}

func TestWithTypedExecutor(t *testing.T) {
	t.Parallel()

	const execName = "executor"

	mock := gomock.NewController(t)
	exec := argconv.NewMockTypedExecutor[any](mock)

	options := MakeDefaultOptions().
		WithExecutor(execName, argconv.BoxExecutor(exec))

	require.Len(t, options.Executors, 1)
	require.NotNil(t, options.Executors[execName])
}

func TestWithPlugins(t *testing.T) {
	t.Parallel()

	plugins := []string{
		"plugin a",
		"plugin b",
	}

	options := MakeDefaultOptions().
		WithPlugins(plugins...)

	require.ElementsMatch(t, options.Plugins, plugins)
}

func TestWithLocalSession(t *testing.T) {
	t.Parallel()

	options := MakeDefaultOptions().
		WithLocalSession(".",
			WithTask("exec 0", "task 0", []string{}),
			WithTask("exec 1", "task 1", map[string]string{}),
		)

	require.Len(t, options.Sessions, 1)
	for _, tsks := range options.Sessions {
		require.Len(t, tsks, 2)
	}
}

func TestWithObservers(t *testing.T) {
	t.Parallel()

	options := MakeDefaultOptions().
		WithObservers(
			func(observable.TaskStatusMsg) {},
		)

	require.Len(t, options.Observers, 1)
}
