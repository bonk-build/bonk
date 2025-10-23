// Copyright © 2025 Colden Cullen
// SPDX-License-Identifier: MIT

package rpc

import (
	"context"
	"fmt"
	"log/slog"
	"runtime"
	"sync"

	"go.uber.org/multierr"

	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
	"google.golang.org/protobuf/types/known/structpb"
	"google.golang.org/protobuf/types/known/timestamppb"

	"github.com/google/uuid"

	slogmulti "github.com/samber/slog-multi"
	slogctx "github.com/veqryn/slog-context"

	bonkv0 "go.bonk.build/api/bonk/v0"
	"go.bonk.build/pkg/executor"
	"go.bonk.build/pkg/task"
)

type grpcServerSession struct {
	task.Session

	closer chan<- struct{}
	logger *slog.Logger
}

func (s grpcServerSession) LocalPath() string {
	if ls, ok := s.Session.(task.LocalSession); ok {
		return ls.LocalPath()
	}

	return ""
}

type grpcServer struct {
	bonkv0.UnimplementedExecutorServiceServer

	executor executor.Executor

	sessions   map[task.SessionID]grpcServerSession
	sessionsMu sync.RWMutex
}

var _ bonkv0.ExecutorServiceServer = (*grpcServer)(nil)

// RegisterGRPCServer creates a GRPC server which forwards incoming task requests to an Executor.
func RegisterGRPCServer(
	server *grpc.Server,
	executor executor.Executor,
) {
	bonkv0.RegisterExecutorServiceServer(server, &grpcServer{
		executor: executor,
		sessions: make(map[task.SessionID]grpcServerSession),
	})
}

func (s *grpcServer) OpenSession(
	req *bonkv0.OpenSessionRequest,
	stream grpc.ServerStreamingServer[bonkv0.OpenSessionResponse],
) error {
	ctx := stream.Context()
	slog.DebugContext(ctx, "opening session", "session", req.GetSessionId())

	sessionID := uuid.MustParse(req.GetSessionId())
	var session task.Session

	switch req.WhichWorkspaceDescription() {
	case bonkv0.OpenSessionRequest_Local_case:
		session = task.NewLocalSession(sessionID, req.GetLocal().GetAbsolutePath())

	case bonkv0.OpenSessionRequest_Test_case:
		session = task.NewTestSession()

	default:
		return status.Error(codes.InvalidArgument, "unsupported workspace type")
	}

	var logger *slog.Logger
	if req.HasLogStreaming() {
		// Start the logging handler
		logger = slog.New(slogmulti.
			Pipe(
				slogmulti.NewEnabledInlineMiddleware(
					func(ctx context.Context, level slog.Level, next func(context.Context, slog.Level) bool) bool {
						if int(req.GetLogStreaming().GetLevel()) > int(level) {
							return false
						}

						return next(ctx, level)
					},
				),
			).
			Handler(slogmulti.NewHandleInlineHandler(
				func(_ context.Context, _ []string, _ []slog.Attr, record slog.Record) error {
					if req.GetLogStreaming().GetAddSource() {
						fs := runtime.CallersFrames([]uintptr{record.PC})
						f, _ := fs.Next()
						record.AddAttrs(slog.Any(slog.SourceKey, &slog.Source{
							Function: f.Function,
							File:     f.File,
							Line:     f.Line,
						}))
					}

					level := int64(record.Level)
					logInstance := bonkv0.OpenSessionResponse_LogRecord_builder{
						Time:    timestamppb.New(record.Time),
						Message: &record.Message,
						Level:   &level,
						Attrs:   make(map[string]*structpb.Value, record.NumAttrs()),
					}

					record.Attrs(func(attr slog.Attr) bool {
						protoValue, err := ToProtoValue(attr.Value.Any())
						if err != nil {
							panic(err)
						}
						logInstance.Attrs[attr.Key] = protoValue

						return true
					})

					err := stream.Send(bonkv0.OpenSessionResponse_builder{
						LogRecord: logInstance.Build(),
					}.Build())
					if err != nil {
						return fmt.Errorf("failed to send record across gRPC: %w", err)
					}

					return nil
				},
			)),
		)
		ctx = slogctx.NewCtx(ctx, logger)
	}
	err := s.executor.OpenSession(ctx, session)
	if err != nil {
		return err
	}

	err = stream.Send(bonkv0.OpenSessionResponse_builder{
		Ack: &bonkv0.OpenSessionResponse_Ack{},
	}.Build())
	if err != nil {
		return fmt.Errorf("failed to send ack: %w", err)
	}

	closer := make(chan struct{})

	s.sessionsMu.Lock()
	s.sessions[sessionID] = grpcServerSession{
		Session: session,
		closer:  closer,
		logger:  logger,
	}
	s.sessionsMu.Unlock()

	slog.DebugContext(ctx, "successfully opened session", "session", sessionID)

	// Block until the request is canceled/closed
	<-closer

	return nil
}

// CloseSession implements v0.ExecutorServiceServer.
func (s *grpcServer) CloseSession(
	ctx context.Context,
	req *bonkv0.CloseSessionRequest,
) (*bonkv0.CloseSessionResponse, error) {
	// Find the relevant session
	sessionID := uuid.MustParse(req.GetId())

	s.sessionsMu.RLock()
	session, ok := s.sessions[sessionID]
	s.sessionsMu.RUnlock()
	if !ok {
		return nil, status.Errorf(
			codes.InvalidArgument,
			"unopened session id: %s",
			sessionID.String(),
		)
	}

	session.closer <- struct{}{}

	// Close the session
	s.executor.CloseSession(ctx, sessionID)

	s.sessionsMu.Lock()
	delete(s.sessions, sessionID)
	s.sessionsMu.Unlock()

	return &bonkv0.CloseSessionResponse{}, nil
}

func (s *grpcServer) ExecuteTask(
	ctx context.Context,
	req *bonkv0.ExecuteTaskRequest,
) (*bonkv0.ExecuteTaskResponse, error) {
	// Find the relevant session
	sessionID := uuid.MustParse(req.GetSessionId())

	s.sessionsMu.RLock()
	session, ok := s.sessions[sessionID]
	s.sessionsMu.RUnlock()
	if !ok {
		return nil, status.Errorf(
			codes.InvalidArgument,
			"unopened session id: %s",
			sessionID.String(),
		)
	}

	ctx = slogctx.NewCtx(ctx, session.logger)

	tsk := task.Task{
		ID:       task.ID(req.GetId()),
		Executor: req.GetExecutor(),
		Inputs:   req.GetInputs(),
		Args:     req.GetArguments().AsInterface(),
	}

	taskOutputFs := task.OutputFS(session.Session, tsk.ID)

	err := taskOutputFs.MkdirAll("", 0o750)
	if err != nil {
		return nil, status.Errorf(
			codes.Unknown,
			"failed to create output directory: %s",
			err,
		)
	}
	var response task.Result
	err = s.executor.Execute(ctx, session, &tsk, &response)
	if err != nil {
		return nil, status.Error(CodeExecErr, err.Error())
	}

	followups := response.GetFollowupTasks()
	res := bonkv0.ExecuteTaskResponse_builder{
		Output:        response.GetOutputs(),
		FollowupTasks: make([]*bonkv0.ExecuteTaskResponse_FollowupTask, len(followups)),
	}

	for idx, followup := range followups {
		taskProto := bonkv0.ExecuteTaskResponse_FollowupTask_builder{
			Id:       (*string)(&followup.ID),
			Executor: &followup.Executor,
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
