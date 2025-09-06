// Copyright © 2025 Colden Cullen
// SPDX-License-Identifier: MIT

package test

import (
	"testing"

	"github.com/google/uuid"
	"github.com/spf13/afero"

	bonk "go.bonk.build/api/go"
	"go.bonk.build/pkg/executor"
	"go.bonk.build/pkg/task"
)

//go:generate go tool mockgen -destination task_mock.go -package test -copyright_file ../license-header.txt -typed ../pkg/task Executor,SessionManager

// Call like you'd call Serve() but at the top of your test function.
func ServeTest(t *testing.T, plugin *bonk.Plugin) task.Executor {
	t.Helper()

	// client, server := goplugin.TestPluginGRPCConn(t, false, map[string]goplugin.Plugin{
	// 	"executor": &bonk.ExecutorServer{
	// 		Cuectx:    cuecontext.New(),
	// 		Executors: &plugin.ExecutorManager,
	// 	},
	// })

	// raw, err := client.Dispense("executor")
	// if err != nil {
	// 	t.Fatal("failed to dispense plugin:", err)
	// }

	// bonkClient, ok := raw.(bonkv0.ExecutorServiceClient)
	// if !ok {
	// 	t.Fatal("plugin dispensed is of the wrong type")
	// }

	executorManager := executor.NewExecutorManager(plugin.Name())

	err := executorManager.RegisterExecutors(&plugin.ExecutorManager)
	if err != nil {
		t.Fatal("failed to register executor:", plugin.Name())
	}

	// plugin.ForEachExecutor(func(name string, _ executor.Executor) {
	// 	err = executorManager.RegisterExecutor(name, executor.NewRPC(name, bonkClient))
	// 	if err != nil {
	// 		t.Fatal("failed to register executor:", name)
	// 	}
	// })

	// t.Cleanup(func() {
	// 	// Close the GRPC infrastructure
	// 	require.NoError(t, client.Close())
	// 	server.Stop()
	// })

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
