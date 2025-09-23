// Copyright © 2025 Colden Cullen
// SPDX-License-Identifier: MIT

//nolint:contextcheck // It's generally wrong about stream.Context() returning a new context
package rpc

import (
	"context"
	"errors"
	"fmt"
	"log/slog"

	"google.golang.org/grpc"
	"google.golang.org/grpc/status"

	"github.com/spf13/afero"

	bonkv0 "go.bonk.build/api/bonk/v0"
	"go.bonk.build/pkg/task"
)

// NewGRPCClient creates an executor that forwards task invocations across a GRPC connection.
func NewGRPCClient(
	conn *grpc.ClientConn,
) task.Executor {
	return &grpcClient{
		client: bonkv0.NewExecutorServiceClient(conn),
	}
}

type grpcClient struct {
	client bonkv0.ExecutorServiceClient
}

var _ task.Executor = (*grpcClient)(nil)

func (pb *grpcClient) OpenSession(ctx context.Context, session task.Session) error {
	slog.DebugContext(ctx, "opening session", "session", session.ID())

	sessionIdString := session.ID().String()
	defaultLevel := int64(slog.LevelInfo)
	addSource := false
	openSessionRequest := bonkv0.OpenSessionRequest_builder{
		SessionId: &sessionIdString,
		LogStreaming: bonkv0.OpenSessionRequest_LogStreamingOptions_builder{
			Level:     &defaultLevel,
			AddSource: &addSource,
		}.Build(),
	}

	if localSession, ok := session.(task.LocalSession); ok {
		localPath := localSession.LocalPath()
		openSessionRequest.Local = bonkv0.OpenSessionRequest_WorkspaceDescriptionLocal_builder{
			AbsolutePath: &localPath,
		}.Build()
	}
	if _, ok := session.SourceFS().(*afero.MemMapFs); ok {
		openSessionRequest.Test = bonkv0.OpenSessionRequest_WorkspaceDescriptionTest_builder{}.Build()
	}
	stream, err := pb.client.OpenSession(ctx, openSessionRequest.Build())
	if err != nil {
		return fmt.Errorf("failed to open session stream: %w", err)
	}

	// Wait for ack message
	msg, err := stream.Recv()
	if err != nil {
		return fmt.Errorf("error receiving ack: %w", err)
	}
	if msg.WhichMessage() != bonkv0.OpenSessionResponse_Ack_case {
		return errors.New("expected ack, received other message")
	}

	// Start up log streaming goroutine
	go handleLogStreaming(stream)

	return nil
}

func (pb *grpcClient) CloseSession(ctx context.Context, sessionId task.SessionId) {
	sessionIdString := sessionId.String()
	_, err := pb.client.CloseSession(ctx, bonkv0.CloseSessionRequest_builder{
		Id: &sessionIdString,
	}.Build())
	if err != nil {
		slog.ErrorContext(ctx, "got error closing session", "session", sessionId, "error", err)
	}
}

func (pb *grpcClient) Execute(
	ctx context.Context,
	tsk *task.Task,
	result *task.Result,
) error {
	sessionIdStr := tsk.Session.ID().String()
	taskReqBuilder := bonkv0.ExecuteTaskRequest_builder{
		SessionId: &sessionIdStr,
		Id:        (*string)(&tsk.ID),
		Executor:  &tsk.Executor,
		Inputs:    tsk.Inputs,
	}

	var err error
	taskReqBuilder.Arguments, err = ToProtoValue(tsk.Args)
	if err != nil {
		return fmt.Errorf("failed to encode args to proto: %w", err)
	}

	res, err := pb.client.ExecuteTask(ctx, taskReqBuilder.Build())
	if err != nil {
		status := status.Convert(err)
		if status.Code() == CodeExecErr {
			return errors.New(status.Message())
		} else {
			return fmt.Errorf("unknown error performing task: %w", err)
		}
	}

	result.Outputs = res.GetOutput()
	result.FollowupTasks = make([]task.Task, len(res.GetFollowupTasks()))
	for ii, followup := range res.GetFollowupTasks() {
		// Create the new task and append it
		result.FollowupTasks[ii] = *task.New(
			followup.GetId(),
			tsk.Session,
			followup.GetExecutor(),
			followup.GetArguments().AsInterface(),
		).WithInputs(followup.GetInputs()...)
	}

	return nil
}
