// Copyright © 2025 Colden Cullen
// SPDX-License-Identifier: MIT

package rpc

import (
	"context"
	"errors"
	"fmt"
	"io"
	"log/slog"
	"runtime"

	"go.uber.org/multierr"

	"google.golang.org/grpc"
	"google.golang.org/protobuf/types/known/structpb"
	"google.golang.org/protobuf/types/known/timestamppb"

	"github.com/google/uuid"

	slogmulti "github.com/samber/slog-multi"
	slogctx "github.com/veqryn/slog-context"

	bonkv0 "go.bonk.build/api/bonk/v0"
	"go.bonk.build/pkg/task"
)

type grpcServerSession struct {
	task.Session

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

	executor task.GenericExecutor

	sessions map[task.SessionId]grpcServerSession
}

var _ bonkv0.ExecutorServiceServer = (*grpcServer)(nil)

// Creates a GRPC server which forwards incoming task requests to an Executor.
func NewGRPCServer(
	executor task.GenericExecutor,
) bonkv0.ExecutorServiceServer {
	return &grpcServer{
		executor: executor,
		sessions: make(map[task.SessionId]grpcServerSession),
	}
}

func (s *grpcServer) OpenSession(
	stream grpc.BidiStreamingServer[bonkv0.OpenSessionRequest, bonkv0.OpenSessionResponse],
) error {
	ctx := stream.Context()

	req, err := stream.Recv()
	if err != nil {
		return fmt.Errorf("failed to open session: %w", err)
	}
	slog.DebugContext(ctx, "opening session", "session", req.GetSessionId())

	sessionId := uuid.MustParse(req.GetSessionId())
	var session task.Session

	switch req.WhichWorkspaceDescription() {
	case bonkv0.OpenSessionRequest_Local_case:
		session = task.NewLocalSession(sessionId, req.GetLocal().GetAbsolutePath())

	case bonkv0.OpenSessionRequest_Test_case:
		session = task.NewTestSession()

	default:
		return errors.New("unsupported workspace type")
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
				func(ctx context.Context, groups []string, attrs []slog.Attr, record slog.Record) error {
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
						} else {
							logInstance.Attrs[attr.Key] = protoValue
						}

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

	s.sessions[sessionId] = grpcServerSession{
		Session: session,
		logger:  logger,
	}
	err = s.executor.OpenSession(ctx, session)
	if err != nil {
		return err //nolint:wrapcheck
	}

	err = stream.Send(bonkv0.OpenSessionResponse_builder{
		Ack: &bonkv0.OpenSessionResponse_Ack{},
	}.Build())
	if err != nil {
		return fmt.Errorf("failed to send ack: %w", err)
	}

	slog.DebugContext(ctx, "successfully opened session", "session", sessionId)

	// Block until the request is canceled/closed
	_, err = stream.Recv()
	if !errors.Is(err, io.EOF) {
		err = fmt.Errorf("expected eof, got: %w", err)
	}

	// Close the session
	s.executor.CloseSession(ctx, sessionId)
	delete(s.sessions, sessionId)

	return err
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

	ctx = slogctx.NewCtx(ctx, session.logger)

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
