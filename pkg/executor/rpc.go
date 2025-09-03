// Copyright © 2025 Colden Cullen
// SPDX-License-Identifier: MIT

package executor // import "go.bonk.build/pkg/executor"

import (
	"context"
	"fmt"

	"go.uber.org/multierr"

	"google.golang.org/protobuf/types/known/structpb"

	bonkv0 "go.bonk.build/api/go/proto/bonk/v0"
	"go.bonk.build/pkg/task"
)

func NewRPC(name string, client bonkv0.ExecutorServiceClient) Executor {
	return &rpcExecutor{
		name:   name,
		client: client,
	}
}

type rpcExecutor struct {
	name   string
	client bonkv0.ExecutorServiceClient
}

func (pb *rpcExecutor) Execute(ctx context.Context, tsk task.Task, result *task.TaskResult) error {
	outDir := tsk.ID.GetOutputDirectory()
	taskReqBuilder := bonkv0.ExecuteTaskRequest_builder{
		Executor:     &pb.name,
		Inputs:       tsk.Inputs,
		Parameters:   &structpb.Struct{},
		OutDirectory: &outDir,
	}

	err := tsk.Params.Decode(taskReqBuilder.Parameters)
	if err != nil {
		return fmt.Errorf("failed to encode parameters as protobuf: %w", err)
	}

	res, err := pb.client.ExecuteTask(ctx, taskReqBuilder.Build())
	if err != nil {
		return fmt.Errorf("failed to call perform task: %w", err)
	}

	result.Outputs = res.GetOutput()
	result.FollowupTasks = make([]task.Task, len(res.GetFollowupTasks()))
	for ii, followup := range res.GetFollowupTasks() {
		result.FollowupTasks[ii].ID = tsk.ID.GetChild(followup.GetName(), followup.GetExecutor())
		result.FollowupTasks[ii].Inputs = followup.GetInputs()

		multierr.AppendInto(&err,
			result.FollowupTasks[ii].Params.Decode(followup.GetParameters()))
	}

	if err != nil {
		return fmt.Errorf("failed to schedule followup tasks: %w", err)
	}

	return nil
}
