// Copyright © 2025 Colden Cullen
// SPDX-License-Identifier: MIT

package rpc_test

import (
	"context"
	"errors"
	"net"
	"testing"
	"time"

	"go.uber.org/mock/gomock"

	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
	"google.golang.org/grpc/test/bufconn"

	"github.com/stretchr/testify/suite"

	"go.bonk.build/pkg/executor/rpc"
	"go.bonk.build/pkg/task"
)

type Args struct {
	Value int
}

type rpcSuite struct {
	suite.Suite

	mock       *gomock.Controller
	exec       *task.MockExecutor[Args]
	grpcClient task.GenericExecutor
	session    task.Session
}

func (s *rpcSuite) SetupTest() {
	s.mock = gomock.NewController(s.T())
	s.exec = task.NewMockExecutor[Args](s.mock)

	lis := bufconn.Listen(1024 * 1024)
	server := grpc.NewServer()
	rpc.RegisterGRPCServer(server, task.BoxExecutor(s.exec))

	go func() {
		err := server.Serve(lis)
		s.NoError(err, "Server execited with err: %v", err)
	}()

	bufDialer := func(context.Context, string) (net.Conn, error) {
		return lis.Dial()
	}

	clientConn, err := grpc.NewClient("passthrough:///bufnet",
		grpc.WithContextDialer(bufDialer),
		grpc.WithTransportCredentials(insecure.NewCredentials()),
	)
	s.Require().NoError(err)

	s.grpcClient = rpc.NewGRPCClient(clientConn)

	s.session = task.NewTestSession()
}

func (s *rpcSuite) AfterTest(_, _ string) {
	s.Require().Eventually(func() bool {
		return s.mock.Satisfied()
	}, 100*time.Millisecond, 10*time.Millisecond)
}

// func (s *rpcSuite) Test_RPC(t *testing.T) {
// 	t.Parallel()

// 	tests := []testFunc{
// 		test_Connection,
// 		test_Session,
// 		test_Session_Fail,
// 		test_Args,
// 		test_Followups,
// 	}

// 	for _, testFunc := range tests {
// 		name := runtime.FuncForPC(reflect.ValueOf(testFunc).Pointer()).Name()
// 		t.Run(name, func(t *testing.T) {
// 			t.Parallel()

// 			testFunc(t, mock, exec)

// 			require.Eventually(t, func() bool {
// 				return mock.Satisfied()
// 			}, 100*time.Millisecond, 10*time.Millisecond)
// 		})
// 	}
// }

func (s *rpcSuite) Test_Connection() {
	s.NotNil(s.grpcClient)
}

func (s *rpcSuite) Test_Session() {
	s.exec.EXPECT().CloseSession(gomock.Any(), s.session.ID()).Times(1)
	s.exec.EXPECT().OpenSession(gomock.Any(), gomock.Any()).Times(1)

	err := s.grpcClient.OpenSession(s.T().Context(), s.session)
	s.Require().NoError(err)
	defer s.grpcClient.CloseSession(s.T().Context(), s.session.ID())
}

func (s *rpcSuite) Test_Session_Fail() {
	s.exec.EXPECT().OpenSession(gomock.Any(), gomock.Any()).Return(errors.ErrUnsupported).Times(1)
	s.exec.EXPECT().CloseSession(gomock.Any(), s.session.ID()).Times(0)

	err := s.grpcClient.OpenSession(s.T().Context(), s.session)
	s.Require().ErrorContains(err, errors.ErrUnsupported.Error())
	defer s.grpcClient.CloseSession(s.T().Context(), s.session.ID())
}

func (s *rpcSuite) Test_Args() {
	var result task.Result

	s.exec.EXPECT().OpenSession(gomock.Any(), gomock.Any()).Times(1)
	s.exec.EXPECT().CloseSession(gomock.Any(), s.session.ID()).Times(1)

	err := s.grpcClient.OpenSession(s.T().Context(), s.session)
	s.Require().NoError(err)
	defer s.grpcClient.CloseSession(s.T().Context(), s.session.ID())

	s.exec.EXPECT().Execute(gomock.Any(), gomock.Any(), gomock.Any()).
		Times(1).
		Return(nil)

	err = s.grpcClient.Execute(s.T().Context(), task.New(
		"test.task",
		s.session,
		"test.exec",
		Args{
			Value: 3,
		},
	).Box(), &result)
	s.Require().NoError(err)
}

func (s *rpcSuite) Test_Followups() {
	var result task.Result

	s.exec.EXPECT().OpenSession(gomock.Any(), gomock.Any()).Times(1)
	s.exec.EXPECT().CloseSession(gomock.Any(), s.session.ID()).Times(1)

	err := s.grpcClient.OpenSession(s.T().Context(), s.session)
	s.Require().NoError(err)
	defer s.grpcClient.CloseSession(s.T().Context(), s.session.ID())

	expectedTask := task.Task[Args]{
		ID:       task.TaskID("Test.Task"),
		Executor: "Test.Executor",
		Session:  s.session,
		Inputs: []string{
			"File1.txt",
			"File2.txt",
		},
		Args: Args{
			Value: 69420,
		},
	}

	s.exec.EXPECT().Execute(gomock.Any(), gomock.Any(), gomock.Any()).
		Times(1).
		Do(func(ctx context.Context, tsk *task.Task[Args], res *task.Result) {
			res.FollowupTasks = append(res.FollowupTasks, *expectedTask.Box())
		}).
		Return(nil)

	err = s.grpcClient.Execute(s.T().Context(), task.New[any](
		"test.task",
		s.session,
		"test.exec",
		Args{
			Value: 3,
		},
	), &result)

	s.Require().NoError(err)
	s.Len(result.FollowupTasks, 1)

	unboxed, err := task.Unbox[Args](&result.FollowupTasks[0])

	s.Require().NoError(err)
	s.EqualExportedValues(expectedTask, *unboxed)
}

func TestRPC(t *testing.T) {
	t.Parallel()
	suite.Run(t, &rpcSuite{})
}
