// Copyright © 2025 Colden Cullen
// SPDX-License-Identifier: MIT

package bonk // import "go.bonk.build/api/go"

import (
	"fmt"
	"log/slog"

	"github.com/ValerySidorin/shclog"

	goplugin "github.com/hashicorp/go-plugin"

	"go.bonk.build/pkg/executor"
	"go.bonk.build/pkg/plugin"
	"go.bonk.build/pkg/task"
)

type Plugin struct {
	executor.ExecutorManager

	EnableLogStreaming bool
}

var _ task.GenericExecutor = (*Plugin)(nil)

func NewPlugin(name string, initializer func(plugin *Plugin) error) *Plugin {
	plugin := &Plugin{
		ExecutorManager:    executor.NewExecutorManager(name),
		EnableLogStreaming: true,
	}

	err := initializer(plugin)
	if err != nil {
		panic(fmt.Errorf("failed to initialize plugin: %w", err))
	}

	return plugin
}

// Call from main() to start the plugin gRPC server.
func (p *Plugin) Serve() {
	const defaultPluginMapSize = 2
	pluginMap := make(map[string]goplugin.Plugin, defaultPluginMapSize)

	pluginMap["executor"] = &executorServer{
		Executors: &p.ExecutorManager,
	}

	if p.EnableLogStreaming {
		pluginMap["log_streaming"] = &LogStreamingServer{}
	}

	goplugin.Serve(&goplugin.ServeConfig{
		HandshakeConfig: plugin.Handshake,
		Plugins:         pluginMap,
		GRPCServer:      goplugin.DefaultGRPCServer,
		Logger:          shclog.New(slog.Default()),
	})
}
