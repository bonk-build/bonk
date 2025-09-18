// Copyright © 2025 Colden Cullen
// SPDX-License-Identifier: MIT

package taskflow_test

import (
	"testing"

	"go.uber.org/mock/gomock"

	"github.com/stretchr/testify/require"

	"go.bonk.build/pkg/scheduler/taskflow"
	"go.bonk.build/pkg/task"
)

func Test_SenderIsCalled(t *testing.T) {
	t.Parallel()

	mock := gomock.NewController(t)
	session := task.NewTestSession()
	tsk := task.New[any]("task1", session, "test", nil)

	sender := task.NewMockExecutor(mock)
	sender.EXPECT().
		Execute(gomock.Any(), tsk, gomock.Any()).
		Times(1).
		Return(nil)

	scheduler := taskflow.New(1)(t.Context(), sender)

	err := scheduler.AddTask(t.Context(), tsk)
	require.NoError(t, err)

	scheduler.Run()
}
