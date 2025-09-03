// Copyright © 2025 Colden Cullen
// SPDX-License-Identifier: MIT

package main // import "go.bonk.build/plugins/k8s/resources"

import (
	"context"
	"errors"
	"fmt"
	"os"
	"path"

	"cuelang.org/go/cue"
	"cuelang.org/go/pkg/encoding/yaml"

	bonk "go.bonk.build/api/go"
)

const output = "resources.yaml"

type Params struct {
	Resources cue.Value `cue:"[...]" json:"resources"`
}

type Executor_Resources struct{}

func (Executor_Resources) Execute(
	ctx context.Context,
	task bonk.TypedTask[Params],
	res *bonk.Result,
) error {
	outDir, ok := ctx.Value("outDir").(string)
	if !ok {
		panic("no outdir!")
	}

	if len(task.Inputs) > 0 {
		return errors.New("resources task does not accept inputs")
	}

	resourcesYaml, err := yaml.MarshalStream(task.Args.Resources)
	if err != nil {
		return fmt.Errorf("failed to marshal resources into yaml: %w", err)
	}

	err = os.WriteFile(path.Join(outDir, output), []byte(resourcesYaml), 0o600)
	if err != nil {
		return fmt.Errorf("failed to write resources yaml to disk: %w", err)
	}

	res.Outputs = []string{output}

	return nil
}

var Plugin = bonk.NewPlugin(func(plugin *bonk.Plugin) error {
	err := plugin.RegisterExecutor(
		"Resources",
		bonk.WrapTypedExecutor(*plugin.Cuectx, Executor_Resources{}),
	)
	if err != nil {
		return fmt.Errorf("failed to register Test executor: %w", err)
	}

	return nil
})

func main() {
	Plugin.Serve()
}
