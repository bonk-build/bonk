// Copyright © 2025 Colden Cullen
// SPDX-License-Identifier: MIT

package plugin

import (
	"context"
	"errors"
	"fmt"
	"io"
	"log/slog"
	"time"

	"google.golang.org/grpc"

	goplugin "github.com/hashicorp/go-plugin"

	bonkv0 "go.bonk.build/api/go/proto/bonk/v0"
	"go.bonk.build/pkg/executor"
)

type Plugin struct {
	name      string
	client    bonkv0.BonkPluginServiceClient
	executors map[string]executor.Executor
}

func NewPlugin(
	ctx context.Context,
	name string,
	client bonkv0.BonkPluginServiceClient,
) (*Plugin, error) {
	configureCtx, cancel := context.WithTimeout(ctx, 1*time.Second)
	resp, err := client.ConfigurePlugin(configureCtx, &bonkv0.ConfigurePluginRequest{})
	cancel()
	if err != nil {
		return nil, fmt.Errorf("failed to describe plugin: %w", err)
	}

	plugin := &Plugin{
		name:      name,
		client:    client,
		executors: make(map[string]executor.Executor, len(resp.GetExecutors())),
	}

	for _, feature := range resp.GetFeatures() {
		switch feature {
		default:
			// unsupported feature, ignore

		case bonkv0.ConfigurePluginResponse_FEATURE_FLAGS_STREAMING_LOGGING:
			err = plugin.handleFeatureLogging(ctx)
			if err != nil {
				slog.WarnContext(ctx, "failed to configure streaming logging for plugin", "error", err)
			}
		}
	}

	for name := range resp.GetExecutors() {
		_, existed := plugin.executors[name]
		if existed {
			slog.WarnContext(ctx, "duplicate executor detected", "name", name)
		}

		plugin.executors[name] = executor.NewRPC(name, client)
	}

	return plugin, nil
}

func (p *Plugin) handleFeatureLogging(ctx context.Context) error {
	defaultLevel := int64(slog.LevelInfo)
	addSource := false

	req := bonkv0.StreamLogsRequest_builder{
		Level:     &defaultLevel,
		AddSource: &addSource,
	}

	logStream, err := p.client.StreamLogs(ctx, req.Build())
	if err != nil {
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

			slogHandler := slog.Default().With("plugin", p.name).Handler()
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

// Plugin Client

type bonkPluginClient struct {
	goplugin.NetRPCUnsupportedPlugin
}

func (p *bonkPluginClient) GRPCServer(*goplugin.GRPCBroker, *grpc.Server) error {
	return errors.ErrUnsupported
}

func (p *bonkPluginClient) GRPCClient(
	_ context.Context,
	_ *goplugin.GRPCBroker,
	c *grpc.ClientConn,
) (any, error) {
	return bonkv0.NewBonkPluginServiceClient(c), nil
}
