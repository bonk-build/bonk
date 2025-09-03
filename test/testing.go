// Copyright © 2025 Colden Cullen
// SPDX-License-Identifier: MIT

package test

import (
	"testing"

	"github.com/stretchr/testify/require"

	goplugin "github.com/hashicorp/go-plugin"

	bonk "go.bonk.build/api/go"
	bonkv0 "go.bonk.build/api/proto/bonk/v0"
	"go.bonk.build/pkg/executor"
)

// Call like you'd call Serve() but at the top of your test function.
func ServeTest(t *testing.T, plugin *bonk.Plugin) executor.ExecutorManager {
	t.Helper()

	client, server := goplugin.TestPluginGRPCConn(t, false, map[string]goplugin.Plugin{
		"executor": &bonk.ExecutorServer{
			Executors: &plugin.ExecutorManager,
		},
	})

	t.Cleanup(func() {
		// Close the GRPC infrastructure
		require.NoError(t, client.Close())
		server.Stop()
	})

	raw, err := client.Dispense("executor")
	if err != nil {
		t.Fatal("failed to dispense plugin:", err)
	}

	bonkClient, ok := raw.(bonkv0.ExecutorServiceClient)
	if !ok {
		t.Fatal("plugin dispensed is of the wrong type")
	}

	executorManager := executor.NewExecutorManager()

	plugin.ForEachExecutor(func(name string, _ executor.Executor) {
		err = executorManager.RegisterExecutor(name, executor.NewRPC(name, bonkClient))
		if err != nil {
			t.Fatal("failed to register executor:", name)
		}
	})

	return executorManager
}
