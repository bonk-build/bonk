// Copyright © 2025 Colden Cullen
// SPDX-License-Identifier: MIT

package scheduler

import (
	"testing"

	"go.uber.org/mock/gomock"

	"cuelang.org/go/cue"

	"github.com/spf13/afero"
	"github.com/stretchr/testify/require"

	"go.bonk.build/pkg/task"
)

//go:generate go tool mockgen -destination scheduler_mock_test.go -package scheduler -copyright_file ../../license-header.txt -typed . TaskSender

func Test_SenderIsCalled(t *testing.T) {
	t.Parallel()

	mock := gomock.NewController(t)
	project := afero.NewMemMapFs()

	sender := NewMockTaskSender(mock)
	sender.EXPECT().
		Execute(gomock.Any(), gomock.Any(), gomock.Any()).
		Times(1).
		Return(nil)

	scheduler := NewScheduler(project, sender, nil, 1)

	err := scheduler.AddTask(task.New("test", "task1", cue.Value{}))
	require.NoError(t, err)

	err = scheduler.Run(t.Context())
	require.NoError(t, err)
}
