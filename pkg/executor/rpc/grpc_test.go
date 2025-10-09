// Copyright © 2025 Colden Cullen
// SPDX-License-Identifier: MIT

package rpc_test

import (
	"context"
	"net"
	"reflect"
	"strings"
	"testing"
	"time"

	"go.uber.org/mock/gomock"
	"golang.org/x/sync/errgroup"

	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
	"google.golang.org/grpc/test/bufconn"

	"github.com/stretchr/testify/assert"
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
	mock         *gomock.Controller
	exec         *mockexec.MockExecutor
	grpcServer   *grpc.Server
	grpcClient   executor.Executor
	session      task.Session
	serverWaiter errgroup.Group
}

func (s *rpcSuite) SetupTest(t *testing.T) {
	t.Helper()

	s.mock = gomock.NewController(t)
	s.exec = mockexec.NewMockExecutor(s.mock)

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

	require.Eventually(t, func() bool {
		return s.mock.Satisfied()
	}, 100*time.Millisecond, 10*time.Millisecond)
}

func (s *rpcSuite) Test_Connection(t *testing.T) {
	t.Parallel()

	assert.NotNil(t, s.grpcClient)
}

func (s *rpcSuite) Test_Session(t *testing.T) {
	t.Parallel()

	s.exec.EXPECT().OpenSession(gomock.Any(), gomock.Any()).Times(1)
	s.exec.EXPECT().CloseSession(gomock.Any(), s.session.ID()).Times(1)

	err := s.grpcClient.OpenSession(t.Context(), s.session)
	require.NoError(t, err)
	defer s.grpcClient.CloseSession(t.Context(), s.session.ID())
}

func (s *rpcSuite) Test_Session_Fail(t *testing.T) {
	t.Parallel()

	s.exec.EXPECT().OpenSession(gomock.Any(), gomock.Any()).Return(assert.AnError).Times(1)
	s.exec.EXPECT().CloseSession(gomock.Any(), s.session.ID()).Times(0)

	err := s.grpcClient.OpenSession(t.Context(), s.session)
	require.ErrorContains(t, err, assert.AnError.Error())
}

func (s *rpcSuite) Test_Args(t *testing.T) {
	t.Parallel()

	var result task.Result

	s.exec.EXPECT().OpenSession(gomock.Any(), gomock.Any()).Times(1)
	s.exec.EXPECT().CloseSession(gomock.Any(), s.session.ID()).Times(1)

	err := s.grpcClient.OpenSession(t.Context(), s.session)
	require.NoError(t, err)
	defer s.grpcClient.CloseSession(t.Context(), s.session.ID())

	s.exec.EXPECT().Execute(gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any()).
		Times(1).
		Return(nil)

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

	s.exec.EXPECT().OpenSession(gomock.Any(), gomock.Any()).Times(1)
	s.exec.EXPECT().CloseSession(gomock.Any(), s.session.ID()).Times(1)

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

	s.exec.EXPECT().Execute(gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any()).
		Times(1).
		Do(func(_ context.Context, _ *task.Task, res *task.Result) {
			res.FollowupTasks = append(res.FollowupTasks, *expectedTask)
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
	assert.Len(t, result.FollowupTasks, 1)

	unboxed, err := argconv.UnboxArgs[Args](&result.FollowupTasks[0])

	require.NoError(t, err)
	assert.EqualExportedValues(t, expectedTask.Args, *unboxed)
}

func TestRPC(t *testing.T) { //nolint:tparallel
	t.Parallel()

	suiteT := reflect.TypeFor[rpcSuite]()

	for idx := range suiteT.NumMethod() { //nolint:paralleltest
		method := suiteT.Method(idx)
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
