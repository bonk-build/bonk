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

	"go.bonk.build/pkg/executor"
	"go.bonk.build/pkg/executor/argconv"
	"go.bonk.build/pkg/executor/router"
	"go.bonk.build/pkg/executor/rpc"
	"go.bonk.build/pkg/task"
)

// Plugin describes a plugin and the services it provides.
type Plugin struct {
	router.Router
	goplugin.NetRPCUnsupportedPlugin

	name string
}

var (
	_ executor.Executor   = (*Plugin)(nil)
	_ goplugin.GRPCPlugin = (*Plugin)(nil)
)

// PluginOption is a modifier for the plugin.
type PluginOption func(plugin *Plugin) error

// NewPlugin creates a new [Plugin] from the given options.
func NewPlugin(name string, initializers ...PluginOption) *Plugin {
	plugin := &Plugin{
		Router: router.New(),
		name:   name,
	}

	for _, initializer := range initializers {
		err := initializer(plugin)
		if err != nil {
			panic(fmt.Errorf("failed to initialize plugin: %w", err))
		}
	}

	return plugin
}

// Name returns the plugin's name.
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
		HandshakeConfig: handshake,
		Plugins:         p.getPluginSet(),
		GRPCServer:      goplugin.DefaultGRPCServer,
		Logger:          shclog.New(slog.Default()),
	})
}

// GRPCServer calls [rpc.RegisterGRPCServer] for the plugin.
func (p *Plugin) GRPCServer(_ *goplugin.GRPCBroker, server *grpc.Server) error {
	rpc.RegisterGRPCServer(server, p)

	return nil
}

// GRPCClient is unsupported.
func (*Plugin) GRPCClient(
	context.Context,
	*goplugin.GRPCBroker,
	*grpc.ClientConn,
) (any, error) {
	return nil, errors.ErrUnsupported
}

// Execute adds some special details to the context.
func (p *Plugin) Execute(
	ctx context.Context,
	session task.Session,
	tsk *task.Task,
	res *task.Result,
) error {
	ctx, cleanup, err := getTaskLoggingContext(
		ctx,
		session,
		tsk,
	)
	if err != nil {
		return err
	}

	multierr.AppendInto(&err, p.Router.Execute(ctx, session, tsk, res))
	multierr.AppendInto(&err, cleanup())

	return err
}

func (p *Plugin) getPluginSet() goplugin.PluginSet {
	return map[string]goplugin.Plugin{
		"executor": p,
	}
}
