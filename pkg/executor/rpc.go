// Copyright © 2025 Colden Cullen
// SPDX-License-Identifier: MIT

package executor // import "go.bonk.build/pkg/executor"

import (
	"context"
	"fmt"
	"log/slog"

	"go.uber.org/multierr"

	"google.golang.org/protobuf/types/known/structpb"

	"github.com/google/uuid"

	bonkv0 "go.bonk.build/api/proto/bonk/v0"
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

var (
	_ Executor       = (*rpcExecutor)(nil)
	_ SessionManager = (*rpcExecutor)(nil)
)

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
		newTask := &result.FollowupTasks[ii]
		newTask.ID = tsk.ID.GetChild(followup.GetName(), followup.GetExecutor())
		newTask.Session = tsk.Session
		newTask.Inputs = followup.GetInputs()

		multierr.AppendInto(&err,
			newTask.Params.Decode(followup.GetParameters()))
	}

	if err != nil {
		return fmt.Errorf("failed to schedule followup tasks: %w", err)
	}

	return nil
}
