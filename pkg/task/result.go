// Copyright © 2025 Colden Cullen
// SPDX-License-Identifier: MIT

package task

// Result describes the outputs of a task's execution.
type Result struct {
	// Outputs describes any files that have been emitted by the task relative to [Session.OutputFS].
	Outputs []string `json:"outputs"`
	// FollowupTasks is a list of tasks to be executed after this task completes.
	FollowupTasks []Task `json:"followupTasks"`
}
