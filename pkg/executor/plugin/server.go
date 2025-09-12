// Copyright © 2025 Colden Cullen
// SPDX-License-Identifier: MIT

package plugin

import (
	"context"
	"errors"
	"fmt"
	"log/slog"

	"go.uber.org/multierr"

	"google.golang.org/grpc"

	"github.com/ValerySidorin/shclog"

	goplugin "github.com/hashicorp/go-plugin"

	bonkv0 "go.bonk.build/api/proto/bonk/v0"
	"go.bonk.build/pkg/executor/rpc"
	"go.bonk.build/pkg/executor/tree"
	"go.bonk.build/pkg/task"
)

type Plugin struct {
	tree.ExecutorManager

	name string
}

var _ task.GenericExecutor = (*Plugin)(nil)

func NewPlugin(name string, initializer func(plugin *Plugin) error) *Plugin {
	plugin := &Plugin{
		ExecutorManager: tree.NewExecutorManager(),
		name:            name,
	}

	err := initializer(plugin)
	if err != nil {
		panic(fmt.Errorf("failed to initialize plugin: %w", err))
	}

	return plugin
}

func (p *Plugin) Name() string { return p.name }

// Call from main() to start the plugin gRPC server.
func (p *Plugin) Serve() {
	goplugin.Serve(&goplugin.ServeConfig{
		HandshakeConfig: Handshake,
		Plugins: map[string]goplugin.Plugin{
			"executor": &ExecutorServer{
				GenericExecutor: &p.ExecutorManager,
			},
		},
		GRPCServer: goplugin.DefaultGRPCServer,
		Logger:     shclog.New(slog.Default()),
	})
}

type ExecutorServer struct {
	goplugin.NetRPCUnsupportedPlugin
	task.GenericExecutor
}

var (
	_ task.GenericExecutor = (*ExecutorServer)(nil)
	_ goplugin.GRPCPlugin  = (*ExecutorServer)(nil)
)

func (p *ExecutorServer) GRPCServer(_ *goplugin.GRPCBroker, server *grpc.Server) error {
	bonkv0.RegisterExecutorServiceServer(server, rpc.NewGRPCServer(
		p,
	))

	return nil
}

func (*ExecutorServer) GRPCClient(
	_ context.Context,
	_ *goplugin.GRPCBroker,
	c *grpc.ClientConn,
) (any, error) {
	return nil, errors.ErrUnsupported
}

// Override Execute to add some special details to the context.
func (p *ExecutorServer) Execute(
	ctx context.Context,
	tsk *task.GenericTask,
	res *task.Result,
) error {
	ctx, cleanup, err := getTaskLoggingContext(
		ctx,
		tsk,
	)
	if err != nil {
		return err
	}

	multierr.AppendInto(&err, p.GenericExecutor.Execute(ctx, tsk, res))
	multierr.AppendInto(&err, cleanup())

	return err
}
