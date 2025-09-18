// Copyright © 2025 Colden Cullen
// SPDX-License-Identifier: MIT

package argconv_test

import (
	"testing"

	"go.uber.org/mock/gomock"

	"github.com/stretchr/testify/require"

	"go.bonk.build/pkg/executor/argconv"
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
	tsk := task.New("", session, "", defaultArgs)

	unboxed, err := argconv.UnboxArgs[Args](tsk)

	require.NoError(t, err)
	require.Equal(t, defaultArgs, *unboxed)
}

func Test_StringMap(t *testing.T) {
	t.Parallel()

	session := task.NewTestSession()
	tsk := task.New("", session, "", map[string]any{
		"Val1": defaultArgs.Val1,
		"Val2": defaultArgs.Val2,
	})

	unboxed, err := argconv.UnboxArgs[Args](tsk)

	require.NoError(t, err)
	require.Equal(t, defaultArgs, *unboxed)
}

func Test_CueConstraints(t *testing.T) {
	t.Parallel()

	session := task.NewTestSession()
	args := defaultArgs
	args.Val2 = 90000
	tsk := task.New("", session, "", args)

	unboxed, err := argconv.UnboxArgs[Args](tsk)

	require.Error(t, err)
	require.Nil(t, unboxed)
}

func Test_BoxExecutor(t *testing.T) {
	t.Parallel()

	mock := gomock.NewController(t)
	exec := argconv.NewMockTypedExecutor[Args](mock)
	session := task.NewTestSession()

	tsk := task.New("", session, "", defaultArgs)

	exec.EXPECT().Execute(t.Context(), tsk, &defaultArgs, nil).Times(1)

	err := argconv.BoxExecutor(exec).Execute(t.Context(), tsk, nil)
	require.NoError(t, err)
}

func Test_BoxExecutor_Failure(t *testing.T) {
	t.Parallel()

	mock := gomock.NewController(t)
	exec := argconv.NewMockTypedExecutor[Args](mock)
	boxed := argconv.BoxExecutor(exec)
	session := task.NewTestSession()

	typed := task.New("", session, "", 111)

	exec.EXPECT().Execute(t.Context(), typed, gomock.Any(), gomock.Any()).Times(0)

	err := boxed.Execute(t.Context(), typed, nil)
	require.ErrorContains(t, err, "failed to convert params from int to argconv_test.Args")
}
