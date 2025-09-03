// Copyright © 2025 Colden Cullen
// SPDX-License-Identifier: MIT

package main // import "go.bonk.build/plugins/k8s/kustomize"

import (
	"context"
	"fmt"
	"os"
	"path"

	"sigs.k8s.io/kustomize/api/konfig"
	"sigs.k8s.io/kustomize/api/krusty"
	"sigs.k8s.io/kustomize/api/types"
	"sigs.k8s.io/kustomize/kyaml/filesys"
	"sigs.k8s.io/kustomize/kyaml/yaml"

	bonk "go.bonk.build/api/go"
)

const output = "kustomized.yaml"

type Params struct {
	Kustomization types.Kustomization `json:"-"`
}

type Executor_Kustomize struct{}

func (Executor_Kustomize) Execute(
	ctx context.Context,
	task bonk.TypedTask[Params],
	res *bonk.Result,
) error {
	outDir, ok := ctx.Value("outDir").(string)
	if !ok {
		panic("no outdir!")
	}

	// Apply resources and any needed fixes
	task.Args.Kustomization.Resources = task.Inputs
	task.Args.Kustomization.FixKustomization()

	// Write out the kustomization.yaml file
	outFile, err := os.Create(path.Join(outDir, konfig.DefaultKustomizationFileName()))
	if err != nil {
		return fmt.Errorf("failed to open kustomization file: %w", err)
	}

	enc := yaml.NewEncoder(outFile)

	err = enc.Encode(task.Args.Kustomization)
	if err != nil {
		return fmt.Errorf("failed to encode kustomization file as yaml: %w", err)
	}

	err = enc.Close()
	if err != nil {
		return fmt.Errorf("failed to close yaml encoder: %w", err)
	}
	err = outFile.Close()
	if err != nil {
		return fmt.Errorf("failed to close yaml file writer: %w", err)
	}

	// Perform the kustomization
	options := krusty.MakeDefaultOptions()
	options.LoadRestrictions = types.LoadRestrictionsNone
	kusty := krusty.MakeKustomizer(options)

	resMap, err := kusty.Run(filesys.MakeFsOnDisk(), outDir)
	if err != nil {
		return fmt.Errorf("failed to perform kustomization: %w", err)
	}

	// Save the result
	resYaml, err := resMap.AsYaml()
	if err != nil {
		return fmt.Errorf("failed to encode kustomized content as yaml: %w", err)
	}

	err = os.WriteFile(path.Join(outDir, output), resYaml, 0o600)
	if err != nil {
		return fmt.Errorf("failed to write kustomized content to file: %w", err)
	}

	res.Outputs = []string{output}

	return nil
}

var Plugin = bonk.NewPlugin(func(plugin *bonk.Plugin) error {
	err := plugin.RegisterExecutor(
		"Kustomize",
		bonk.WrapTypedExecutor(*plugin.Cuectx, Executor_Kustomize{}),
	)
	if err != nil {
		return fmt.Errorf("failed to register Test executor: %w", err)
	}

	return nil
})

func main() {
	Plugin.Serve()
}
