// Copyright © 2025 Colden Cullen
// SPDX-License-Identifier: MIT

package main

import (
	"context"
	"fmt"

	"sigs.k8s.io/kustomize/api/konfig"
	"sigs.k8s.io/kustomize/api/krusty"
	"sigs.k8s.io/kustomize/api/types"
	"sigs.k8s.io/kustomize/kyaml/yaml"

	"github.com/spf13/afero"

	"go.bonk.build/pkg/executor/plugin"
	"go.bonk.build/pkg/task"
)

const output = "kustomized.yaml"

type Params struct {
	Kustomization types.Kustomization `json:"-"`
}

type Executor_Kustomize struct {
	task.NoopSessionManager
}

func (Executor_Kustomize) Execute(
	ctx context.Context,
	task *task.Task[Params],
	res *task.Result,
) error {
	// Apply resources and any needed fixes
	task.Args.Kustomization.Resources = task.Inputs
	task.Args.Kustomization.FixKustomization()

	kustomFs := afero.NewCopyOnWriteFs(task.Session.SourceFS(), afero.NewMemMapFs())

	// Write out the kustomization.yaml file
	kustFile, err := kustomFs.Create("/" + konfig.DefaultKustomizationFileName())
	if err != nil {
		return fmt.Errorf("failed to open kustomization file: %w", err)
	}

	enc := yaml.NewEncoder(kustFile)

	err = enc.Encode(task.Args.Kustomization)
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

	resMap, err := kusty.Run(KyamlFilesys{kustomFs}, "/")
	if err != nil {
		return fmt.Errorf("failed to perform kustomization: %w", err)
	}

	// Save the result
	resYaml, err := resMap.AsYaml()
	if err != nil {
		return fmt.Errorf("failed to encode kustomized content as yaml: %w", err)
	}

	outFile, err := task.OutputFS().Create(output)
	if err != nil {
		return fmt.Errorf("failed to create kustomized file: %w", err)
	}
	_, err = outFile.Write(resYaml)
	if err != nil {
		return fmt.Errorf("failed to write kustomized content to file: %w", err)
	}

	res.Outputs = []string{output}

	return nil
}

var Plugin = plugin.NewPlugin("kustomize",
	plugin.WithExecutor("Kustomize", Executor_Kustomize{}),
)

func main() {
	Plugin.Serve()
}
