// Copyright © 2025 Colden Cullen
// SPDX-License-Identifier: MIT

package task

import (
	"encoding/json"
	"sync"
)

// Result describes the outputs of a task's execution.
type Result struct {
	mu sync.RWMutex

	// outputs describes any files that have been emitted by the task relative to [Session.OutputFS].
	outputs []string
	// followupTasks is a list of tasks to be executed after this task completes.
	followupTasks []Task
}

type resultJson struct {
	Outputs       []string `json:"outputs"`
	FollowupTasks []*Task  `json:"followupTasks"`
}

var (
	_ json.Marshaler   = (*Result)(nil)
	_ json.Unmarshaler = (*Result)(nil)
)

func (r *Result) GetOutputs() []string {
	if r == nil {
		return []string{}
	}

	r.mu.RLock()
	defer r.mu.RUnlock()

	return r.outputs
}

func (r *Result) AddOutputs(outputs ...string) {
	if r == nil {
		return
	}

	r.mu.Lock()
	defer r.mu.Unlock()

	r.outputs = append(r.outputs, outputs...)
}

func (r *Result) GetFollowupTasks() []*Task {
	if r == nil {
		return []*Task{}
	}

	r.mu.RLock()
	defer r.mu.RUnlock()

	result := make([]*Task, len(r.followupTasks))
	for idx := range r.followupTasks {
		result[idx] = &r.followupTasks[idx]
	}

	return result
}

func (r *Result) AddFollowupTasks(tasks ...*Task) {
	if r == nil {
		return
	}

	r.mu.Lock()
	defer r.mu.Unlock()

	newFollowups := make([]Task, len(r.followupTasks), len(r.followupTasks)+len(tasks))
	copy(newFollowups, r.followupTasks)

	for _, tsk := range tasks {
		newFollowups = append(newFollowups, *tsk)
	}
	r.followupTasks = newFollowups
}

func (r *Result) Append(other *Result) {
	if r == nil || other == nil || r == other {
		return
	}

	r.mu.Lock()
	other.mu.RLock()
	defer r.mu.Unlock()
	defer other.mu.RUnlock()

	r.outputs = append(r.outputs, other.outputs...)
	r.followupTasks = append(r.followupTasks, other.followupTasks...)
}

// MarshalJSON implements json.Marshaler.
func (r *Result) MarshalJSON() ([]byte, error) {
	if r == nil {
		return json.Marshal(nil) //nolint:wrapcheck
	}

	return json.Marshal(resultJson{ //nolint:wrapcheck
		Outputs:       r.GetOutputs(),
		FollowupTasks: r.GetFollowupTasks(),
	})
}

// UnmarshalJSON implements json.Unmarshaler.
func (r *Result) UnmarshalJSON(data []byte) error {
	res := resultJson{}
	err := json.Unmarshal(data, &res)
	if err != nil {
		return err //nolint:wrapcheck
	}

	r.AddOutputs(res.Outputs...)
	r.AddFollowupTasks(res.FollowupTasks...)

	return nil
}
