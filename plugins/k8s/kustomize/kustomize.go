// Copyright © 2025 Colden Cullen
// SPDX-License-Identifier: MIT

package main // import "go.bonk.build/plugins/k8s/kustomize"

import (
	"context"
	"fmt"

	"sigs.k8s.io/kustomize/api/internal/target"
	"sigs.k8s.io/kustomize/api/konfig"
	"sigs.k8s.io/kustomize/api/krusty"
	"sigs.k8s.io/kustomize/api/pkg/loader"
	"sigs.k8s.io/kustomize/api/provider"
	"sigs.k8s.io/kustomize/api/resmap"
	"sigs.k8s.io/kustomize/api/types"
	"sigs.k8s.io/kustomize/kyaml/yaml"

	bonk "go.bonk.build/api/go"
)

const output = "kustomized.yaml"

type Params struct {
	Kustomization types.Kustomization `json:"-"`
}

func kustomize(ctx context.Context, params *bonk.TaskParams[Params]) error {
	// Apply resources and any needed fixes
	params.Params.Kustomization.Resources = params.Inputs
	params.Params.Kustomization.FixKustomization()

	// Write out the kustomization.yaml file
	kustFile, err := params.TaskFs.Create(konfig.DefaultKustomizationFileName())
	if err != nil {
		return fmt.Errorf("failed to open kustomization file: %w", err)
	}

	enc := yaml.NewEncoder(kustFile)

	err = enc.Encode(params.Params.Kustomization)
	if err != nil {
		return fmt.Errorf("failed to encode kustomization file as yaml: %w", err)
	}

	err = enc.Close()
	if err != nil {
		return fmt.Errorf("failed to close yaml encoder: %w", err)
	}
	err = kustFile.Close()
	if err != nil {
		return fmt.Errorf("failed to close yaml file writer: %w", err)
	}

	// Perform the kustomization
	options := krusty.MakeDefaultOptions()
	options.LoadRestrictions = types.LoadRestrictionsNone
	kusty := krusty.MakeKustomizer(options)

	///////

	depProvider := provider.NewDepProvider()
	resmapFactory := resmap.NewFactory(depProvider.GetResourceFactory())

	ldr := loader.NewFileLoaderAtRoot(KyamlFilesys{Fs: params.ProjectFs})
	defer ldr.Cleanup()

	kt := target.NewKustTarget(
		ldr,
		depProvider.GetFieldValidator(),
		resmapFactory,
		// The plugin configs are always located on disk, regardless of the fSys passed in
		// pLdr.NewLoader(b.options.PluginConfig, resmapFactory, filesys.MakeFsOnDisk()),
		nil,
	)
	err = kt.Load()

	res, err := kusty.Run(KyamlFilesys{Fs: params.ProjectFs}, "")
	if err != nil {
		return fmt.Errorf("failed to perform kustomization: %w", err)
	}

	//////////

	// Save the result
	resYaml, err := res.AsYaml()
	if err != nil {
		return fmt.Errorf("failed to encode kustomized content as yaml: %w", err)
	}

	outFile, err := params.TaskFs.Create(output)
	if err != nil {
		return fmt.Errorf("failed to create kustomized file: %w", err)
	}
	_, err = outFile.Write(resYaml)
	if err != nil {
		return fmt.Errorf("failed to write kustomized content to file: %w", err)
	}

	bonk.AddOutputs(ctx, output)

	return nil
}

func main() {
	bonk.Serve(
		bonk.NewExecutor(
			"Kustomize",
			kustomize,
		),
	)
}
