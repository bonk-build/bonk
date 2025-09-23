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

	"go.bonk.build/pkg/executor/argconv"
	"go.bonk.build/pkg/executor/rpc"
	"go.bonk.build/pkg/executor/tree"
	"go.bonk.build/pkg/task"
)

type Plugin struct {
	tree.ExecutorTree
	goplugin.NetRPCUnsupportedPlugin

	name string
}

var (
	_ task.Executor       = (*Plugin)(nil)
	_ goplugin.GRPCPlugin = (*Plugin)(nil)
)

type PluginOption func(plugin *Plugin) error

func NewPlugin(name string, initializers ...PluginOption) *Plugin {
	plugin := &Plugin{
		ExecutorTree: tree.New(),
		name:         name,
	}

	for _, initializer := range initializers {
		err := initializer(plugin)
		if err != nil {
			panic(fmt.Errorf("failed to initialize plugin: %w", err))
		}
	}

	return plugin
}

func (p *Plugin) Name() string { return p.name }

// WithExecutor registers an executor with the plugin.
func WithExecutor[Params any](name string, exec argconv.TypedExecutor[Params]) PluginOption {
	return func(plugin *Plugin) error {
		return plugin.RegisterExecutor(name, argconv.BoxExecutor(exec))
	}
}

// Serve starts the plugin gRPC server.
func (p *Plugin) Serve() {
	goplugin.Serve(&goplugin.ServeConfig{
		HandshakeConfig: Handshake,
		Plugins:         p.getPluginSet(),
		GRPCServer:      goplugin.DefaultGRPCServer,
		Logger:          shclog.New(slog.Default()),
	})
}

func (p *Plugin) GRPCServer(_ *goplugin.GRPCBroker, server *grpc.Server) error {
	rpc.RegisterGRPCServer(server, p)

	return nil
}

// GRPCClient: Unsupported.
func (*Plugin) GRPCClient(
	_ context.Context,
	_ *goplugin.GRPCBroker,
	c *grpc.ClientConn,
) (any, error) {
	return nil, errors.ErrUnsupported
}

// Execute adds some special details to the context.
func (p *Plugin) Execute(
	ctx context.Context,
	tsk *task.Task,
	res *task.Result,
) error {
	ctx, cleanup, err := getTaskLoggingContext(
		ctx,
		tsk,
	)
	if err != nil {
		return err
	}

	multierr.AppendInto(&err, p.ExecutorTree.Execute(ctx, tsk, res))
	multierr.AppendInto(&err, cleanup())

	return err
}

func (p *Plugin) getPluginSet() goplugin.PluginSet {
	return map[string]goplugin.Plugin{
		"executor": p,
	}
}
