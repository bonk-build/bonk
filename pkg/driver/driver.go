// Copyright © 2025 Colden Cullen
// SPDX-License-Identifier: MIT

package driver

import (
	"context"

	"go.bonk.build/pkg/executor/plugin"
	"go.bonk.build/pkg/scheduler"
	"go.bonk.build/pkg/task"
)

//go:generate go tool mockgen -destination driver_mock.go -package driver -copyright_file ../../license-header.txt -typed . Driver

type Driver interface {
	plugin.PluginClientManager
	scheduler.Scheduler

	NewLocalSession(
		ctx context.Context,
		path string,
	) (task.LocalSession, error)

	Shutdown(ctx context.Context)
}
