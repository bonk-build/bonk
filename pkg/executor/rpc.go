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

func NewRPC(name string, client bonkv0.BonkPluginServiceClient) Executor {
	return &rpcExecutor{
		name:   name,
		client: client,
	}
}

type rpcExecutor struct {
	name   string
	client bonkv0.BonkPluginServiceClient
}

func (pb *rpcExecutor) Execute(ctx context.Context, tsk task.Task) error {
	outDir := tsk.GetOutputDirectory()
	taskReqBuilder := bonkv0.PerformTaskRequest_builder{
		Executor:     &pb.name,
		Inputs:       tsk.Inputs,
		Parameters:   &structpb.Struct{},
		OutDirectory: &outDir,
	}

	err := tsk.Params.Decode(taskReqBuilder.Parameters)
	if err != nil {
		return fmt.Errorf("failed to encode parameters as protobuf: %w", err)
	}

	_, err = pb.client.PerformTask(ctx, taskReqBuilder.Build())
	if err != nil {
		return fmt.Errorf("failed to call perform task: %w", err)
	}

	return nil
}
