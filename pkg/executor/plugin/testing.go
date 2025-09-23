// Copyright © 2025 Colden Cullen
// SPDX-License-Identifier: MIT

package plugin

import (
	"testing"

	"github.com/stretchr/testify/require"

	goplugin "github.com/hashicorp/go-plugin"

	"go.bonk.build/pkg/executor/rpc"
	"go.bonk.build/pkg/executor/tree"
	"go.bonk.build/pkg/task"
)

// ServeTest sets up a test gRPC connection which serves plugin and returns a client executor.
func (plugin *Plugin) ServeTest(t *testing.T) task.Executor {
	t.Helper()

	client, server := goplugin.TestPluginGRPCConn(t, false, plugin.getPluginSet())

	pluginClient := rpc.NewGRPCClient(client.Conn)

	t.Cleanup(func() {
		// Close the GRPC infrastructure
		require.NoError(t, client.Close())
		server.Stop()
	})

	executorManager := tree.New()
	require.NoError(t, executorManager.RegisterExecutor(plugin.Name(), pluginClient))

	return &executorManager
}
