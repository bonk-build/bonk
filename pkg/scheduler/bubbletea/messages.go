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
	StatusScheduled TaskStatus = iota
	StatusSuccess
	StatusFail
)

// Message signifying a task's change in status.
type TaskStatusMsg struct {
	tskId  string
	status TaskStatus
}

// Command that emits a task status update.
func TaskStatusUpdate(tsk *task.GenericTask, status TaskStatus) tea.Cmd {
	return func() tea.Msg {
		return TaskStatusMsg{
			tskId:  tsk.ID.Name,
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
			return TaskStatusUpdate(tsk.tsk, StatusFail)
		}

		followups := make([]tea.Cmd, 1+len(result.FollowupTasks))
		followups[0] = TaskStatusUpdate(tsk.tsk, StatusSuccess)
		for idx, followup := range result.FollowupTasks {
			followups[1+idx] = ScheduleTask(
				tsk.ctx,
				&followup,
				tsk.exec,
			)
		}

		return tea.BatchMsg(followups)
	})
}
