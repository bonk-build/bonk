// Copyright © 2025 Colden Cullen
// SPDX-License-Identifier: MIT

package task

type Result struct {
	Outputs       []string      `json:"outputs"`
	FollowupTasks []GenericTask `json:"followupTasks"`
}
