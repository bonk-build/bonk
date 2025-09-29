// Copyright © 2025 Colden Cullen
// SPDX-License-Identifier: MIT

// Package executor provides useful executors for building task-executing heirarchies.
// Each package exports just a few objects conforming to the [task.Executor] interface,
// and optionally a few helpers.
package executor

import (
	_ "go.bonk.build/pkg/task"
)
