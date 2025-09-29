// Copyright © 2025 Colden Cullen
// SPDX-License-Identifier: MIT

// test provides a plugin purely for testing purposes.
package main

import (
	"context"
	"log/slog"

	"go.bonk.build/pkg/executor"
	"go.bonk.build/pkg/executor/plugin"
	"go.bonk.build/pkg/task"
)

type Params struct {
	Value int `json:"value"`
}

type ExecutorTest struct {
	executor.NoopSessionManager
}

func (ExecutorTest) Execute(
	ctx context.Context,
	_ *task.Task,
	args *Params,
	_ *task.Result,
) error {
	slog.InfoContext(ctx, "it's happening!", "thing", args.Value)

	return nil
}

var Plugin = plugin.NewPlugin("test",
	plugin.WithExecutor("Test", ExecutorTest{}),
)

func main() {
	Plugin.Serve()
}
