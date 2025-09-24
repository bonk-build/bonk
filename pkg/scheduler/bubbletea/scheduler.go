// Copyright © 2025 Colden Cullen
// SPDX-License-Identifier: MIT

package bubbletea

import (
	"context"
	"log/slog"

	tea "github.com/charmbracelet/bubbletea/v2"
	slogmulti "github.com/samber/slog-multi"
	slogctx "github.com/veqryn/slog-context"

	"go.bonk.build/pkg/scheduler"
	"go.bonk.build/pkg/task"
)

type sched struct {
	program *tea.Program
	exec    task.Executor
}

// New creates a new scheduler driven by bubbletea.
func New(
	debugDump bool,
) scheduler.SchedulerFactory {
	return func(ctx context.Context, exec task.Executor) scheduler.Scheduler {
		result := &sched{
			program: tea.NewProgram(
				&teaModel{
					debugDump: debugDump,
				},
				tea.WithContext(ctx),
			),
			exec: exec,
		}

		// The logger needs to be replaced to avoid writing to stdout.
		defaultLogger := slog.Default()
		slog.SetDefault(
			slog.New(slogmulti.
				Pipe(slogctx.NewMiddleware(nil)).
				Handler(slogmulti.NewHandleInlineHandler(
					func(ctx context.Context, groups []string, attrs []slog.Attr, record slog.Record) error {
						go result.program.Printf("%s", record.Message)

						return nil
					},
				),
				),
			),
		)

		// Start the program
		go func() {
			_, err := result.program.Run()
			slog.SetDefault(defaultLogger)

			if err != nil {
				slog.ErrorContext(ctx, "error running bubbletea program", "error", err)
			}
		}()

		return result
	}
}

// AddTask implements scheduler.Scheduler.
func (s *sched) AddTask(ctx context.Context, tsk *task.Task) error {
	// This can block, so run in a goroutine
	go s.program.Send(TaskScheduleMsg{
		ctx:  ctx,
		tsk:  tsk,
		exec: s.exec,
	})

	return nil
}

// Run implements scheduler.Scheduler.
func (s *sched) Run() {
	s.program.Wait()
}
