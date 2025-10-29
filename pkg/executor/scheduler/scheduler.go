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

func New(exec executor.Executor, maxConcurrency int) *Scheduler {
	return &Scheduler{
		Executor:       exec,
		maxConcurrency: maxConcurrency,
	}
}

type Scheduler struct {
	executor.Executor

	maxConcurrency int
}

// Execute implements executor.Executor.
// Execute will execute the task and all of it's followups, as well as wait for dependencies to resolve.
func (s *Scheduler) Execute(
	ctx context.Context,
	session task.Session,
	tsk *task.Task,
	result *task.Result,
) error {
	errgrp, ctx := errgroup.WithContext(ctx)
	errgrp.SetLimit(s.maxConcurrency)

	err := s.executeImpl(errgrp, ctx, session, tsk, result)
	if err != nil {
		return err
	}

	return errgrp.Wait()
}

func (s *Scheduler) ExecuteMany(
	ctx context.Context,
	session task.Session,
	tsks []*task.Task,
	result *task.Result,
) error {
	errgrp, ctx := errgroup.WithContext(ctx)
	errgrp.SetLimit(s.maxConcurrency)

	for _, tsk := range tsks {
		errgrp.Go(func() error {
			return s.executeImpl(errgrp, ctx, session, tsk, result)
		})
	}

	return errgrp.Wait()
}

func (s *Scheduler) executeImpl(
	errgrp *errgroup.Group,
	ctx context.Context,
	session task.Session,
	tsk *task.Task,
	result *task.Result,
) error {
	localRes := task.Result{}

	err := s.Executor.Execute(ctx, session, tsk, &localRes)
	if err != nil {
		return err
	}

	for _, followup := range localRes.GetFollowupTasks() {
		errgrp.Go(func() error {
			// Update the ID to be the child of this task.
			followup.ID = tsk.ID.GetChild(followup.ID.String())

			return s.executeImpl(errgrp, ctx, session, followup, result)
		})
	}

	// Only append outputs, as we've handled followups
	result.AddOutputs(localRes.GetOutputs()...)

	return nil
}
