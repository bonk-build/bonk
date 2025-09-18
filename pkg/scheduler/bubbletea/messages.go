// Copyright © 2025 Colden Cullen
// SPDX-License-Identifier: MIT

package bubbletea

import (
	"context"

	tea "github.com/charmbracelet/bubbletea/v2"

	"go.bonk.build/pkg/task"
)

type TaskStatus int

const (
	StatusNone TaskStatus = iota
	StatusScheduled
	StatusSuccess
	StatusFail
)

// Message signifying a task's change in status.
type TaskStatusMsg struct {
	tskId  task.TaskID
	status TaskStatus

	err error
}

// Command that emits a task status update.
func TaskStatusUpdate(tsk *task.GenericTask, status TaskStatus) tea.Cmd {
	return func() tea.Msg {
		return TaskStatusMsg{
			tskId:  tsk.ID,
			status: status,
		}
	}
}

type TaskScheduleMsg struct {
	ctx  context.Context //nolint:containedctx
	tsk  *task.GenericTask
	exec task.GenericExecutor
}

func ScheduleTask(
	ctx context.Context,
	tsk *task.GenericTask,
	exec task.GenericExecutor,
) tea.Cmd {
	return func() tea.Msg {
		return TaskScheduleMsg{
			ctx:  ctx,
			tsk:  tsk,
			exec: exec,
		}
	}
}

func (tsk TaskScheduleMsg) GetExecCmd() tea.Cmd {
	return tea.Sequence(TaskStatusUpdate(tsk.tsk, StatusScheduled), func() tea.Msg {
		var result task.Result
		err := tsk.exec.Execute(tsk.ctx, tsk.tsk, &result)
		if err != nil {
			return TaskStatusMsg{
				tskId:  tsk.tsk.ID,
				status: StatusFail,
				err:    err,
			}
		}

		statusUpdateCmd := TaskStatusUpdate(tsk.tsk, StatusSuccess)

		if len(result.FollowupTasks) > 0 {
			followups := make([]tea.Cmd, len(result.FollowupTasks))
			for idx, followup := range result.FollowupTasks {
				followup.ID = tsk.tsk.ID.GetChild(followup.ID.String())
				followups[idx] = ScheduleTask(
					tsk.ctx,
					&followup,
					tsk.exec,
				)
			}

			statusUpdateCmd = tea.Sequence(tea.Batch(followups...), statusUpdateCmd)
		}

		return statusUpdateCmd()
	})
}
