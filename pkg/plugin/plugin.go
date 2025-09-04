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
	"time"

	"go.uber.org/multierr"

	goplugin "github.com/hashicorp/go-plugin"

	bonkv0 "go.bonk.build/api/proto/bonk/v0"
	"go.bonk.build/pkg/executor"
)

var Handshake = goplugin.HandshakeConfig{
	ProtocolVersion:  0,
	MagicCookieKey:   "BONK_PLUGIN",
	MagicCookieValue: "bonk the builder",
}

type Plugin struct {
	name           string
	executorClient bonkv0.ExecutorServiceClient
	executors      map[string]executor.Executor
}

func (plugin *Plugin) Configure(ctx context.Context, client goplugin.ClientProtocol) error {
	var err error

	multierr.AppendInto(&err, plugin.handleFeatureExecutor(ctx, client))
	multierr.AppendInto(&err, plugin.handleFeatureLogStreaming(ctx, client))

	return err
}

func (plugin *Plugin) handleFeatureExecutor(
	ctx context.Context,
	client goplugin.ClientProtocol,
) error {
	configureCtx, cancel := context.WithTimeout(ctx, 1*time.Second)
	defer cancel()

	executorPlugin, err := client.Dispense("executor")
	if err != nil {
		return fmt.Errorf("failed to dispense executor plugin: %w", err)
	}

	var ok bool
	plugin.executorClient, ok = executorPlugin.(bonkv0.ExecutorServiceClient)
	if !ok {
		panic(
			fmt.Sprintf(
				"plugin %s reports supporting executors but client returned was of the wrong type",
				plugin.name,
			),
		)
	}

	resp, err := plugin.executorClient.DescribeExecutors(
		configureCtx,
		&bonkv0.DescribeExecutorsRequest{},
	)
	if err != nil {
		return fmt.Errorf("failed to describe plugin: %w", err)
	}

	plugin.executors = make(map[string]executor.Executor)

	for name := range resp.GetExecutors() {
		_, existed := plugin.executors[name]
		if existed {
			slog.WarnContext(ctx, "duplicate executor detected", "name", name)
		}

		plugin.executors[name] = executor.NewRPC(name, plugin.executorClient)
	}

	return nil
}

func (plugin *Plugin) handleFeatureLogStreaming(
	ctx context.Context,
	client goplugin.ClientProtocol,
) error {
	logStreamingPlugin, err := client.Dispense("log_streaming")
	if err != nil {
		slog.DebugContext(ctx, "plugin does not support log streaming, skipping", "plugin", plugin.name)

		return nil //nolint: nilerr
	}

	logStreamingClient, ok := logStreamingPlugin.(bonkv0.LogStreamingServiceClient)
	if !ok {
		panic(
			fmt.Sprintf(
				"plugin %s reports supporting log streaming but client returned was of the wrong type: %s",
				plugin.name,
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

			slogHandler := slog.Default().With("plugin", plugin.name).Handler()
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
