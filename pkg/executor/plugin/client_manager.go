// Copyright © 2025 Colden Cullen
// SPDX-License-Identifier: MIT

package plugin // import "go.bonk.build/pkg/executor/plugin"

import (
	"context"
	"sync"

	"go.uber.org/multierr"

	"go.bonk.build/pkg/executor/tree"
	"go.bonk.build/pkg/task"
)

type PluginClientManager struct {
	tree.ExecutorManager

	mu sync.Mutex
}

func NewPluginClientManager() *PluginClientManager {
	return &PluginClientManager{
		ExecutorManager: tree.NewExecutorManager(""),
	}
}

func (pm *PluginClientManager) StartPlugin(ctx context.Context, pluginPath string) error {
	plug, err := NewPluginClient(ctx, pluginPath)
	if err != nil {
		return err
	}

	pm.mu.Lock()
	err = pm.RegisterExecutors(plug)
	pm.mu.Unlock()

	if err != nil {
		return err //nolint:wrapcheck
	}

	return nil
}

func (pm *PluginClientManager) StartPlugins(ctx context.Context, pluginPath ...string) error {
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

func (pm *PluginClientManager) Shutdown() {
	pm.mu.Lock()
	pm.ForEachExecutor(func(name string, exec task.GenericExecutor) {
		pm.UnregisterExecutors(name)

		if plug, ok := exec.(*PluginClient); ok {
			plug.Shutdown()
		}
	})
	pm.mu.Unlock()
}
