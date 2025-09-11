// Copyright © 2025 Colden Cullen
// SPDX-License-Identifier: MIT

package plugin // import "go.bonk.build/pkg/executor/plugin"

import (
	"context"
	"errors"
	"io"
	"log/slog"

	"google.golang.org/grpc"

	bonkv0 "go.bonk.build/api/proto/bonk/v0"
)

// When GRPCClient is called, it means that Dispense() was invoked and we can begin processing logs.
//
//nolint:contextcheck
func handleLogStreaming(
	ctx context.Context,
	c *grpc.ClientConn,
	name string,
) {
	logStreamingClient := bonkv0.NewLogStreamingServiceClient(c)

	defaultLevel := int64(slog.LevelInfo)
	addSource := false

	req := bonkv0.StreamLogsRequest_builder{
		Level:     &defaultLevel,
		AddSource: &addSource,
	}

	logStream, err := logStreamingClient.StreamLogs(ctx, req.Build())
	if err != nil {
		slog.ErrorContext(ctx, "failed to call client.StreamLogs", "error", err)

		return
	}

	recvCtx := logStream.Context()
	for {
		msg, err := logStream.Recv()
		if err != nil {
			if recvCtx.Err() != nil || errors.Is(err, io.EOF) {
				// If this occurs, the log stream is imply shutting down and we should exit
				break
			} else {
				slog.ErrorContext(recvCtx, "received error on log stream", "error", err, "context err", recvCtx.Err())

				continue
			}
		}

		record := slog.NewRecord(
			msg.GetTime().AsTime(),
			slog.Level(msg.GetLevel()),
			msg.GetMessage(),
			0,
		)

		for key, value := range msg.GetAttrs() {
			record.AddAttrs(slog.Attr{
				Key:   key,
				Value: slog.AnyValue(value.AsInterface()),
			})
		}

		slogHandler := slog.Default().With("plugin", name).Handler()
		if slogHandler.Enabled(recvCtx, record.Level) {
			err = slogHandler.Handle(recvCtx, record)
			if err != nil {
				slog.ErrorContext(recvCtx, "failed to handle message")
			}
		}
	}
}
