// Copyright © 2025 Colden Cullen
// SPDX-License-Identifier: MIT

package rpc

import (
	"errors"
	"io"
	"log/slog"

	"google.golang.org/grpc"

	bonkv0 "go.bonk.build/api/bonk/v0"
)

func handleLogStreaming(
	stream grpc.ServerStreamingClient[bonkv0.OpenSessionResponse],
) {
	for {
		msg, err := stream.Recv()
		if err != nil {
			if stream.Context().Err() != nil || errors.Is(err, io.EOF) {
				// If this occurs, the log stream is imply shutting down and we should exit
				break
			} else {
				slog.ErrorContext(
					stream.Context(),
					"received error on log stream",
					"error", err,
					"context err", stream.Context().Err())

				continue
			}
		}

		switch msg.WhichMessage() {
		case bonkv0.OpenSessionResponse_LogRecord_case:
			attrs := make([]slog.Attr, 0, len(msg.GetLogRecord().GetAttrs()))
			for key, value := range msg.GetLogRecord().GetAttrs() {
				attrs = append(attrs, slog.Attr{
					Key:   key,
					Value: slog.AnyValue(value.AsInterface()),
				})
			}

			slog.LogAttrs(
				stream.Context(),
				slog.Level(msg.GetLogRecord().GetLevel()),
				msg.GetLogRecord().GetMessage(),
				attrs...,
			)

		default:
			slog.ErrorContext(stream.Context(), "received unknown session response")

			continue
		}
	}
}
