// Copyright © 2025 Colden Cullen
// SPDX-License-Identifier: MIT

//nolint:contextcheck // It's generally wrong about stream.Context() returning a new context
package rpc

import (
	"context"
	"errors"
	"fmt"
	"io"
	"log/slog"

	"google.golang.org/grpc"
	"google.golang.org/grpc/status"

	"github.com/spf13/afero"

	bonkv0 "go.bonk.build/api/bonk/v0"
	"go.bonk.build/pkg/task"
)

// Creates an executor that forwards task invocations across a GRPC connection.
func NewGRPCClient(
	conn *grpc.ClientConn,
) task.GenericExecutor {
	return &grpcClient{
		client:   bonkv0.NewExecutorServiceClient(conn),
		sessions: make(map[task.SessionId]grpcClientSession),
	}
}

type grpcClientSession struct {
	closeSession context.CancelFunc
}

type grpcClient struct {
	client bonkv0.ExecutorServiceClient

	sessions map[task.SessionId]grpcClientSession
}

var _ task.GenericExecutor = (*grpcClient)(nil)

func (pb *grpcClient) OpenSession(ctx context.Context, session task.Session) error {
	slog.DebugContext(ctx, "opening session", "session", session.ID())

	sessionCtx, cancel := context.WithCancel(ctx)
	pb.sessions[session.ID()] = grpcClientSession{
		closeSession: cancel,
	}

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
	stream, err := pb.client.OpenSession(ctx)
	if err != nil {
		return fmt.Errorf("failed to open session stream: %w", err)
	}

	err = stream.Send(openSessionRequest.Build())
	if err != nil {
		return fmt.Errorf("failed to send open session request: %w", err)
	}

	ack := make(chan error)

	// Start up log streaming goroutine
	go func() {
		msg, err := stream.Recv()
		if err != nil {
			ack <- err

			return
		}
		if msg.WhichMessage() == bonkv0.OpenSessionResponse_Ack_case {
			ack <- nil
		} else {
			ack <- errors.New("expected ack, received other message")

			return
		}

		for {
			msg, err := stream.Recv()
			if err != nil {
				if stream.Context().Err() != nil || errors.Is(err, io.EOF) {
					// If this occurs, the log stream is imply shutting down and we should exit
					break
				} else {
					slog.ErrorContext(
						stream.Context(),
						"received error on log stream",
						"error", err,
						"context err", stream.Context().Err())

					continue
				}
			}

			switch msg.WhichMessage() {
			case bonkv0.OpenSessionResponse_LogRecord_case:
				attrs := make([]slog.Attr, 0, len(msg.GetLogRecord().GetAttrs()))
				for key, value := range msg.GetLogRecord().GetAttrs() {
					attrs = append(attrs, slog.Attr{
						Key:   key,
						Value: slog.AnyValue(value.AsInterface()),
					})
				}

				slog.LogAttrs(
					stream.Context(),
					slog.Level(msg.GetLogRecord().GetLevel()),
					msg.GetLogRecord().GetMessage(),
					attrs...,
				)

			default:
				slog.ErrorContext(stream.Context(), "received unknown session response")

				continue
			}
		}
	}()

	// Wait for the ack to come through
	err = <-ack
	if err != nil {
		return err
	}

	go func() {
		// Wait for cancel() and close
		<-sessionCtx.Done()

		slog.DebugContext(sessionCtx, "closing session for rpc")

		err = stream.CloseSend()
		if err != nil {
			slog.WarnContext(sessionCtx, "failed to close session", "error", err)
		}
	}()

	return nil
}

func (pb *grpcClient) CloseSession(ctx context.Context, sessionId task.SessionId) {
	session, ok := pb.sessions[sessionId]
	if !ok {
		slog.ErrorContext(ctx, "attempting to close session that isn't open", "session", sessionId)
	}
	session.closeSession()
}

func (pb *grpcClient) Execute(
	ctx context.Context,
	tsk *task.GenericTask,
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
	result.FollowupTasks = make([]task.GenericTask, len(res.GetFollowupTasks()))
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
