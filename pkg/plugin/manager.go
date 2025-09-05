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
	"sync"

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
	mu      sync.RWMutex
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
	err = plug.Configure(ctx, rpcClient, pm.executor)
	if err != nil {
		return fmt.Errorf("failed to create plugin %s: %w", pluginName, err)
	}

	pm.mu.Lock()
	pm.plugins[pluginName] = plug
	pm.mu.Unlock()

	return nil
}

func (pm *PluginManager) StartPlugins(ctx context.Context, pluginPath ...string) error {
	var (
		pluginWaiter sync.WaitGroup
		allErrs      error
		errMu        sync.Mutex
	)

	for _, plugin := range pluginPath {
		pluginWaiter.Add(1)
		go func() {
			err := pm.StartPlugin(ctx, plugin)

			errMu.Lock()
			multierr.AppendInto(&allErrs, err)
			errMu.Unlock()

			pluginWaiter.Done()
		}()
	}

	pluginWaiter.Wait()

	return allErrs
}

func (pm *PluginManager) Shutdown() {
	pm.mu.RLock()
	for pluginName := range pm.plugins {
		pm.executor.UnregisterExecutor(pluginName)
	}
	pm.mu.RUnlock()

	pm.mu.Lock()
	pm.plugins = make(map[string]Plugin)
	pm.mu.Unlock()

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
