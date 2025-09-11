// Copyright © 2025 Colden Cullen
// SPDX-License-Identifier: MIT

package scheduler

import (
	"testing"

	"go.uber.org/mock/gomock"

	"github.com/stretchr/testify/require"

	"go.bonk.build/pkg/task"
)

func Test_SenderIsCalled(t *testing.T) {
	t.Parallel()

	mock := gomock.NewController(t)
	session := task.NewTestSession()

	sender := task.NewMockExecutor[any](mock)
	sender.EXPECT().
		Execute(gomock.Any(), gomock.Any(), gomock.Any()).
		Times(1).
		Return(nil)

	scheduler := NewScheduler(sender, 1)

	require.NoError(t, scheduler.AddTask(task.New[any](session, "test", "task1", nil)))

	scheduler.Run()
}
