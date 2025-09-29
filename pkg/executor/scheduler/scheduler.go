// Copyright © 2025 Colden Cullen
// SPDX-License-Identifier: MIT

package scheduler

import (
	"context"

	"golang.org/x/sync/errgroup"

	"go.bonk.build/pkg/executor"
	"go.bonk.build/pkg/task"
)

func New(exec executor.Executor) executor.Executor {
	return &scheduler{
		Executor: exec,
	}
}

type scheduler struct {
	executor.Executor
}

// Execute implements executor.Executor.
// Execute will execute the task and all of it's followups, as well as wait for dependencies to resolve.
func (s *scheduler) Execute(ctx context.Context, tsk *task.Task, result *task.Result) error {
	errgrp, ctx := errgroup.WithContext(ctx)

	err := s.executeImpl(errgrp, ctx, tsk, result)
	if err != nil {
		return err
	}

	return errgrp.Wait() //nolint:wrapcheck
}

func (s *scheduler) executeImpl(
	errgrp *errgroup.Group,
	ctx context.Context,
	tsk *task.Task,
	result *task.Result,
) error {
	err := s.Executor.Execute(ctx, tsk, result)
	if err != nil {
		return err //nolint:wrapcheck
	}

	for _, followup := range result.FollowupTasks {
		errgrp.Go(func() error {
			var res task.Result

			// Update the ID to be the child of this task.
			followup.ID = tsk.ID.GetChild(followup.ID.String())

			err := s.executeImpl(errgrp, ctx, &followup, &res)

			return err
		})
	}

	return nil
}
