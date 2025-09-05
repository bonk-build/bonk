// Copyright © 2025 Colden Cullen
// SPDX-License-Identifier: MIT

package bonk // import "go.bonk.build/api/go"

import (
	"context"
	"errors"
	"fmt"
	"log/slog"

	"go.uber.org/multierr"

	"google.golang.org/grpc"

	"cuelang.org/go/cue"

	"github.com/google/uuid"
	"github.com/spf13/afero"

	goplugin "github.com/hashicorp/go-plugin"
	slogctx "github.com/veqryn/slog-context"

	bonkv0 "go.bonk.build/api/proto/bonk/v0"
	"go.bonk.build/pkg/executor"
	"go.bonk.build/pkg/task"
)

// Re-exports for cleanliness.
type (
	Task                      = task.Task
	TypedTask[Params any]     = task.TypedTask[Params]
	Result                    = task.Result
	Executor                  = executor.Executor
	TypedExecutor[Params any] = executor.TypedExecutor[Params]
)

func WrapTypedExecutor[Params any](
	cuectx *cue.Context,
	impl TypedExecutor[Params],
) Executor {
	return executor.WrapTypedExecutor(cuectx, impl)
}

// PRIVATE

type ExecutorServer struct {
	goplugin.NetRPCUnsupportedPlugin
	goplugin.GRPCPlugin

	Name      string
	Cuectx    *cue.Context
	Executors *executor.ExecutorManager
}

func (p *ExecutorServer) GRPCServer(_ *goplugin.GRPCBroker, server *grpc.Server) error {
	bonkv0.RegisterExecutorServiceServer(server, &executorGRPCServer{
		name:      p.Name,
		cuectx:    p.Cuectx,
		executors: p.Executors,
		sessions:  make(map[uuid.UUID]task.DefaultSession),
	})

	return nil
}

func (p *ExecutorServer) GRPCClient(
	_ context.Context,
	_ *goplugin.GRPCBroker,
	c *grpc.ClientConn,
) (any, error) {
	return bonkv0.NewExecutorServiceClient(c), nil
}

// Here is the gRPC server that GRPCClient talks to.
type executorGRPCServer struct {
	bonkv0.UnimplementedExecutorServiceServer

	name      string
	cuectx    *cue.Context
	executors *executor.ExecutorManager

	sessions map[uuid.UUID]task.DefaultSession
}

func (s *executorGRPCServer) DescribeExecutors(
	ctx context.Context,
	req *bonkv0.DescribeExecutorsRequest,
) (*bonkv0.DescribeExecutorsResponse, error) {
	slog.DebugContext(ctx, "configuring plugin")

	respBuilder := bonkv0.DescribeExecutorsResponse_builder{
		PluginName: &s.name,
		Executors: make(
			map[string]*bonkv0.DescribeExecutorsResponse_ExecutorDescription,
			s.executors.GetNumExecutors(),
		),
	}

	s.executors.ForEachExecutor(func(name string, _ executor.Executor) {
		respBuilder.Executors[name] = bonkv0.DescribeExecutorsResponse_ExecutorDescription_builder{}.Build()
	})

	return respBuilder.Build(), nil
}

func (s *executorGRPCServer) OpenSession(
	ctx context.Context,
	req *bonkv0.OpenSessionRequest,
) (*bonkv0.OpenSessionResponse, error) {
	slog.DebugContext(ctx, "opening session", "session", req.GetSessionId())

	sessionId := uuid.MustParse(req.GetSessionId())
	var sessionFs afero.Fs

	switch req.WhichWorkspaceDescription() {
	case bonkv0.OpenSessionRequest_Local_case:
		sessionFs = afero.NewBasePathFs(afero.NewOsFs(), req.GetLocal().GetAbsolutePath())

	default:
		return nil, errors.New("unsupported workspace type")
	}

	s.sessions[sessionId] = task.DefaultSession{
		Id: sessionId,
		Fs: sessionFs,
	}

	return bonkv0.OpenSessionResponse_builder{}.Build(), nil
}

func (s *executorGRPCServer) CloseSession(
	ctx context.Context,
	req *bonkv0.CloseSessionRequest,
) (*bonkv0.CloseSessionResponse, error) {
	slog.DebugContext(ctx, "closing session", "session", req.GetSessionId())

	sessionId := uuid.MustParse(req.GetSessionId())
	delete(s.sessions, sessionId)

	return bonkv0.CloseSessionResponse_builder{}.Build(), nil
}

func (s *executorGRPCServer) ExecuteTask(
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
	tsk := task.Task{
		ID:      tskId,
		Session: &session,
		Inputs:  req.GetInputs(),

		OutputFs: afero.NewBasePathFs(session.FS(), tskId.GetOutDirectory()),
	}

	err := tsk.OutputFs.MkdirAll("", 0o750)
	if err != nil {
		return nil, fmt.Errorf("failed to create output directory: %w", err)
	}

	tsk.Params = s.cuectx.Encode(req.GetParameters())
	if tsk.Params.Err() != nil {
		return nil, fmt.Errorf("failed to decode parameters: %w", err)
	}

	execCtx, cleanup, err := getTaskLoggingContext(ctx, tsk.OutputFs)
	if err != nil {
		return nil, err
	}

	// Append executor information
	execCtx = slogctx.Append(execCtx, "executor", req.GetExecutor())

	var response task.Result
	multierr.AppendInto(&err, s.executors.Execute(execCtx, tsk, &response))
	multierr.AppendFunc(&err, cleanup)
	if err != nil {
		return nil, fmt.Errorf("failed to execute task: %w", err)
	}

	res := bonkv0.ExecuteTaskResponse_builder{
		Output:        response.Outputs,
		FollowupTasks: make([]*bonkv0.ExecuteTaskResponse_FollowupTask, len(response.FollowupTasks)),
	}

	for idx, followup := range response.FollowupTasks {
		taskProto := bonkv0.ExecuteTaskResponse_FollowupTask_builder{}
		taskProto.Executor = &followup.ID.Executor
		taskProto.Inputs = followup.Inputs
		err := followup.Params.Decode(taskProto.Parameters)
		if err != nil {
			slog.ErrorContext(ctx, "cannot enqueue followup task as params cue failed to decode to proto")

			continue
		}

		res.FollowupTasks[idx] = taskProto.Build()
	}

	return res.Build(), nil
}
