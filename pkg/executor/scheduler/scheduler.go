// Copyright © 2025 Colden Cullen
// SPDX-License-Identifier: MIT

// Package scheduler provides an executor which executes followup tasks and resolves dependencies.
// This executor is meant to be the root of an executor tree, as Execute will return a combined result
// for the task executed and all followups.
package scheduler

import (
	"context"

	"golang.org/x/sync/errgroup"

	"go.bonk.build/pkg/executor"
	"go.bonk.build/pkg/task"
)

const NoConcurrencyLimit int = -1

func New(exec executor.Executor, maxConcurrency int) executor.Executor {
	return &scheduler{
		Executor:       exec,
		maxConcurrency: maxConcurrency,
	}
}

type scheduler struct {
	executor.Executor

	maxConcurrency int
}

// Execute implements executor.Executor.
// Execute will execute the task and all of it's followups, as well as wait for dependencies to resolve.
func (s *scheduler) Execute(ctx context.Context, tsk *task.Task, result *task.Result) error {
	errgrp, ctx := errgroup.WithContext(ctx)
	errgrp.SetLimit(s.maxConcurrency)

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
