// Copyright © 2025 Colden Cullen
// SPDX-License-Identifier: MIT

package bubbletea

import (
	"context"

	tea "github.com/charmbracelet/bubbletea/v2"

	"go.bonk.build/pkg/executor/observable"
	"go.bonk.build/pkg/task"
)

// TaskStatusUpdate returns a command that emits a task status update.
func TaskStatusUpdate(tsk *task.Task, status observable.TaskStatus) tea.Cmd {
	return func() tea.Msg {
		return observable.TaskStatusMsg{
			TaskID: tsk.ID,
			Status: status,
		}
	}
}

type TaskScheduleMsg struct {
	ctx  context.Context //nolint:containedctx
	tsk  *task.Task
	exec task.Executor
}

func ScheduleTask(
	ctx context.Context,
	tsk *task.Task,
	exec task.Executor,
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
	return tea.Sequence(TaskStatusUpdate(tsk.tsk, observable.StatusRunning), func() tea.Msg {
		var result task.Result
		err := tsk.exec.Execute(tsk.ctx, tsk.tsk, &result)
		if err != nil {
			return observable.TaskFinishedMsg(tsk.tsk.ID, err)
		}

		statusUpdateCmd := TaskStatusUpdate(tsk.tsk, observable.StatusSuccess)

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
