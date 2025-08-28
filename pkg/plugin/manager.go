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

	"github.com/ValerySidorin/shclog"

	goplugin "github.com/hashicorp/go-plugin"

	plugin "go.bonk.build/api/go"
	bonkv0 "go.bonk.build/api/go/proto/bonk/v0"
	"go.bonk.build/pkg/executor"
)

type ExecutorRegistrar interface {
	RegisterExecutor(name string, impl executor.Executor) error
	UnregisterExecutor(name string)
}

type PluginManager struct {
	plugins map[string]*Plugin

	executor ExecutorRegistrar
}

func NewPluginManager(executor ExecutorRegistrar) *PluginManager {
	pm := &PluginManager{}
	pm.plugins = make(map[string]*Plugin)
	pm.executor = executor

	return pm
}

func (pm *PluginManager) StartPlugin(ctx context.Context, pluginPath string) error {
	pluginName := path.Base(pluginPath)

	process := goplugin.NewClient(&goplugin.ClientConfig{
		HandshakeConfig: plugin.Handshake,
		Plugins: map[string]goplugin.Plugin{
			plugin.PluginType: &bonkPluginClient{},
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

	pluginClient, err := rpcClient.Dispense(plugin.PluginType)
	if err != nil {
		return fmt.Errorf("failed to dispense bonk plugin: %w", err)
	}

	bonkClient, ok := pluginClient.(bonkv0.BonkPluginServiceClient)
	if !ok {
		return errors.New("got unexpected plugin client type")
	}

	plug, err := NewPlugin(ctx, pluginName, bonkClient)
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
	pm.plugins = make(map[string]*Plugin)

	goplugin.CleanupClients()
}
