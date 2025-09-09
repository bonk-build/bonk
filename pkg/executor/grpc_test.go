// Copyright © 2025 Colden Cullen
// SPDX-License-Identifier: MIT

package executor_test

import (
	"context"
	"net"
	"testing"

	"go.uber.org/mock/gomock"

	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
	"google.golang.org/grpc/test/bufconn"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	bonkv0 "go.bonk.build/api/proto/bonk/v0"
	"go.bonk.build/pkg/executor"
	"go.bonk.build/pkg/task"
	"go.bonk.build/test"
)

func openConnection(t *testing.T, exec task.GenericExecutor) task.GenericExecutor {
	t.Helper()

	lis := bufconn.Listen(1024 * 1024)
	s := grpc.NewServer()
	bonkv0.RegisterExecutorServiceServer(s, executor.NewGRPCServer("", exec))

	go func() {
		err := s.Serve(lis)
		assert.NoErrorf(t, err, "Server execited with err: %v", err)
	}()

	bufDialer := func(context.Context, string) (net.Conn, error) {
		return lis.Dial()
	}

	clientConn, err := grpc.NewClient("passthrough:///bufnet",
		grpc.WithContextDialer(bufDialer),
		grpc.WithTransportCredentials(insecure.NewCredentials()),
	)
	require.NoError(t, err)
	client := bonkv0.NewExecutorServiceClient(clientConn)

	return executor.NewGRPCClient("", client)
}

func Test_TestConnection(t *testing.T) {
	t.Parallel()

	mock := gomock.NewController(t)
	exec := test.NewMockExecutor[any](mock)

	client := openConnection(t, exec)
	require.NotNil(t, client)
}

func Test_Args(t *testing.T) {
	t.Parallel()

	type Args struct {
		Value int
	}

	mock := gomock.NewController(t)
	exec := test.NewMockExecutor[Args](mock)
	session := test.NewTestSession()

	client := openConnection(t, task.BoxExecutor(exec))
	require.NotNil(t, client)

	var result task.Result

	ssm, ok := client.(task.SessionManager)
	require.True(t, ok)

	err := ssm.OpenSession(t.Context(), session)
	require.NoError(t, err)
	defer ssm.CloseSession(t.Context(), session.ID())

	exec.EXPECT().Execute(gomock.Any(), gomock.Any(), gomock.Any()).
		Times(1).
		Return(nil)

	err = client.Execute(t.Context(), task.New(
		session,
		"test.task",
		"test.exec",
		Args{
			Value: 3,
		},
	).Box(), &result)
	require.NoError(t, err)
}
