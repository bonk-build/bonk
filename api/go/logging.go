// Copyright © 2025 Colden Cullen
// SPDX-License-Identifier: MIT

package bonk

import (
	"context"
	"fmt"
	"log/slog"
	"runtime"

	"google.golang.org/grpc"
	"google.golang.org/protobuf/types/known/structpb"
	"google.golang.org/protobuf/types/known/timestamppb"

	slogctx "github.com/veqryn/slog-context"

	bonkv0 "go.bonk.build/api/go/proto/bonk/v0"
)

type streamHandler struct {
	slog.HandlerOptions

	sender grpc.ServerStreamingServer[bonkv0.StreamLogsResponse]
}

func (stream *streamHandler) Enabled(_ context.Context, level slog.Level) bool {
	return int(stream.Level.Level()) <= int(level)
}

func (stream *streamHandler) Handle(ctx context.Context, record slog.Record) error {
	if stream.AddSource {
		fs := runtime.CallersFrames([]uintptr{record.PC})
		f, _ := fs.Next()
		record.AddAttrs(slog.Any(slog.SourceKey, &slog.Source{
			Function: f.Function,
			File:     f.File,
			Line:     f.Line,
		}))
	}

	level := int64(record.Level)
	res := bonkv0.StreamLogsResponse_builder{
		Time:    timestamppb.New(record.Time),
		Message: &record.Message,
		Level:   &level,
		Attrs:   make(map[string]*structpb.Value, record.NumAttrs()),
	}

	record.Attrs(func(attr slog.Attr) bool {
		protoValue, err := structpb.NewValue(attr.Value.Any())
		if err != nil {
			panic(err)
		} else {
			res.Attrs[attr.Key] = protoValue
		}

		return true
	})

	err := stream.sender.Send(res.Build())
	if err != nil {
		return fmt.Errorf("failed to send record across gRPC: %w", err)
	}

	return nil
}

func (stream *streamHandler) WithAttrs(attrs []slog.Attr) slog.Handler {
	return stream
}

func (stream *streamHandler) WithGroup(name string) slog.Handler {
	return stream
}

func (s *grpcServer) StreamLogs(
	req *bonkv0.StreamLogsRequest,
	res grpc.ServerStreamingServer[bonkv0.StreamLogsResponse],
) error {
	slogDefault := slog.Default()

	slog.SetDefault(slog.New(
		slogctx.NewHandler(
			&streamHandler{
				HandlerOptions: slog.HandlerOptions{
					Level:     slog.Level(req.GetLevel()),
					AddSource: req.GetAddSource(),
				},
				sender: res,
			},
			nil,
		),
	))

	// Sleep until the request is canceled
	<-res.Context().Done()

	slog.SetDefault(slogDefault)

	return nil
}
