// Copyright © 2025 Colden Cullen
// SPDX-License-Identifier: MIT

package bonk

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"os"
	"runtime"
	"time"

	"go.uber.org/multierr"

	"google.golang.org/grpc"
	"google.golang.org/protobuf/types/known/structpb"
	"google.golang.org/protobuf/types/known/timestamppb"

	"github.com/delicb/slogbuffer"

	goplugin "github.com/hashicorp/go-plugin"
	slogmulti "github.com/samber/slog-multi"
	slogctx "github.com/veqryn/slog-context"

	bonkv0 "go.bonk.build/api/proto/bonk/v0"
)

var (
	bufferedHandler             *slogbuffer.BufferLogHandler
	cancelWaitForStreamingSetup chan struct{}
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

func init() {
	bufferedHandler = slogbuffer.NewBufferLogHandler(slog.LevelDebug)

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
		).Handler(bufferedHandler),
	))

	// Start a timer to just use stdout after some time if streaming isn't configured
	streamingSetupTimeout := time.NewTimer(1 * time.Second)
	cancelWaitForStreamingSetup = make(chan struct{})
	go func() {
		select {
		case <-streamingSetupTimeout.C:
			err := bufferedHandler.SetRealHandler(
				context.Background(),
				slog.NewTextHandler(os.Stdout, nil),
			)
			if err != nil {
				slog.Error("failed to flush buffered handler", "error", err)
			} else {
				slog.Debug("timed out waiting for streaming setup, switching to stdout")
			}

		case <-cancelWaitForStreamingSetup:
			streamingSetupTimeout.Stop()
		}
	}()
}

func getTaskLoggingContext(
	ctx context.Context,
	root *os.Root,
) (context.Context, func() error, error) {
	// Open log txt and json files
	logFileText, err := root.Create("log.txt")
	if err != nil {
		return nil, nil, errors.New("failed to open log txt file")
	}
	logFileJSON, err := root.Create("log.jsonl")
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

	// Add logger which writes to the default handler, but also local files
	return slogctx.NewCtx(ctx, slog.New(slogmulti.Fanout(
		slog.NewTextHandler(logFileText, &config),
		slog.NewJSONHandler(logFileJSON, &config),
		bufferedHandler,
	))), cleanup, nil
}

type LogStreamingServer struct {
	goplugin.NetRPCUnsupportedPlugin
	goplugin.GRPCPlugin
}

func (p *LogStreamingServer) GRPCServer(_ *goplugin.GRPCBroker, s *grpc.Server) error {
	bonkv0.RegisterLogStreamingServiceServer(s, &logStreamingGRPCServer{})

	return nil
}

func (p *LogStreamingServer) GRPCClient(
	_ context.Context,
	_ *goplugin.GRPCBroker,
	c *grpc.ClientConn,
) (any, error) {
	return bonkv0.NewLogStreamingServiceClient(c), nil
}

type logStreamingGRPCServer struct {
	bonkv0.UnimplementedLogStreamingServiceServer
}

func (s *logStreamingGRPCServer) StreamLogs(
	req *bonkv0.StreamLogsRequest,
	res grpc.ServerStreamingServer[bonkv0.StreamLogsResponse],
) error {
	// Cancel the timeout now that a request has been received
	cancelWaitForStreamingSetup <- struct{}{}

	ctx := res.Context()

	streamer := streamHandler{
		HandlerOptions: slog.HandlerOptions{
			Level:     slog.Level(req.GetLevel()),
			AddSource: req.GetAddSource(),
		},
		sender: res,
	}

	err := bufferedHandler.SetRealHandler(ctx, &streamer)
	if err != nil {
		slog.ErrorContext(ctx, "failed to flush buffered logs to the streamer", "error", err)
	}

	// Sleep until the request is canceled
	<-ctx.Done()

	// Once the stream is requested to be closed, forward logs to stdout.
	err = bufferedHandler.SetRealHandler(ctx, slog.NewTextHandler(os.Stdout, nil))
	if err != nil {
		slog.ErrorContext(ctx, "failed to flush buffered logs to the streamer", "error", err)
	}

	return nil
}
