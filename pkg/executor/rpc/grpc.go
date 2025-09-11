// Copyright © 2025 Colden Cullen
// SPDX-License-Identifier: MIT

package rpc // import "go.bonk.build/pkg/executor/rpc"

import (
	"context"
	"errors"
	"fmt"
	"log/slog"

	"go.uber.org/multierr"

	"google.golang.org/grpc"

	"github.com/google/uuid"
	"github.com/spf13/afero"

	bonkv0 "go.bonk.build/api/proto/bonk/v0"
	"go.bonk.build/pkg/task"
)

// Creates an executor that forwards task invocations across a GRPC connection.
func NewGRPCClient(
	name string,
	conn *grpc.ClientConn,
) task.GenericExecutor {
	return &grpcClient{
		name:   name,
		client: bonkv0.NewExecutorServiceClient(conn),
	}
}

type grpcClient struct {
	name   string
	client bonkv0.ExecutorServiceClient
}

var _ task.GenericExecutor = (*grpcClient)(nil)

func (pb *grpcClient) Name() string {
	return pb.name
}

func (pb *grpcClient) OpenSession(ctx context.Context, session task.Session) error {
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
	if _, ok := session.SourceFS().(*afero.MemMapFs); ok {
		openSessionRequest.Test = bonkv0.OpenSessionRequest_WorkspaceDescriptionTest_builder{}.Build()
	}
	_, err := pb.client.OpenSession(ctx, openSessionRequest.Build())
	if err != nil {
		return fmt.Errorf("failed to open session with executor: %w", err)
	}

	return nil
}

func (pb *grpcClient) CloseSession(ctx context.Context, sessionId task.SessionId) {
	sessionIdString := sessionId.String()
	_, err := pb.client.CloseSession(ctx, bonkv0.CloseSessionRequest_builder{
		SessionId: &sessionIdString,
	}.Build())
	if err != nil {
		slog.WarnContext(
			ctx,
			"error returned when closing session",
			"plugin",
			pb.Name(),
			"session",
			sessionId.String(),
		)
	}
}

func (pb *grpcClient) Execute(
	ctx context.Context,
	tsk *task.GenericTask,
	result *task.Result,
) error {
	sessionIdStr := tsk.Session.ID().String()
	taskReqBuilder := bonkv0.ExecuteTaskRequest_builder{
		SessionId: &sessionIdStr,
		Name:      &tsk.ID.Name,
		Executor:  &tsk.ID.Executor,
		Inputs:    tsk.Inputs,
	}

	var err error
	taskReqBuilder.Arguments, err = ToProtoValue(tsk.Args)
	if err != nil {
		return fmt.Errorf("failed to encode args to proto: %w", err)
	}

	res, err := pb.client.ExecuteTask(ctx, taskReqBuilder.Build())
	if err != nil {
		return fmt.Errorf("failed to call perform task: %w", err)
	}

	result.Outputs = res.GetOutput()
	result.FollowupTasks = make([]task.GenericTask, len(res.GetFollowupTasks()))
	for ii, followup := range res.GetFollowupTasks() {
		// Create the new task and append it
		result.FollowupTasks[ii] = *task.New(
			tsk.Session,
			followup.GetExecutor(),
			fmt.Sprintf("%s.%s", tsk.ID.Name, followup.GetName()),
			followup.GetArguments().AsInterface(),
			followup.GetInputs()...,
		)
	}

	return nil
}

type grpcServer struct {
	bonkv0.UnimplementedExecutorServiceServer

	name     string
	executor task.GenericExecutor

	sessions map[task.SessionId]task.Session
}

var _ bonkv0.ExecutorServiceServer = (*grpcServer)(nil)

// Creates a GRPC server which forwards incoming task requests to an Executor.
func NewGRPCServer(
	name string,
	executor task.GenericExecutor,
) bonkv0.ExecutorServiceServer {
	return &grpcServer{
		name:     name,
		executor: executor,
		sessions: make(map[task.SessionId]task.Session),
	}
}

func (s *grpcServer) OpenSession(
	ctx context.Context,
	req *bonkv0.OpenSessionRequest,
) (*bonkv0.OpenSessionResponse, error) {
	slog.DebugContext(ctx, "opening session", "session", req.GetSessionId())

	sessionId := uuid.MustParse(req.GetSessionId())
	var session task.Session

	switch req.WhichWorkspaceDescription() {
	case bonkv0.OpenSessionRequest_Local_case:
		session = task.NewLocalSession(sessionId, req.GetLocal().GetAbsolutePath())

	case bonkv0.OpenSessionRequest_Test_case:
		session = task.NewTestSession()

	default:
		return nil, errors.New("unsupported workspace type")
	}

	s.sessions[sessionId] = session

	err := s.executor.OpenSession(ctx, session)
	if err != nil {
		return nil, err //nolint:wrapcheck
	}

	return bonkv0.OpenSessionResponse_builder{}.Build(), nil
}

func (s *grpcServer) CloseSession(
	ctx context.Context,
	req *bonkv0.CloseSessionRequest,
) (*bonkv0.CloseSessionResponse, error) {
	slog.DebugContext(ctx, "closing session", "session", req.GetSessionId())

	sessionId := uuid.MustParse(req.GetSessionId())

	s.executor.CloseSession(ctx, sessionId)

	delete(s.sessions, sessionId)

	return bonkv0.CloseSessionResponse_builder{}.Build(), nil
}

func (s *grpcServer) ExecuteTask(
	ctx context.Context,
	req *bonkv0.ExecuteTaskRequest,
) (*bonkv0.ExecuteTaskResponse, error) {
	// Find the relevant session
	sessionId := uuid.MustParse(req.GetSessionId())
	session, ok := s.sessions[sessionId]
	if !ok {
		return nil, fmt.Errorf("unopened session id: %s", sessionId.String())
	}

	tskId := task.TaskId{
		Name:     req.GetName(),
		Executor: req.GetExecutor(),
	}
	tsk := task.GenericTask{
		ID:      tskId,
		Session: session,
		Inputs:  req.GetInputs(),
		Args:    req.GetArguments().AsInterface(),
	}

	err := tsk.OutputFS().MkdirAll("", 0o750)
	if err != nil {
		return nil, fmt.Errorf("failed to create output directory: %w", err)
	}
	var response task.Result
	err = s.executor.Execute(ctx, &tsk, &response)
	if err != nil {
		return nil, err //nolint:wrapcheck
	}

	res := bonkv0.ExecuteTaskResponse_builder{
		Output:        response.Outputs,
		FollowupTasks: make([]*bonkv0.ExecuteTaskResponse_FollowupTask, len(response.FollowupTasks)),
	}

	for idx, followup := range response.FollowupTasks {
		taskProto := bonkv0.ExecuteTaskResponse_FollowupTask_builder{
			Name:     &followup.ID.Name,
			Executor: &followup.ID.Executor,
			Inputs:   followup.Inputs,
		}

		var newValErr error
		taskProto.Arguments, newValErr = ToProtoValue(followup.Args)

		if multierr.AppendInto(&err, newValErr) {
			continue
		}

		res.FollowupTasks[idx] = taskProto.Build()
	}

	return res.Build(), err
}
