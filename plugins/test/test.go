// Copyright © 2025 Colden Cullen
// SPDX-License-Identifier: MIT

package main // import "go.bonk.build/plugins/test"

import (
	"context"
	"fmt"
	"log/slog"

	bonk "go.bonk.build/api/go"
)

type Params struct {
	Value int `json:"value"`
}

type Executor_Test struct{}

func (Executor_Test) Execute(
	ctx context.Context,
	task bonk.TypedTask[Params],
	res *bonk.Result,
) error {
	slog.InfoContext(ctx, "it's happening!", "thing", task.Args.Value)

	return nil
}

var Plugin = bonk.NewPlugin("test", func(plugin *bonk.Plugin) error {
	err := plugin.RegisterExecutor("Test", bonk.WrapTypedExecutor(plugin.Cuectx, Executor_Test{}))
	if err != nil {
		return fmt.Errorf("failed to register Test executor: %w", err)
	}

	return nil
})

func main() {
	Plugin.Serve()
}
