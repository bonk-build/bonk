// Copyright © 2025 Colden Cullen
// SPDX-License-Identifier: MIT

package plugin

import (
	goplugin "github.com/hashicorp/go-plugin"
)

var (
	_ goplugin.GRPCPlugin = (*executorPluginClient)(nil)
	_ goplugin.GRPCPlugin = (*logStreamingPluginClient)(nil)
)
