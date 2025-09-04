// Copyright © 2025 Colden Cullen
// SPDX-License-Identifier: MIT

package holos

import (
	core "github.com/holos-run/holos/api/core/v1alpha5"

	"go.bonk.build/pkg/executor/plugin"
)

var Plugin = plugin.NewPlugin("holos",
	plugin.WithExecutor[Params_RenderPlatform]("RenderPlatform", Executor_RenderPlatform{}),
	plugin.WithExecutor[core.Component]("RenderComponent", Executor_RenderComponent{}),
)
