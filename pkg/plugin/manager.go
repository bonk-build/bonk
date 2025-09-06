// Copyright © 2025 Colden Cullen
// SPDX-License-Identifier: MIT

package plugin // import "go.bonk.build/pkg/plugin"

import (
	"context"
	"fmt"
	"log/slog"
	"os/exec"
	"path"
	"sync"

	"go.uber.org/multierr"

	"cuelang.org/go/cue"

	"github.com/ValerySidorin/shclog"

	goplugin "github.com/hashicorp/go-plugin"

	"go.bonk.build/pkg/task"
)

type ExecutorRegistrar interface {
	RegisterExecutors(execs ...task.Executor) error
	UnregisterExecutors(names ...string)
}

type PluginManager struct {
	mu      sync.RWMutex
	plugins map[string]Plugin

	cuectx   *cue.Context
	executor ExecutorRegistrar
}

func NewPluginManager(cuectx *cue.Context, executor ExecutorRegistrar) *PluginManager {
	return &PluginManager{
		plugins:  make(map[string]Plugin),
		cuectx:   cuectx,
		executor: executor,
	}
}

func (pm *PluginManager) StartPlugin(ctx context.Context, pluginPath string) error {
	pluginName := path.Base(pluginPath)

	process := goplugin.NewClient(&goplugin.ClientConfig{
		HandshakeConfig: Handshake,
		Plugins: map[string]goplugin.Plugin{
			"executor":      &executorPlugin{},
			"log_streaming": &logStreamingPluginClient{},
		},
		Cmd: exec.CommandContext(ctx, "go", "run", pluginPath),
		AllowedProtocols: []goplugin.Protocol{
			goplugin.ProtocolGRPC,
		},
		Logger: shclog.New(slog.Default()),
	})

	plug := Plugin{
		name: pluginName,
	}
	err := plug.Configure(ctx, pm.cuectx, process, pm.executor)
	if err != nil {
		return fmt.Errorf("failed to create plugin %s: %w", pluginName, err)
	}

	pm.mu.Lock()
	pm.plugins[pluginName] = plug
	pm.mu.Unlock()

	return nil
}

func (pm *PluginManager) StartPlugins(ctx context.Context, pluginPath ...string) error {
	var (
		pluginWaiter sync.WaitGroup
		allErrs      error
		errMu        sync.Mutex
	)

	for _, plugin := range pluginPath {
		pluginWaiter.Add(1)
		go func() {
			err := pm.StartPlugin(ctx, plugin)

			errMu.Lock()
			multierr.AppendInto(&allErrs, err)
			errMu.Unlock()

			pluginWaiter.Done()
		}()
	}

	pluginWaiter.Wait()

	return allErrs
}

func (pm *PluginManager) Shutdown() {
	pm.mu.RLock()
	for pluginName := range pm.plugins {
		pm.executor.UnregisterExecutors(pluginName)
	}
	pm.mu.RUnlock()

	pm.mu.Lock()
	for _, plugin := range pm.plugins {
		plugin.pluginClient.Kill()
	}
	pm.plugins = make(map[string]Plugin)
	pm.mu.Unlock()
}
