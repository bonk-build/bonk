// Copyright © 2025 Colden Cullen
// SPDX-License-Identifier: MIT

package plugin // import "go.bonk.build/pkg/executor/plugin"

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"os/exec"

	"github.com/ValerySidorin/shclog"

	goplugin "github.com/hashicorp/go-plugin"

	"go.bonk.build/pkg/executor/rpc"
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
		Cmd:             exec.CommandContext(ctx, "go", "run", goCmdPath),
		AllowedProtocols: []goplugin.Protocol{
			goplugin.ProtocolGRPC,
		},
		// Necessary for it to not abort immediately
		VersionedPlugins: map[int]goplugin.PluginSet{
			int(Handshake.ProtocolVersion): {}, //nolint:gosec
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

	plug.GenericExecutor = rpc.NewGRPCClient(grpcClient.Conn)

	return plug, nil
}

func (plugin *PluginClient) Shutdown() {
	plugin.pluginClient.Kill()
}
