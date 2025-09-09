// Copyright © 2025 Colden Cullen
// SPDX-License-Identifier: MIT

package test

import (
	"testing"

	"github.com/google/uuid"
	"github.com/spf13/afero"
	"github.com/stretchr/testify/require"

	goplugin "github.com/hashicorp/go-plugin"

	bonk "go.bonk.build/api/go"
	bonkv0 "go.bonk.build/api/proto/bonk/v0"
	"go.bonk.build/pkg/executor"
	"go.bonk.build/pkg/task"
)

//go:generate go tool mockgen -destination task_mock.go -package test -copyright_file ../license-header.txt -typed ../pkg/task Executor

// Call like you'd call Serve() but at the top of your test function.
func ServeTest(t *testing.T, pluginServer *bonk.Plugin) task.GenericExecutor {
	t.Helper()

	client, server := goplugin.TestPluginGRPCConn(t, false, map[string]goplugin.Plugin{
		"executor": &bonk.ExecutorServer{
			GenericExecutor: &pluginServer.ExecutorManager,
		},
	})

	executorPlugin, err := client.Dispense("executor")
	require.NoError(t, err)

	executorClient, ok := executorPlugin.(bonkv0.ExecutorServiceClient)
	require.Truef(
		t,
		ok,
		"plugin reports supporting executors but client returned was of the wrong type",
	)

	pluginClient := executor.NewGRPCClient(pluginServer.Name(), executorClient)

	t.Cleanup(func() {
		// Close the GRPC infrastructure
		require.NoError(t, client.Close())
		server.Stop()
	})

	executorManager := executor.NewExecutorManager(pluginClient.Name())
	require.NoError(t, executorManager.RegisterExecutors(pluginClient))

	return &executorManager
}

type testSession struct {
	memmapFs afero.MemMapFs
}

func NewTestSession() task.Session {
	return &testSession{
		memmapFs: afero.MemMapFs{},
	}
}

func (ts *testSession) ID() task.SessionId {
	return uuid.Nil
}

func (ts *testSession) FS() afero.Fs {
	return &ts.memmapFs
}
