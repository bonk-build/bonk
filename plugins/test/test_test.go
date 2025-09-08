// Copyright © 2025 Colden Cullen
// SPDX-License-Identifier: MIT

package main

import (
	"testing"

	"go.bonk.build/pkg/task"
	"go.bonk.build/test"
)

func Test_Plugin(t *testing.T) {
	t.Parallel()

	executors := test.ServeTest(t, Plugin)
	session := test.NewTestSession()

	var result task.Result
	err := executors.Execute(
		t.Context(),
		*task.New[any](session, "test.Test", "testing", Params{
			Value: 2,
		}),
		&result,
	)
	if err != nil {
		t.Fatal("failed call to PerformTask:", err)
	}
}
