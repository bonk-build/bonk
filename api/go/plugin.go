// Copyright © 2025 Colden Cullen
// SPDX-License-Identifier: MIT

package bonk // import "go.bonk.build/api/go"

import (
	"fmt"
	"log/slog"

	"cuelang.org/go/cue"
	"cuelang.org/go/cue/cuecontext"

	"github.com/ValerySidorin/shclog"

	goplugin "github.com/hashicorp/go-plugin"

	"go.bonk.build/pkg/executor"
	"go.bonk.build/pkg/plugin"
	"go.bonk.build/pkg/task"
)

type Plugin struct {
	executor.ExecutorManager

	EnableLogStreaming bool
	Cuectx             *cue.Context
}

var (
	_ task.Executor       = (*Plugin)(nil)
	_ task.SessionManager = (*Plugin)(nil)
)

func NewPlugin(name string, initializer func(plugin *Plugin) error) *Plugin {
	plugin := &Plugin{
		ExecutorManager:    executor.NewExecutorManager(name),
		EnableLogStreaming: true,
		Cuectx:             cuecontext.New(),
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

	if p.GetNumExecutors() != 0 {
		pluginMap["executor"] = &ExecutorServer{
			Name:      p.Name(),
			Executors: &p.ExecutorManager,
			Cuectx:    p.Cuectx,
		}
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
