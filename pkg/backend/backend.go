// Copyright © 2025 Colden Cullen
// SPDX-License-Identifier: MIT

package backend // import "go.bonk.build/pkg/backend"

import (
	"context"

	"go.bonk.build/pkg/task"
)

type Backend interface {
	Execute(ctx context.Context, tsk task.Task) error
}
