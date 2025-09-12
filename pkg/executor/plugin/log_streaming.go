// Copyright © 2025 Colden Cullen
// SPDX-License-Identifier: MIT

package plugin

import (
	"context"
	"errors"
	"log/slog"

	"go.uber.org/multierr"

	slogmulti "github.com/samber/slog-multi"
	slogctx "github.com/veqryn/slog-context"

	"go.bonk.build/pkg/task"
)

// This installs a default log handler into plugins that import this package.
func init() {
	// Install the default log handler
	slog.SetDefault(slog.New(
		slogmulti.Pipe(
			// Append added context information
			slogctx.NewMiddleware(nil),
			// Check if the context has a logger in it, and use that if so
			slogmulti.NewHandleInlineMiddleware(
				func(ctx context.Context, record slog.Record, next func(context.Context, slog.Record) error) error {
					if ctxLogger := slogctx.FromCtx(ctx); ctxLogger != slog.Default() {
						if ctxLogger.Enabled(ctx, record.Level) {
							return ctxLogger.Handler().Handle(ctx, record)
						} else {
							return nil
						}
					} else {
						return next(ctx, record)
					}
				},
			),
			// Route to the buffered handler
		).Handler(slog.Default().Handler()),
	))
}

func getTaskLoggingContext(
	ctx context.Context,
	tsk *task.GenericTask,
) (context.Context, func() error, error) {
	// Open log txt and json files
	logFileText, err := tsk.OutputFS().Create("log.txt")
	if err != nil {
		return nil, nil, errors.New("failed to open log txt file")
	}
	logFileJSON, err := tsk.OutputFS().Create("log.jsonl")
	if err != nil {
		return nil, nil, errors.New("failed to open log json file")
	}

	config := slog.HandlerOptions{
		Level: slog.LevelDebug,
	}

	cleanup := func() error {
		return multierr.Combine(
			logFileText.Close(),
			logFileJSON.Close(),
		)
	}

	ctx = slogctx.Append(ctx,
		"executor", tsk.ID.Executor,
	)

	// Add logger which writes to the default handler, but also local files
	return slogctx.Append(slogctx.NewCtx(ctx,
		slog.New(slogmulti.Fanout(
			slog.NewTextHandler(logFileText, &config),
			slog.NewJSONHandler(logFileJSON, &config),
			// If ctx already contains a logger, we can chain to it.
			// If not, we just hit default.
			slogctx.FromCtx(ctx).Handler(),
		)))), cleanup, nil
}
