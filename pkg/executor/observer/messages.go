// Copyright © 2025 Colden Cullen
// SPDX-License-Identifier: MIT

package observer

import "go.bonk.build/pkg/task"

// TaskStatus describes the current status of a task.
type TaskStatus int

const (
	// StatusNone means that the task is unknown. [Observer] will never emit this status.
	StatusNone TaskStatus = iota
	// StatusRunning means that the task has begun executing.
	StatusRunning
	// StatusSuccess means the task has completed successfully.
	StatusSuccess
	// StatusError means the task has returned an error.
	StatusError
)

// TaskStatusMsg signifies a task's change in status.
type TaskStatusMsg struct {
	// TaskID is the task that this event is referring to.
	TaskID task.ID
	// Status is the new status for the task.
	Status TaskStatus

	// If Status == [StatusError], this will contain the error message returned from the executor chain.
	Error error
}

// TaskRunningMsg creates a [TaskStatusMsg] for a task with [StatusRunning].
func TaskRunningMsg(id task.ID) TaskStatusMsg {
	return TaskStatusMsg{
		TaskID: id,
		Status: StatusRunning,
	}
}

// TaskFinishedMsg creates a [TaskStatusMsg] for a task that has finished executing.
// Status is set to either [StatusSuccess] or [StatusError] (in which case Error is also set).
func TaskFinishedMsg(id task.ID, err error) TaskStatusMsg {
	if err != nil {
		return TaskStatusMsg{
			TaskID: id,
			Status: StatusError,
			Error:  err,
		}
	}

	return TaskStatusMsg{
		TaskID: id,
		Status: StatusSuccess,
	}
}
