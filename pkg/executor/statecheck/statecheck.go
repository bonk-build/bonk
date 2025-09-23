// Copyright © 2025 Colden Cullen
// SPDX-License-Identifier: MIT

// Package statecheck provides an executor that avoids re-running tasks if they are already up to date.
// State files are saved in the task's output fs as [StateFile].
package statecheck

import (
	"context"
	"log/slog"

	"go.bonk.build/pkg/task"
)

type statechecker struct {
	task.Executor
}

func New(child task.Executor) task.Executor {
	return statechecker{
		Executor: child,
	}
}

// Execute implements task.Executor.
func (s statechecker) Execute(
	ctx context.Context,
	tsk *task.Task,
	result *task.Result,
) error {
	mismatches := DetectStateMismatches(tsk)
	if mismatches == nil {
		slog.DebugContext(ctx, "states match, skipping task")

		return nil
	}

	slog.DebugContext(ctx, "state mismatch, running task", "mismatches", mismatches)

	err := s.Executor.Execute(ctx, tsk, result)
	if err != nil {
		return err //nolint:wrapcheck
	}

	slog.DebugContext(ctx, "task succeeded, saving state")

	err = SaveState(tsk, result)
	if err != nil {
		slog.WarnContext(ctx, "failed to save task state", "error", err)

		return err
	}

	return nil
}
