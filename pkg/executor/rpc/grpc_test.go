// Copyright © 2025 Colden Cullen
// SPDX-License-Identifier: MIT

//nolint:thelper // It doesn't recognize the subtests
package rpc_test

import (
	"context"
	"errors"
	"net"
	"reflect"
	"runtime"
	"testing"
	"time"

	"go.uber.org/mock/gomock"

	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
	"google.golang.org/grpc/test/bufconn"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"go.bonk.build/pkg/executor/rpc"
	"go.bonk.build/pkg/task"
)

type Args struct {
	Value int
}

func openConnection(t *testing.T, exec task.GenericExecutor) task.GenericExecutor {
	t.Helper()

	lis := bufconn.Listen(1024 * 1024)
	s := grpc.NewServer()
	rpc.RegisterGRPCServer(s, exec)

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

	return rpc.NewGRPCClient(clientConn)
}

type testFunc = func(t *testing.T, mock *gomock.Controller, exec *task.MockExecutor[Args])

func Test_RPC(t *testing.T) {
	t.Parallel()

	tests := []testFunc{
		test_Connection,
		test_Session,
		test_Session_Fail,
		test_Args,
		test_Followups,
	}

	for _, testFunc := range tests {
		name := runtime.FuncForPC(reflect.ValueOf(testFunc).Pointer()).Name()
		t.Run(name, func(t *testing.T) {
			t.Parallel()

			mock := gomock.NewController(t)
			exec := task.NewMockExecutor[Args](mock)

			testFunc(t, mock, exec)

			require.Eventually(t, func() bool {
				return mock.Satisfied()
			}, 100*time.Millisecond, 10*time.Millisecond)
		})
	}
}

func test_Connection(t *testing.T, mock *gomock.Controller, exec *task.MockExecutor[Args]) {
	client := openConnection(t, task.BoxExecutor(exec))
	require.NotNil(t, client)
}

func test_Session(t *testing.T, mock *gomock.Controller, exec *task.MockExecutor[Args]) {
	session := task.NewTestSession()

	client := openConnection(t, task.BoxExecutor(exec))
	require.NotNil(t, client)

	exec.EXPECT().CloseSession(gomock.Any(), session.ID()).Times(1)
	exec.EXPECT().OpenSession(gomock.Any(), gomock.Any()).Times(1)

	err := client.OpenSession(t.Context(), session)
	require.NoError(t, err)
	defer client.CloseSession(t.Context(), session.ID())
}

func test_Session_Fail(t *testing.T, mock *gomock.Controller, exec *task.MockExecutor[Args]) {
	session := task.NewTestSession()

	client := openConnection(t, task.BoxExecutor(exec))
	require.NotNil(t, client)

	exec.EXPECT().OpenSession(gomock.Any(), gomock.Any()).Return(errors.ErrUnsupported).Times(1)
	exec.EXPECT().CloseSession(gomock.Any(), session.ID()).Times(0)

	err := client.OpenSession(t.Context(), session)
	require.ErrorContains(t, err, errors.ErrUnsupported.Error())
	defer client.CloseSession(t.Context(), session.ID())

	require.Eventually(t, func() bool {
		return mock.Satisfied()
	}, 100*time.Millisecond, 10*time.Millisecond)
}

func test_Args(t *testing.T, mock *gomock.Controller, exec *task.MockExecutor[Args]) {
	session := task.NewTestSession()

	client := openConnection(t, task.BoxExecutor(exec))
	require.NotNil(t, client)

	var result task.Result

	exec.EXPECT().OpenSession(gomock.Any(), gomock.Any()).Times(1)
	exec.EXPECT().CloseSession(gomock.Any(), session.ID()).Times(1)

	err := client.OpenSession(t.Context(), session)
	require.NoError(t, err)
	defer client.CloseSession(t.Context(), session.ID())

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

func test_Followups(t *testing.T, mock *gomock.Controller, exec *task.MockExecutor[Args]) {
	session := task.NewTestSession()

	client := openConnection(t, task.BoxExecutor(exec))
	require.NotNil(t, client)

	var result task.Result

	exec.EXPECT().OpenSession(gomock.Any(), gomock.Any()).Times(1)
	exec.EXPECT().CloseSession(gomock.Any(), session.ID()).Times(1)

	err := client.OpenSession(t.Context(), session)
	require.NoError(t, err)
	defer client.CloseSession(t.Context(), session.ID())

	expectedTask := task.Task[Args]{
		ID: task.TaskId{
			Name:     "Test.Task",
			Executor: "Test.Executor",
		},
		Session: session,
		Inputs: []string{
			"File1.txt",
			"File2.txt",
		},
		Args: Args{
			Value: 69420,
		},
	}

	exec.EXPECT().Execute(gomock.Any(), gomock.Any(), gomock.Any()).
		Times(1).
		Do(func(ctx context.Context, tsk *task.Task[Args], res *task.Result) {
			res.FollowupTasks = append(res.FollowupTasks, *expectedTask.Box())
		}).
		Return(nil)

	err = client.Execute(t.Context(), task.New(
		session,
		"test.exec",
		"test.task",
		Args{
			Value: 3,
		},
	).Box(), &result)
	require.NoError(t, err)

	require.Len(t, result.FollowupTasks, 1)

	unboxed, err := task.Unbox[Args](&result.FollowupTasks[0])

	// Update the name since it gets modified
	expectedTask.ID.Name = "test.task." + expectedTask.ID.Name

	require.NoError(t, err)
	require.EqualExportedValues(t, expectedTask, *unboxed)
}
