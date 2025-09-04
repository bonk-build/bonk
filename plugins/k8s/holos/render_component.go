// Copyright © 2025 Colden Cullen
// SPDX-License-Identifier: MIT

package holos

import (
	"context"
	"fmt"
	"log/slog"

	"cuelang.org/go/cue"
	"cuelang.org/go/cue/cuecontext"
	"cuelang.org/go/cue/load"

	core "github.com/holos-run/holos/api/core/v1alpha5"

	"go.bonk.build/pkg/executor"
	"go.bonk.build/pkg/task"
)

type Executor_RenderComponent struct {
	executor.NoopSessionManager
}

func (Executor_RenderComponent) Execute(
	ctx context.Context,
	session task.Session,
	tsk *task.Task,
	args *core.Component,
	res *task.Result,
) error {
	cuectx := cuecontext.New()
	config := load.Config{}

	// If there's more than 1 arg, use the first as the root
	if localFs, ok := session.(task.LocalSession); ok {
		config.Dir = localFs.LocalPath()
	}

	slog.InfoContext(
		ctx,
		"loading component",
		"component",
		args.Name,
		"path",
		args.Path,
		"directory",
		config.Dir,
	)

	insts := load.Instances([]string{"./" + args.Path}, &config)
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

	buildPlan := core.BuildPlan{}
	err = holosConfig.Decode(&buildPlan)
	if err != nil {
		return fmt.Errorf("failed to decode buildplan: %w", err)
	}

	if buildPlan.Spec.Disabled {
		slog.DebugContext(ctx, "buildplan is disabled, skipping", "buildplan", buildPlan.Metadata.Name)

		return nil
	}

	slog.InfoContext(ctx, "successfully described component")

	for _, artifact := range buildPlan.Spec.Artifacts {
		if artifact.Skip {
			slog.DebugContext(ctx, "artifact is skipped", "artifact", artifact.Artifact)

			continue
		}

		for _, generator := range artifact.Generators {
			switch generator.Kind {
			case "Resources":
				resources := []core.Resource{}
				for _, kind := range generator.Resources {
					for _, resource := range kind {
						resources = append(resources, resource)
					}
				}

				if len(resources) > 0 {
					res.AddFollowupTasks(task.New(
						"resources",
						"resources.Resources",
						map[string]any{
							"resources": resources,
						},
					))
				}

			case "Helm":
			case "File":

			default:
				slog.WarnContext(ctx, "unknown generator kind", "kind", generator.Kind)
			}
		}
	}

	return nil
}
