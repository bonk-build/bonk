// Copyright © 2025 Colden Cullen
// SPDX-License-Identifier: MIT

package main

import (
	"testing"

	"github.com/stretchr/testify/require"

	"go.bonk.build/pkg/task"
	"go.bonk.build/test"
)

func Test_Plugin(t *testing.T) {
	t.Parallel()

	executors := test.ServeTest(t, Plugin)
	session := test.NewTestSession()

	require.NoError(t, executors.OpenSession(t.Context(), session))

	var result task.Result
	require.NoError(t, executors.Execute(
		t.Context(),
		task.New[any](session, "test.Test", "testing", Params{
			Value: 2,
		}),
		&result,
	))

	executors.CloseSession(t.Context(), session.ID())
}
