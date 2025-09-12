// Copyright © 2025 Colden Cullen
// SPDX-License-Identifier: MIT

package task_test

import (
	"testing"

	"go.uber.org/mock/gomock"

	"github.com/stretchr/testify/require"

	"go.bonk.build/pkg/task"
)

type Args struct {
	Val1 string `json:"val1"`
	Val2 int    `json:"val2" cue:"<70000"`
}

var defaultArgs = Args{
	Val1: "test string",
	Val2: 69420,
}

func Test_StraightConversion(t *testing.T) {
	t.Parallel()

	session := task.NewTestSession()
	typed := task.New(session, "", "", defaultArgs)

	boxed := typed.Box()
	unboxed, err := task.Unbox[Args](boxed)

	require.NoError(t, err)
	require.Equal(t, typed, unboxed)
}

func Test_StringMap(t *testing.T) {
	t.Parallel()

	session := task.NewTestSession()
	typed := task.New(session, "", "", map[string]any{
		"Val1": defaultArgs.Val1,
		"Val2": defaultArgs.Val2,
	})

	boxed := typed.Box()
	unboxed, err := task.Unbox[Args](boxed)

	require.NoError(t, err)
	require.Equal(t, defaultArgs, unboxed.Args)
}

func Test_CueConstraints(t *testing.T) {
	t.Parallel()

	session := task.NewTestSession()
	args := defaultArgs
	args.Val2 = 90000
	typed := task.New(session, "", "", args)

	boxed := typed.Box()
	unboxed, err := task.Unbox[Args](boxed)

	require.Error(t, err)
	require.Nil(t, unboxed)
}

func Test_BoxExecutor(t *testing.T) {
	t.Parallel()

	mock := gomock.NewController(t)
	exec := task.NewMockExecutor[Args](mock)
	session := task.NewTestSession()

	exec.EXPECT().Execute(t.Context(), gomock.Any(), nil).Times(1)

	typed := task.New(session, "", "", defaultArgs)
	boxed := typed.Box()

	err := task.BoxExecutor(exec).Execute(t.Context(), boxed, nil)
	require.NoError(t, err)
}

func Test_BoxExecutor_Failure(t *testing.T) {
	t.Parallel()

	mock := gomock.NewController(t)
	exec := task.NewMockExecutor[Args](mock)
	boxed := task.BoxExecutor(exec)
	session := task.NewTestSession()

	exec.EXPECT().Execute(gomock.Any(), gomock.Any(), gomock.Any()).Times(0)

	typed := task.New[any](session, "", "", 111)

	err := boxed.Execute(t.Context(), typed, nil)
	require.ErrorContains(t, err, "failed to convert params from int to task_test.Args")
}
