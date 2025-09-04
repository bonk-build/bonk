// Copyright © 2025 Colden Cullen
// SPDX-License-Identifier: MIT

package holos

import (
	"fmt"

	"go.uber.org/multierr"

	"go.bonk.build/pkg/executor/plugin"
	"go.bonk.build/pkg/task"
)

var Plugin = plugin.NewPlugin("holos", func(plugin *plugin.Plugin) error {
	var err error
	multierr.AppendInto(
		&err,
		plugin.RegisterExecutor("RenderPlatform", task.BoxExecutor(Executor_RenderPlatform{})),
	)
	multierr.AppendInto(
		&err,
		plugin.RegisterExecutor("RenderComponent", task.BoxExecutor(Executor_RenderComponent{})),
	)
	if err != nil {
		return fmt.Errorf("failed to register holos executors: %w", err)
	}

	return nil
})
