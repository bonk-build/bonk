// Copyright © 2025 Colden Cullen
// SPDX-License-Identifier: MIT

package test

import (
	"testing"

	goplugin "github.com/hashicorp/go-plugin"

	bonk "go.bonk.build/api/go"
	bonkv0 "go.bonk.build/api/go/proto/bonk/v0"
	"go.bonk.build/pkg/backend"
)

// Call like you'd call Serve() but at the top of your test function.
func ServeTest(t *testing.T, backends ...bonk.BonkBackend) *backend.BackendManager {
	t.Helper()

	backendMap := make(map[string]bonk.BonkBackend, len(backends))
	for _, be := range backends {
		backendMap[be.Name] = be
	}

	client, server := goplugin.TestPluginGRPCConn(t, false, map[string]goplugin.Plugin{
		bonk.PluginType: &bonk.BonkPluginServer{
			Backends: backendMap,
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

	backendManager := backend.NewBackendManager()

	for _, be := range backends {
		err = backendManager.RegisterBackend(be.Name, backend.NewRPC(be.Name, bonkClient))
		if err != nil {
			t.Fatal("failed to register backend:", be.Name)
		}
	}

	return backendManager
}
