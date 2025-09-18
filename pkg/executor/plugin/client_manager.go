// Copyright © 2025 Colden Cullen
// SPDX-License-Identifier: MIT

package plugin // import "go.bonk.build/pkg/executor/plugin"

import (
	"context"
	"path"
	"sync"

	"go.uber.org/multierr"

	"go.bonk.build/pkg/executor/tree"
	"go.bonk.build/pkg/task"
)

type PluginClientManager interface {
	task.Executor

	// NOTE(colden): these should eventually be moved out of here
	RegisterExecutor(name string, exec task.Executor) error
	UnregisterExecutors(names ...string)

	StartPlugins(ctx context.Context, plugins ...string) error
	Shutdown(ctx context.Context)
}

type pluginClientManager struct {
	tree.ExecutorTree

	mu sync.Mutex
}

func NewPluginClientManager() PluginClientManager {
	return &pluginClientManager{
		ExecutorTree: tree.New(),
	}
}

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

func (pm *pluginClientManager) StartPlugins(ctx context.Context, pluginPath ...string) error {
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

func (pm *pluginClientManager) Shutdown(context.Context) {
	pm.mu.Lock()
	pm.ForEachExecutor(func(name string, exec task.Executor) {
		pm.UnregisterExecutors(name)

		if plug, ok := exec.(*PluginClient); ok {
			plug.Shutdown()
		}
	})
	pm.mu.Unlock()
}
