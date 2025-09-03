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
)

type Plugin struct {
	executor.ExecutorManager

	EnableLogStreaming bool
	Cuectx             *cue.Context
}

func NewPlugin(initializer func(plugin *Plugin) error) *Plugin {
	plugin := &Plugin{
		ExecutorManager:    executor.NewExecutorManager(),
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
func (plugin *Plugin) Serve() {
	const defaultPluginMapSize = 2
	pluginMap := make(map[string]goplugin.Plugin, defaultPluginMapSize)

	if plugin.GetNumExecutors() != 0 {
		pluginMap["executor"] = &ExecutorServer{
			Executors: &plugin.ExecutorManager,
		}
	}

	if plugin.EnableLogStreaming {
		pluginMap["log_streaming"] = &LogStreamingServer{}
	}

	goplugin.Serve(&goplugin.ServeConfig{
		HandshakeConfig: Handshake,
		Plugins:         pluginMap,
		GRPCServer:      goplugin.DefaultGRPCServer,
		Logger:          shclog.New(slog.Default()),
	})
}

var Handshake = goplugin.HandshakeConfig{
	ProtocolVersion:  0,
	MagicCookieKey:   "BONK_PLUGIN",
	MagicCookieValue: "bonk the builder",
}
