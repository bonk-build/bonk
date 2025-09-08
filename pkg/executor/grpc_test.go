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

	"cuelang.org/go/cue/cuecontext"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	bonkv0 "go.bonk.build/api/proto/bonk/v0"
	"go.bonk.build/pkg/executor"
	"go.bonk.build/pkg/task"
	"go.bonk.build/test"
)

func openConnection(t *testing.T, exec task.Executor) task.Executor {
	t.Helper()

	lis := bufconn.Listen(1024 * 1024)
	s := grpc.NewServer()
	bonkv0.RegisterExecutorServiceServer(s, executor.NewGRPCServer("", cuecontext.New(), exec))

	go func() {
		err := s.Serve(lis)
		assert.NoErrorf(t, err, "Server execited with err: %v", err)
	}()

	bufDialer := func(context.Context, string) (net.Conn, error) {
		return lis.Dial()
	}

	clientConn, err := grpc.NewClient("bufnet",
		grpc.WithContextDialer(bufDialer),
		grpc.WithTransportCredentials(insecure.NewCredentials()),
	)
	require.NoError(t, err)
	client := bonkv0.NewExecutorServiceClient(clientConn)

	return executor.NewGRPCClient("", cuecontext.New(), client)
}

func Test_TestConnection(t *testing.T) {
	t.Parallel()

	mock := gomock.NewController(t)
	exec := test.NewMockExecutor(mock)

	client := openConnection(t, exec)
	require.NotNil(t, client)
}
