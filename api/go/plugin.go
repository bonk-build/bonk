// Copyright © 2025 Colden Cullen
// SPDX-License-Identifier: MIT

package bonk // import "go.bonk.build/api/go"

import (
	"fmt"
	"log/slog"

	"github.com/ValerySidorin/shclog"

	goplugin "github.com/hashicorp/go-plugin"

	"go.bonk.build/pkg/executor/plugin"
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
		HandshakeConfig: plugin.Handshake,
		Plugins: map[string]goplugin.Plugin{
			"executor": &ExecutorServer{
				GenericExecutor: &p.ExecutorManager,
			},
		},
		GRPCServer: goplugin.DefaultGRPCServer,
		Logger:     shclog.New(slog.Default()),
	})
}
