// Copyright © 2025 Colden Cullen
// SPDX-License-Identifier: MIT

package main // import "go.bonk.build/plugins/test"

import (
	"context"
	"fmt"
	"log/slog"

	"go.bonk.build/pkg/executor/plugin"
	"go.bonk.build/pkg/task"
)

type Params struct {
	Value int `json:"value"`
}

type Executor_Test struct {
	task.NoopSessionManager
}

func (Executor_Test) Execute(
	ctx context.Context,
	task *task.Task[Params],
	res *task.Result,
) error {
	slog.InfoContext(ctx, "it's happening!", "thing", task.Args.Value)

	return nil
}

var Plugin = plugin.NewPlugin("test", func(plugin *plugin.Plugin) error {
	err := plugin.RegisterExecutor("Test", task.BoxExecutor(Executor_Test{}))
	if err != nil {
		return fmt.Errorf("failed to register Test executor: %w", err)
	}

	return nil
})

func main() {
	Plugin.Serve()
}
