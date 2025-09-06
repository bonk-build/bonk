// Copyright © 2025 Colden Cullen
// SPDX-License-Identifier: MIT

package bonk // import "go.bonk.build/api/go"

import (
	"context"

	"go.uber.org/multierr"

	"google.golang.org/grpc"

	"cuelang.org/go/cue"

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
	Executor                  = task.Executor
	TypedExecutor[Params any] = task.TypedExecutor[Params]
)

func WrapTypedExecutor[Params any](
	cuectx *cue.Context,
	impl TypedExecutor[Params],
) Executor {
	return task.WrapTypedExecutor(cuectx, impl)
}

type executorServer struct {
	goplugin.NetRPCUnsupportedPlugin
	goplugin.GRPCPlugin

	Name      string
	Cuectx    *cue.Context
	Executors *executor.ExecutorManager
}

func (p *executorServer) GRPCServer(_ *goplugin.GRPCBroker, server *grpc.Server) error {
	bonkv0.RegisterExecutorServiceServer(server, executor.NewGRPCServer(
		p.Name,
		p.Cuectx,
		pluginExecutor{
			ExecutorManager: p.Executors,
		},
	))

	return nil
}

func (p *executorServer) GRPCClient(
	_ context.Context,
	_ *goplugin.GRPCBroker,
	c *grpc.ClientConn,
) (any, error) {
	return bonkv0.NewExecutorServiceClient(c), nil
}

type pluginExecutor struct {
	*executor.ExecutorManager
}

var (
	_ task.Executor       = pluginExecutor{}
	_ task.SessionManager = pluginExecutor{}
)

// Override Execute to add some special details to the context.
func (pe pluginExecutor) Execute(ctx context.Context, tsk task.Task, res *task.Result) error {
	execCtx, cleanup, err := getTaskLoggingContext(ctx, tsk.OutputFs)
	if err != nil {
		return err
	}

	// Append executor information
	execCtx = slogctx.Append(execCtx, "executor", tsk.Executor())

	multierr.AppendInto(&err, pe.ExecutorManager.Execute(execCtx, tsk, res))
	multierr.AppendInto(&err, cleanup())

	return err
}
