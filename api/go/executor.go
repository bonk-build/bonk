// Copyright © 2025 Colden Cullen
// SPDX-License-Identifier: MIT

package bonk // import "go.bonk.build/api/go"

import (
	"context"

	"go.uber.org/multierr"

	"google.golang.org/grpc"

	goplugin "github.com/hashicorp/go-plugin"
	slogctx "github.com/veqryn/slog-context"

	bonkv0 "go.bonk.build/api/proto/bonk/v0"
	"go.bonk.build/pkg/executor"
	"go.bonk.build/pkg/task"
)

type executorServer struct {
	goplugin.NetRPCUnsupportedPlugin
	task.GenericExecutor
}

var (
	_ task.GenericExecutor = (*executorServer)(nil)
	_ goplugin.GRPCPlugin  = (*executorServer)(nil)
)

func (p *executorServer) GRPCServer(_ *goplugin.GRPCBroker, server *grpc.Server) error {
	bonkv0.RegisterExecutorServiceServer(server, executor.NewGRPCServer(
		p.Name(),
		p,
	))

	return nil
}

func (*executorServer) GRPCClient(
	_ context.Context,
	_ *goplugin.GRPCBroker,
	c *grpc.ClientConn,
) (any, error) {
	return bonkv0.NewExecutorServiceClient(c), nil
}

// Override Execute to add some special details to the context.
func (p *executorServer) Execute(
	ctx context.Context,
	tsk *task.GenericTask,
	res *task.Result,
) error {
	execCtx, cleanup, err := getTaskLoggingContext(ctx, tsk.OutputFs)
	if err != nil {
		return err
	}

	// Append executor information
	execCtx = slogctx.Append(execCtx, "executor", tsk.ID.Executor)

	multierr.AppendInto(&err, p.GenericExecutor.Execute(execCtx, tsk, res))
	multierr.AppendInto(&err, cleanup())

	return err
}
