// Copyright © 2025 Colden Cullen
// SPDX-License-Identifier: MIT

package bubbletea

import (
	"context"
	"log/slog"
	"sync"

	tea "charm.land/bubbletea/v2"
	slogmulti "github.com/samber/slog-multi"
	slogctx "github.com/veqryn/slog-context"

	"go.bonk.build/pkg/executor/observable"
)

type observer struct {
	program *tea.Program
	waiter  sync.WaitGroup
}

// New creates a new scheduler driven by bubbletea.
func New(ctx context.Context, debugDump bool) *observer {
	result := &observer{
		program: tea.NewProgram(
			&teaModel{
				debugDump: debugDump,
			},
			tea.WithContext(ctx),
		),
	}

	// The logger needs to be replaced to avoid writing to stdout.
	defaultLogger := slog.Default()
	slog.SetDefault(
		slog.New(slogmulti.
			Pipe(slogctx.NewMiddleware(nil)).
			Handler(slogmulti.NewHandleInlineHandler(
				func(ctx context.Context, groups []string, attrs []slog.Attr, record slog.Record) error {
					result.waiter.Go(func() {
						result.program.Printf("%s", record.Message)
					})

					return nil
				},
			),
			),
		),
	)

	// Start the program
	result.waiter.Go(func() {
		_, err := result.program.Run()
		slog.SetDefault(defaultLogger)

		if err != nil {
			slog.ErrorContext(ctx, "error running bubbletea program", "error", err)
		}
	})

	return result
}

func (o *observer) OnTaskStatusMsg(tsm observable.TaskStatusMsg) {
	o.program.Send(tsm)
}

func (o *observer) Quit() {
	o.program.Quit()
	o.waiter.Wait()
}
