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

var Backend_Test = plugin.NewBackend(
	"Test",
	[]string{},
	func(ctx context.Context, param *plugin.TaskParams[Params]) error {
		slog.InfoContext(ctx, "it's happening!", "thing", "value")

		return nil
	},
)

func main() {
	plugin.Serve(
		Backend_Test,
	)
}
