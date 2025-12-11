// Copyright © 2025 Colden Cullen
// SPDX-License-Identifier: MIT

package task

import (
	"encoding/json"
	"fmt"
	"slices"
	"strings"
	"sync"
)

// Result describes the outputs of a task's execution.
type Result struct {
	mu sync.RWMutex

	// outputs describes any files that have been emitted by the task relative to [Session.OutputFS].
	outputs []FileReference
	// followupTasks is a list of tasks to be executed after this task completes.
	followupTasks []Task
}

type resultJson struct {
	Outputs       []FileReference `json:"outputs"`
	FollowupTasks []*Task         `json:"followupTasks"`
}

var (
	_ json.Marshaler   = (*Result)(nil)
	_ json.Unmarshaler = (*Result)(nil)
	_ fmt.Stringer     = (*Result)(nil)
)

func (r *Result) GetOutputs() []FileReference {
	if r == nil {
		return []FileReference{}
	}

	r.mu.RLock()
	defer r.mu.RUnlock()

	return r.outputs
}

func (r *Result) AddOutputs(outputs ...FileReference) []FileReference {
	if r == nil {
		return []FileReference{}
	}

	r.mu.Lock()
	defer r.mu.Unlock()

	r.outputs = append(r.outputs, outputs...)

	return r.outputs[len(r.outputs)-len(outputs):]
}

func (r *Result) AddOutputPaths(outputPaths ...string) []FileReference {
	if r == nil {
		return []FileReference{}
	}

	r.mu.Lock()
	defer r.mu.Unlock()

	r.outputs = slices.Grow(r.outputs, len(outputPaths))
	for _, outPath := range outputPaths {
		r.outputs = append(r.outputs, FileReference{
			FileSystem: FsOutput,
			Path:       outPath,
		})
	}

	return r.outputs[len(r.outputs)-len(outputPaths):]
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

	r.outputs = res.Outputs
	r.AddFollowupTasks(res.FollowupTasks...)

	return nil
}

func (r *Result) String() string {
	if r == nil {
		return "<nil>"
	}

	propStrings := make([]string, 0, 2) //nolint:mnd
	if outputs := r.GetOutputs(); len(outputs) > 0 {
		propStrings = append(propStrings, fmt.Sprintf("Outputs: %v", outputs))
	}
	if followups := r.GetFollowupTasks(); len(followups) > 0 {
		propStrings = append(propStrings, fmt.Sprintf("Followups: %v", followups))
	}

	return fmt.Sprintf("Result{%s}", strings.Join(propStrings, ", "))
}
