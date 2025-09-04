// Copyright © 2025 Colden Cullen
// SPDX-License-Identifier: MIT

package main

import (
	"context"
	"fmt"
	"log/slog"

	"cuelang.org/go/cue"
	"cuelang.org/go/cue/cuecontext"
	"cuelang.org/go/cue/load"

	"github.com/spf13/afero"

	core "github.com/holos-run/holos/api/core/v1alpha5"

	bonk "go.bonk.build/api/go"
	"go.bonk.build/pkg/task"
)

type Params_RenderPlatform struct {
	Platform string `json:"platform"`
}

type Executor_RenderPlatform struct{}

func (Executor_RenderPlatform) Name() string {
	return "RenderPlatform"
}

func (exe Executor_RenderPlatform) Execute(
	ctx context.Context,
	tsk bonk.TypedTask[Params_RenderPlatform],
	res *bonk.Result,
) error {
	cuectx := cuecontext.New()
	config := load.Config{}

	// If there's more than 1 arg, use the first as the root
	if localFs, ok := tsk.Session.FS().(*afero.BasePathFs); ok {
		var err error
		config.Dir, err = localFs.RealPath("")
		if err != nil {
			return fmt.Errorf("failed to get directory path: %w", err)
		}
	}

	slog.InfoContext(ctx, "loading platform", "directory", config.Dir, "platform", tsk.Args.Platform)

	insts := load.Instances([]string{"./" + tsk.Args.Platform}, &config)
	values, err := cuectx.BuildInstances(insts)
	if err != nil {
		return fmt.Errorf("failed to load platform: %w", err)
	}

	// Unify all of the values into a single source of truth
	value := cue.Value{}
	for _, valuePart := range values {
		value = value.Unify(valuePart)
	}

	holosConfig := value.LookupPath(cue.MakePath(cue.Str("holos")))
	if holosConfig.Err() != nil {
		return fmt.Errorf("failed to find holos config: %w", holosConfig.Err())
	}

	platform := core.Platform{}
	err = holosConfig.Decode(&platform)
	if err != nil {
		return fmt.Errorf("failed to decode platform: %w", err)
	}

	for _, component := range platform.Spec.Components {
		res.FollowupTasks = append(res.FollowupTasks, task.NewTyped(
			tsk.Session,
			"holos.RenderComponent",
			"component."+component.Name,
			cuectx,
			component,
			component.Path+"/*.cue",
		).Task)
	}

	return nil
}
