// Copyright © 2025 Colden Cullen
// SPDX-License-Identifier: MIT

package rpc_test

import (
	"context"
	"net"
	"reflect"
	"strings"
	"testing"

	"golang.org/x/sync/errgroup"

	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
	"google.golang.org/grpc/test/bufconn"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"

	"go.bonk.build/pkg/executor"
	"go.bonk.build/pkg/executor/argconv"
	"go.bonk.build/pkg/executor/mockexec"
	"go.bonk.build/pkg/executor/rpc"
	"go.bonk.build/pkg/task"
)

type Args struct {
	Value int
}

type rpcSuite struct {
	exec         *mockexec.MockExecutor
	grpcServer   *grpc.Server
	grpcClient   executor.Executor
	session      task.Session
	serverWaiter errgroup.Group
}

func (s *rpcSuite) SetupTest(t *testing.T) {
	t.Helper()

	s.exec = mockexec.NewMockExecutor(t)

	lis := bufconn.Listen(1024 * 1024)
	s.grpcServer = grpc.NewServer()
	rpc.RegisterGRPCServer(s.grpcServer, s.exec)

	s.serverWaiter.Go(func() error {
		return s.grpcServer.Serve(lis)
	})

	bufDialer := func(context.Context, string) (net.Conn, error) {
		return lis.Dial()
	}

	clientConn, err := grpc.NewClient("passthrough:///bufnet",
		grpc.WithContextDialer(bufDialer),
		grpc.WithTransportCredentials(insecure.NewCredentials()),
	)
	require.NoError(t, err)

	s.grpcClient = rpc.NewGRPCClient(clientConn)

	s.session = task.NewTestSession()
}

func (s *rpcSuite) AfterTest(t *testing.T) {
	t.Helper()

	s.grpcServer.GracefulStop()
	err := s.serverWaiter.Wait()
	if err != nil {
		require.ErrorIs(t, err, grpc.ErrServerStopped)
	}
}

func (s *rpcSuite) Test_Connection(t *testing.T) {
	t.Parallel()

	assert.NotNil(t, s.grpcClient)
}

func (s *rpcSuite) Test_Session(t *testing.T) {
	t.Parallel()

	s.exec.EXPECT().OpenSession(mock.Anything, mock.Anything)
	s.exec.EXPECT().CloseSession(mock.Anything, s.session.ID())

	err := s.grpcClient.OpenSession(t.Context(), s.session)
	require.NoError(t, err)
	defer s.grpcClient.CloseSession(t.Context(), s.session.ID())
}

func (s *rpcSuite) Test_Session_Fail(t *testing.T) {
	t.Parallel()

	s.exec.EXPECT().OpenSession(mock.Anything, mock.Anything).Return(assert.AnError)

	err := s.grpcClient.OpenSession(t.Context(), s.session)
	require.ErrorContains(t, err, assert.AnError.Error())
}

func (s *rpcSuite) Test_Args(t *testing.T) {
	t.Parallel()

	var result task.Result

	s.exec.EXPECT().OpenSession(mock.Anything, mock.Anything)
	s.exec.EXPECT().CloseSession(mock.Anything, s.session.ID())

	err := s.grpcClient.OpenSession(t.Context(), s.session)
	require.NoError(t, err)
	defer s.grpcClient.CloseSession(t.Context(), s.session.ID())

	s.exec.EXPECT().Execute(mock.Anything, mock.Anything, mock.Anything, mock.Anything).Return(nil)

	err = s.grpcClient.Execute(t.Context(), s.session, task.New(
		"test.task",
		"test.exec",
		Args{
			Value: 3,
		},
	), &result)
	require.NoError(t, err)
}

func (s *rpcSuite) Test_Followups(t *testing.T) {
	t.Parallel()

	var result task.Result

	s.exec.EXPECT().OpenSession(mock.Anything, mock.Anything)
	s.exec.EXPECT().CloseSession(mock.Anything, s.session.ID())

	err := s.grpcClient.OpenSession(t.Context(), s.session)
	require.NoError(t, err)
	defer s.grpcClient.CloseSession(t.Context(), s.session.ID())

	expectedTask := task.New(
		task.ID("Test.Task"),
		"Test.Executor",
		Args{
			Value: 69420,
		},
		task.WithInputs(
			"File1.txt",
			"File2.txt",
		),
	)

	s.exec.EXPECT().Execute(mock.Anything, mock.Anything, mock.Anything, mock.Anything).
		Run(func(_ context.Context, session task.Session, _ *task.Task, res *task.Result) {
			res.AddFollowupTasks(expectedTask)
		}).
		Return(nil)

	err = s.grpcClient.Execute(t.Context(), s.session, task.New(
		task.NewID("test", "task"),
		"test.exec",
		Args{
			Value: 3,
		},
	), &result)

	require.NoError(t, err)
	assert.Len(t, result.GetFollowupTasks(), 1)

	unboxed, err := argconv.UnboxArgs[Args](result.GetFollowupTasks()[0])

	require.NoError(t, err)
	assert.EqualExportedValues(t, expectedTask.Args, *unboxed)
}

func TestRPC(t *testing.T) { //nolint:tparallel
	t.Parallel()

	suiteT := reflect.TypeFor[rpcSuite]()

	for method := range suiteT.Methods() { //nolint:paralleltest
		if !strings.HasPrefix(method.Name, "Test") {
			continue
		}

		t.Run(method.Name, func(t *testing.T) {
			suite := rpcSuite{}

			suite.SetupTest(t)

			method.Func.Call([]reflect.Value{
				reflect.ValueOf(&suite),
			})

			suite.AfterTest(t)
		})
	}
}
