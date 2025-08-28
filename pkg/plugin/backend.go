// Copyright © 2025 Colden Cullen
// SPDX-License-Identifier: MIT

package plugin // import "go.bonk.build/pkg/plugin"

import (
	"context"
	"fmt"

	"google.golang.org/protobuf/types/known/structpb"

	bonkv0 "go.bonk.build/api/go/proto/bonk/v0"
	"go.bonk.build/pkg/backend"
	"go.bonk.build/pkg/task"
)

func NewBackend(name string, client bonkv0.BonkPluginServiceClient) backend.Backend {
	return &pluginBackend{
		name:   name,
		client: client,
	}
}

type pluginBackend struct {
	name   string
	client bonkv0.BonkPluginServiceClient
}

func (pb *pluginBackend) Execute(ctx context.Context, tsk task.Task) error {
	outDir := tsk.GetOutputDirectory()
	taskReqBuilder := bonkv0.PerformTaskRequest_builder{
		Backend:      &pb.name,
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

	err = tsk.SaveChecksum()
	if err != nil {
		return fmt.Errorf("failed to checksum task: %w", err)
	}

	return nil
}
