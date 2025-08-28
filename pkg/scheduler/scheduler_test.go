// Copyright © 2025 Colden Cullen
// SPDX-License-Identifier: MIT

package scheduler

import (
	"os"
	"testing"

	"go.uber.org/mock/gomock"

	"cuelang.org/go/cue"

	"github.com/stretchr/testify/require"

	"go.bonk.build/pkg/task"
)

//go:generate go tool mockgen -destination scheduler_mock_test.go -package scheduler -typed . TaskSender

func Test_SenderIsCalled(t *testing.T) {
	t.Parallel()

	// Make sure this doesn't persist to other tests
	t.Cleanup(func() {
		require.NoError(t, os.RemoveAll(".bonk"))
	})

	mock := gomock.NewController(t)

	result := task.TaskResult{}

	sender := NewMockTaskSender(mock)
	sender.EXPECT().
		SendTask(gomock.Any(), gomock.Any()).
		Times(1).
		Return(&result, nil)

	scheduler := NewScheduler(sender, 1)

	require.NoError(t, scheduler.AddTask(task.New("test", "task1", cue.Value{})))

	scheduler.Run()
}
