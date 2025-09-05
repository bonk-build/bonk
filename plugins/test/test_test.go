// Copyright © 2025 Colden Cullen
// SPDX-License-Identifier: MIT

package main

import (
	"testing"

	"cuelang.org/go/cue"

	"github.com/google/uuid"

	"go.bonk.build/pkg/task"
	"go.bonk.build/test"
)

func Test_Plugin(t *testing.T) {
	t.Parallel()

	executors := test.ServeTest(t, Plugin)

	var result task.Result
	err := executors.Execute(
		t.Context(),
		task.New(uuid.Nil, "test.Test", "testing", cue.Value{}),
		&result,
	)
	if err != nil {
		t.Fatal("failed call to PerformTask:", err)
	}
}
