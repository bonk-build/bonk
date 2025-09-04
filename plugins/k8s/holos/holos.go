// Copyright © 2025 Colden Cullen
// SPDX-License-Identifier: MIT

package main

import (
	"fmt"

	bonk "go.bonk.build/api/go"
)

var Plugin = bonk.NewPlugin("holos", func(plugin *bonk.Plugin) error {
	err := plugin.RegisterExecutors(
		bonk.WrapTypedExecutor(plugin.Cuectx, Executor_RenderPlatform{}),
		bonk.WrapTypedExecutor(plugin.Cuectx, Executor_RenderComponent{}),
	)
	if err != nil {
		return fmt.Errorf("failed to register holos executors: %w", err)
	}

	return nil
})

func main() {
	Plugin.Serve()
}
