// Copyright © 2025 Colden Cullen
// SPDX-License-Identifier: MIT

package executor // import "go.bonk.build/pkg/executor"

import (
	"context"
	"fmt"
	"log/slog"

	"go.uber.org/multierr"

	"google.golang.org/protobuf/types/known/structpb"

	"cuelang.org/go/cue"

	"github.com/google/uuid"

	bonkv0 "go.bonk.build/api/proto/bonk/v0"
	"go.bonk.build/pkg/task"
)

func NewRPC(name string, cuectx *cue.Context, client bonkv0.ExecutorServiceClient) Executor {
	return &rpcExecutor{
		name:   name,
		cuectx: cuectx,
		client: client,
	}
}

type rpcExecutor struct {
	name   string
	cuectx *cue.Context
	client bonkv0.ExecutorServiceClient
}

var (
	_ Executor       = (*rpcExecutor)(nil)
	_ SessionManager = (*rpcExecutor)(nil)
)

func (pb *rpcExecutor) Name() string {
	return pb.name
}

func (pb *rpcExecutor) OpenSession(ctx context.Context, session task.Session) error {
	sessionIdString := session.ID().String()
	openSessionRequest := bonkv0.OpenSessionRequest_builder{
		SessionId: &sessionIdString,
	}
	if localSession, ok := session.(task.LocalSession); ok {
		localPath := localSession.LocalPath()
		openSessionRequest.Local = bonkv0.OpenSessionRequest_WorkspaceDescriptionLocal_builder{
			AbsolutePath: &localPath,
		}.Build()
	}
	_, err := pb.client.OpenSession(ctx, openSessionRequest.Build())
	if err != nil {
		return fmt.Errorf("failed to open session with executor: %w", err)
	}

	return nil
}

func (pb *rpcExecutor) CloseSession(ctx context.Context, sessionId uuid.UUID) {
	sessionIdString := sessionId.String()
	_, err := pb.client.CloseSession(ctx, bonkv0.CloseSessionRequest_builder{
		SessionId: &sessionIdString,
	}.Build())
	if err != nil {
		slog.WarnContext(
			ctx,
			"error returned when closing session",
			"plugin",
			pb.name,
			"session",
			sessionId.String(),
		)
	}
}

func (pb *rpcExecutor) Execute(ctx context.Context, tsk task.Task, result *task.Result) error {
	sessionIdStr := tsk.Session.ID().String()
	taskReqBuilder := bonkv0.ExecuteTaskRequest_builder{
		SessionId:  &sessionIdStr,
		Name:       &tsk.ID.Name,
		Executor:   &tsk.ID.Executor,
		Inputs:     tsk.Inputs,
		Parameters: &structpb.Struct{},
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
		// Attmpt to decode the event params
		params := pb.cuectx.Encode(followup.GetParameters())
		if !multierr.AppendInto(&err, params.Err()) {
			// Create the new task and append it
			result.FollowupTasks[ii] = task.New(
				tsk.Session,
				followup.GetExecutor(),
				fmt.Sprintf("%s.%s", tsk.ID.Name, followup.GetName()),
				params,
				followup.GetInputs()...,
			)
		}
	}

	if err != nil {
		return fmt.Errorf("failed to schedule followup tasks: %w", err)
	}

	return nil
}
