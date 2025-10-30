// Copyright © 2025 Colden Cullen
// SPDX-License-Identifier: MIT

package plugin

import (
	"testing"

	"github.com/stretchr/testify/require"

	goplugin "github.com/hashicorp/go-plugin"

	"go.bonk.build/pkg/executor"
	"go.bonk.build/pkg/executor/router"
	"go.bonk.build/pkg/executor/rpc"
)

// ServeTest sets up a test gRPC connection which serves plugin and returns a client executor.
func (plugin *Plugin) ServeTest(t *testing.T) executor.Executor {
	t.Helper()

	client, server := goplugin.TestPluginGRPCConn(t, false, plugin.getPluginSet())

	pluginClient := rpc.NewGRPCClient(client.Conn)

	t.Cleanup(func() {
		// Close the GRPC infrastructure
		server.Stop()
		require.NoError(t, client.Close())
	})

	rtr := router.New()
	require.NoError(t, rtr.RegisterExecutor(plugin.Name(), pluginClient))

	return &rtr
}
