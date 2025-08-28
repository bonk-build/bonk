// Copyright © 2025 Colden Cullen
// SPDX-License-Identifier: MIT

package test

import (
	"testing"

	goplugin "github.com/hashicorp/go-plugin"

	bonk "go.bonk.build/api/go"
	bonkv0 "go.bonk.build/api/go/proto/bonk/v0"
	"go.bonk.build/pkg/executor"
)

// Call like you'd call Serve() but at the top of your test function.
func ServeTest(t *testing.T, executors ...bonk.BonkExecutor) *executor.ExecutorManager {
	t.Helper()

	executorMap := make(map[string]bonk.BonkExecutor, len(executors))
	for _, be := range executors {
		executorMap[be.Name] = be
	}

	client, server := goplugin.TestPluginGRPCConn(t, false, map[string]goplugin.Plugin{
		bonk.PluginType: &bonk.BonkPluginServer{
			Executors: executorMap,
		},
	})

	go func() {
		// Wait for the test to finish
		<-t.Context().Done()

		// Close the GRPC infrastructure
		_ = client.Close()
		server.Stop()
	}()

	raw, err := client.Dispense(bonk.PluginType)
	if err != nil {
		t.Fatal("failed to dispense plugin:", err)
	}

	bonkClient, ok := raw.(bonkv0.BonkPluginServiceClient)
	if !ok {
		t.Fatal("plugin dispensed is of the wrong type")
	}

	executorManager := executor.NewExecutorManager()

	for _, be := range executors {
		err = executorManager.RegisterExecutor(be.Name, executor.NewRPC(be.Name, bonkClient))
		if err != nil {
			t.Fatal("failed to register executor:", be.Name)
		}
	}

	return executorManager
}
