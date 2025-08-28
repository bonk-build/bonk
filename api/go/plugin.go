// Copyright © 2025 Colden Cullen
// SPDX-License-Identifier: MIT

package bonk // import "go.bonk.build/api/go"

import (
	"log/slog"

	"github.com/ValerySidorin/shclog"

	goplugin "github.com/hashicorp/go-plugin"
)

// Call from main() to start the plugin gRPC server.
func Serve(executors ...BonkExecutor) {
	executorMap := make(map[string]BonkExecutor, len(executors))
	for _, executor := range executors {
		executorMap[executor.Name] = executor
	}

	goplugin.Serve(&goplugin.ServeConfig{
		HandshakeConfig: Handshake,
		Plugins: map[string]goplugin.Plugin{
			"executor": &ExecutorServer{
				Executors: executorMap,
			},
			"log_streaming": &LogStreamingServer{},
		},
		GRPCServer: goplugin.DefaultGRPCServer,
		Logger:     shclog.New(slog.Default()),
	})
}

var Handshake = goplugin.HandshakeConfig{
	ProtocolVersion:  0,
	MagicCookieKey:   "BONK_PLUGIN",
	MagicCookieValue: "bonk the builder",
}
