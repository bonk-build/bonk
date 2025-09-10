// Copyright © 2025 Colden Cullen
// SPDX-License-Identifier: MIT

package plugin

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"os/exec"
	"path"

	"github.com/ValerySidorin/shclog"

	goplugin "github.com/hashicorp/go-plugin"

	"go.bonk.build/pkg/executor"
	"go.bonk.build/pkg/task"
)

var Handshake = goplugin.HandshakeConfig{
	ProtocolVersion:  0,
	MagicCookieKey:   "BONK_PLUGIN",
	MagicCookieValue: "bonk the builder",
}

type PluginClient struct {
	task.GenericExecutor

	pluginClient *goplugin.Client
}

var _ task.GenericExecutor = (*PluginClient)(nil)

func NewPluginClient(ctx context.Context, goCmdPath string) (*PluginClient, error) {
	client := goplugin.NewClient(&goplugin.ClientConfig{
		HandshakeConfig: Handshake,
		Plugins: map[string]goplugin.Plugin{
			"log_streaming": &logStreamingPluginClient{},
		},
		Cmd: exec.CommandContext(ctx, "go", "run", goCmdPath),
		AllowedProtocols: []goplugin.Protocol{
			goplugin.ProtocolGRPC,
		},
		Logger: shclog.New(slog.Default()),
	})

	plug := &PluginClient{
		pluginClient: client,
	}

	rpcClient, err := client.Client()
	if err != nil {
		return nil, fmt.Errorf("failed to create client: %w", err)
	}

	grpcClient, ok := rpcClient.(*goplugin.GRPCClient)
	if !ok {
		panic(errors.New("rpcclient is of the wrong type"))
	}

	pluginName := path.Base(goCmdPath)
	plug.GenericExecutor = executor.NewGRPCClient(pluginName, grpcClient.Conn)
	err = plug.handleFeatureLogStreaming(ctx, pluginName, rpcClient)
	if err != nil {
		slog.DebugContext(ctx, "plugin does not support log streaming, skipping", "plugin", pluginName)
	}

	return plug, nil
}

func (plugin *PluginClient) Shutdown() {
	plugin.pluginClient.Kill()
}
