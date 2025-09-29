// Copyright © 2025 Colden Cullen
// SPDX-License-Identifier: MIT

package plugin

import (
	"context"
	"path"
	"sync"

	"go.uber.org/multierr"

	"go.bonk.build/pkg/executor"
	"go.bonk.build/pkg/executor/tree"
)

// PluginClientManager manages a set of [PluginClient]s and functions as a distributing [tree.ExecutorTree].
type PluginClientManager interface {
	executor.Executor

	// NOTE(colden): these should eventually be moved out of here
	RegisterExecutor(name string, exec executor.Executor) error
	UnregisterExecutors(names ...string)

	StartPlugins(ctx context.Context, plugins ...string) error
	Shutdown(ctx context.Context)
}

type pluginClientManager struct {
	tree.ExecutorTree

	mu sync.Mutex
}

// NewPluginClientManager creates a new empty [PluginClientManager].
func NewPluginClientManager() PluginClientManager {
	return &pluginClientManager{
		ExecutorTree: tree.New(),
	}
}

// StartPlugin calls [NewPluginClient] and registers the executor by the plugin's name.
func (pm *pluginClientManager) StartPlugin(ctx context.Context, pluginPath string) error {
	plug, err := NewPluginClient(ctx, pluginPath)
	if err != nil {
		return err
	}

	pluginName := path.Base(pluginPath)

	pm.mu.Lock()
	err = pm.RegisterExecutor(pluginName, plug)
	pm.mu.Unlock()

	if err != nil {
		return err //nolint:wrapcheck
	}

	return nil
}

// StartPlugins calls [StartPlugin] in parallel per pluginPath.
func (pm *pluginClientManager) StartPlugins(ctx context.Context, pluginPath ...string) error {
	var (
		pluginWaiter sync.WaitGroup
		allErrs      error
		errMu        sync.Mutex
	)

	for _, plugin := range pluginPath {
		pluginWaiter.Go(func() {
			err := pm.StartPlugin(ctx, plugin)

			errMu.Lock()
			multierr.AppendInto(&allErrs, err)
			errMu.Unlock()
		})
	}

	pluginWaiter.Wait()

	return allErrs
}

// Shutdown does de initialization and kills all plugin processes.
func (pm *pluginClientManager) Shutdown(context.Context) {
	pm.mu.Lock()
	pm.ForEachExecutor(func(name string, exec executor.Executor) {
		pm.UnregisterExecutors(name)

		if plug, ok := exec.(*PluginClient); ok {
			plug.Shutdown()
		}
	})
	pm.mu.Unlock()
}
