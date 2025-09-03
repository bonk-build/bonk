// Copyright © 2025 Colden Cullen
// SPDX-License-Identifier: MIT

package plugin // import "go.bonk.build/pkg/plugin"

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"os/exec"
	"path"

	"go.uber.org/multierr"

	"google.golang.org/grpc"

	"github.com/ValerySidorin/shclog"

	goplugin "github.com/hashicorp/go-plugin"

	bonkv0 "go.bonk.build/api/proto/bonk/v0"
	"go.bonk.build/pkg/executor"
)

type ExecutorRegistrar interface {
	RegisterExecutor(name string, impl executor.Executor) error
	UnregisterExecutor(name string)
}

type PluginManager struct {
	plugins map[string]Plugin

	executor ExecutorRegistrar
}

func NewPluginManager(executor ExecutorRegistrar) *PluginManager {
	pm := &PluginManager{}
	pm.plugins = make(map[string]Plugin)
	pm.executor = executor

	return pm
}

func (pm *PluginManager) StartPlugin(ctx context.Context, pluginPath string) error {
	pluginName := path.Base(pluginPath)

	process := goplugin.NewClient(&goplugin.ClientConfig{
		HandshakeConfig: Handshake,
		Plugins: map[string]goplugin.Plugin{
			"executor":      &executorPluginClient{},
			"log_streaming": &logStreamingPluginClient{},
		},
		Cmd:     exec.CommandContext(ctx, "go", "run", pluginPath),
		Managed: true,
		AllowedProtocols: []goplugin.Protocol{
			goplugin.ProtocolGRPC,
		},
		Logger: shclog.New(slog.Default()),
	})

	rpcClient, err := process.Client()
	if err != nil {
		return fmt.Errorf("failed to create client: %w", err)
	}

	plug := Plugin{
		name: pluginName,
	}
	err = plug.Configure(ctx, rpcClient)
	if err != nil {
		return fmt.Errorf("failed to create plugin %s: %w", pluginName, err)
	}

	pm.plugins[pluginName] = plug

	for executorName, executor := range plug.executors {
		multierr.AppendInto(&err,
			pm.executor.RegisterExecutor(fmt.Sprintf("%s:%s", pluginName, executorName), executor),
		)
	}

	if err != nil {
		return fmt.Errorf("failed to register plugin %s executors: %w", pluginName, err)
	}

	return nil
}

func (pm *PluginManager) Shutdown() {
	for pluginName, plugin := range pm.plugins {
		for executorName := range plugin.executors {
			pm.executor.UnregisterExecutor(fmt.Sprintf("%s:%s", pluginName, executorName))
		}
	}
	pm.plugins = make(map[string]Plugin)

	goplugin.CleanupClients()
}

// Plugin Client

type executorPluginClient struct {
	goplugin.NetRPCUnsupportedPlugin
}

var _ goplugin.GRPCPlugin = (*executorPluginClient)(nil)

func (p *executorPluginClient) GRPCServer(*goplugin.GRPCBroker, *grpc.Server) error {
	return errors.ErrUnsupported
}

func (p *executorPluginClient) GRPCClient(
	_ context.Context,
	_ *goplugin.GRPCBroker,
	c *grpc.ClientConn,
) (any, error) {
	return bonkv0.NewExecutorServiceClient(c), nil
}

type logStreamingPluginClient struct {
	goplugin.NetRPCUnsupportedPlugin
}

var _ goplugin.GRPCPlugin = (*logStreamingPluginClient)(nil)

func (p *logStreamingPluginClient) GRPCServer(*goplugin.GRPCBroker, *grpc.Server) error {
	return errors.ErrUnsupported
}

func (p *logStreamingPluginClient) GRPCClient(
	_ context.Context,
	_ *goplugin.GRPCBroker,
	c *grpc.ClientConn,
) (any, error) {
	return bonkv0.NewLogStreamingServiceClient(c), nil
}
