// Copyright © 2025 Colden Cullen
// SPDX-License-Identifier: MIT

package bonk

import (
	"testing"

	"github.com/stretchr/testify/require"

	goplugin "github.com/hashicorp/go-plugin"

	"go.bonk.build/pkg/executor/rpc"
	"go.bonk.build/pkg/executor/tree"
	"go.bonk.build/pkg/task"
)

// Call like you'd call Serve() but at the top of your test function.
func (plugin *Plugin) ServeTest(t *testing.T) task.GenericExecutor {
	t.Helper()

	client, server := goplugin.TestPluginGRPCConn(t, false, map[string]goplugin.Plugin{
		"executor": &ExecutorServer{
			GenericExecutor: &plugin.ExecutorManager,
		},
	})

	pluginClient := rpc.NewGRPCClient(plugin.Name(), client.Conn)

	t.Cleanup(func() {
		// Close the GRPC infrastructure
		require.NoError(t, client.Close())
		server.Stop()
	})

	executorManager := tree.NewExecutorManager(pluginClient.Name())
	require.NoError(t, executorManager.RegisterExecutors(pluginClient))

	return &executorManager
}
