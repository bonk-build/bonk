// Copyright © 2025 Colden Cullen
// SPDX-License-Identifier: MIT

package main

import (
	"testing"

	"github.com/stretchr/testify/require"

	"go.bonk.build/pkg/task"
)

func Test_Plugin(t *testing.T) {
	t.Parallel()

	executors := Plugin.ServeTest(t)
	session := task.NewTestSession()

	require.NoError(t, executors.OpenSession(t.Context(), session))

	var result task.Result
	require.NoError(t, executors.Execute(
		t.Context(),
		task.New("testing", session, "test.Test", Params{
			Value: 2,
		}),
		&result,
	))

	executors.CloseSession(t.Context(), session.ID())
}
