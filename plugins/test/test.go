// Copyright © 2025 Colden Cullen
// SPDX-License-Identifier: MIT

package main // import "go.bonk.build/plugins/test"

import (
	"context"
	"log/slog"

	plugin "go.bonk.build/api/go"
)

type Params struct {
	Value int `json:"value"`
}

var Executor_Test = plugin.NewExecutor(
	"Test",
	func(ctx context.Context, param *plugin.TaskParams[Params]) error {
		slog.InfoContext(ctx, "it's happening!", "thing", "value")

		return nil
	},
)

func main() {
	plugin.Serve(
		Executor_Test,
	)
}
