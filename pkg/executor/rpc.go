// Copyright © 2025 Colden Cullen
// SPDX-License-Identifier: MIT

package executor // import "go.bonk.build/pkg/executor"

import (
	"context"
	"fmt"

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

func (pb *rpcExecutor) Execute(ctx context.Context, tsk task.Task) ([]string, error) {
	outDir := tsk.GetOutputDirectory()
	taskReqBuilder := bonkv0.ExecuteTaskRequest_builder{
		Executor:     &pb.name,
		Inputs:       tsk.Inputs,
		Parameters:   &structpb.Struct{},
		OutDirectory: &outDir,
	}

	err := tsk.Params.Decode(taskReqBuilder.Parameters)
	if err != nil {
		return nil, fmt.Errorf("failed to encode parameters as protobuf: %w", err)
	}

	res, err := pb.client.ExecuteTask(ctx, taskReqBuilder.Build())
	if err != nil {
		return nil, fmt.Errorf("failed to call perform task: %w", err)
	}

	return res.GetOutput(), nil
}
