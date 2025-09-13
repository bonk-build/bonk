// Copyright © 2025 Colden Cullen
// SPDX-License-Identifier: MIT

package logging

import (
	"context"
	"log/slog"

	slogmulti "github.com/samber/slog-multi"
)

func NewHandler() slog.Handler {
	return slogmulti.NewHandleInlineHandler(
		func(ctx context.Context, groups []string, attrs []slog.Attr, record slog.Record) error {
			return nil
		},
	)
}
