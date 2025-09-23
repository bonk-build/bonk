// Copyright © 2025 Colden Cullen
// SPDX-License-Identifier: MIT

// Package driver provides an interface for defining a driver to perform a bonk pipeline's work.
package driver

import (
	"context"

	"go.bonk.build/pkg/executor/plugin"
	"go.bonk.build/pkg/scheduler"
	"go.bonk.build/pkg/task"
)

//go:generate go tool mockgen -destination driver_mock.go -package driver -copyright_file ../../license-header.txt -typed -write_package_comment=false . Driver

// Driver is an interface for an object that can delegate the work of a bonk pipeline.
type Driver interface {
	plugin.PluginClientManager
	scheduler.Scheduler

	NewLocalSession(
		ctx context.Context,
		path string,
	) (task.LocalSession, error)

	Shutdown(ctx context.Context)
}
