// Copyright © 2025 Colden Cullen
// SPDX-License-Identifier: MIT

package plugin

import (
	"context"
	"errors"
	"fmt"
	"io"
	"log/slog"
	"reflect"

	"google.golang.org/grpc"

	goplugin "github.com/hashicorp/go-plugin"

	bonkv0 "go.bonk.build/api/proto/bonk/v0"
)

type logStreamingPluginClient struct {
	goplugin.NetRPCUnsupportedPlugin
}

var _ goplugin.GRPCPlugin = (*logStreamingPluginClient)(nil)

func (p *logStreamingPluginClient) GRPCServer(*goplugin.GRPCBroker, *grpc.Server) error {
	return errors.ErrUnsupported
}

func (p *logStreamingPluginClient) GRPCClient(
	_ context.Context,
	_ *goplugin.GRPCBroker,
	c *grpc.ClientConn,
) (any, error) {
	return bonkv0.NewLogStreamingServiceClient(c), nil
}

func (plugin *PluginClient) handleFeatureLogStreaming(
	ctx context.Context,
	pluginName string,
	client goplugin.ClientProtocol,
) error {
	logStreamingPlugin, err := client.Dispense("log_streaming")
	if err != nil {
		slog.DebugContext(ctx, "plugin does not support log streaming, skipping", "plugin", pluginName)

		return nil //nolint: nilerr
	}

	logStreamingClient, ok := logStreamingPlugin.(bonkv0.LogStreamingServiceClient)
	if !ok {
		panic(
			fmt.Sprintf(
				"plugin %s reports supporting log streaming but client returned was of the wrong type: %s",
				pluginName,
				reflect.TypeOf(logStreamingPlugin),
			),
		)
	}

	defaultLevel := int64(slog.LevelInfo)
	addSource := false

	req := bonkv0.StreamLogsRequest_builder{
		Level:     &defaultLevel,
		AddSource: &addSource,
	}

	logStream, err := logStreamingClient.StreamLogs(ctx, req.Build())
	if err != nil {
		slog.ErrorContext(ctx, "failed to call client.StreamLogs", "error", err)

		return fmt.Errorf("failed to call client.StreamLogs: %w", err)
	}

	go func() { //nolint: contextcheck
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

			slogHandler := slog.Default().With("plugin", pluginName).Handler()
			if slogHandler.Enabled(recvCtx, record.Level) {
				err = slogHandler.Handle(recvCtx, record)
				if err != nil {
					slog.ErrorContext(recvCtx, "failed to handle message")
				}
			}
		}
	}()

	return nil
}
