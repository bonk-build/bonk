// Copyright © 2025 Colden Cullen
// SPDX-License-Identifier: MIT

package main // import "go.bonk.build/plugins/k8s/resources"

import (
	"context"
	"errors"
	"fmt"
	"reflect"

	"go.uber.org/multierr"
	"go.yaml.in/yaml/v4"

	bonk "go.bonk.build/api/go"
)

const output = "resources.yaml"

type Params struct {
	Resources any `cue:"[...]" json:"resources"`
}

type Executor_Resources struct{}

func (Executor_Resources) Name() string {
	return "Resources"
}

func (Executor_Resources) Execute(
	ctx context.Context,
	task bonk.Task[Params],
	res *bonk.Result,
) error {
	if len(task.Inputs) > 0 {
		return errors.New("resources task does not accept inputs")
	}

	file, err := task.OutputFs.Create(output)
	if err != nil {
		return fmt.Errorf("failed to create resources yaml: %w", err)
	}

	encoder := yaml.NewEncoder(file)

	switch value := reflect.ValueOf(task.Args.Resources); value.Type().Kind() {
	// Slices and arrays need to be written over multiple calls to force them into separate documents.
	case reflect.Slice, reflect.Array:
		for _, val := range value.Seq2() {
			multierr.AppendInto(&err, encoder.Encode(val.Interface()))
		}
		if err != nil {
			return fmt.Errorf("failed to encode resources as yaml: %w", err)
		}

	default:
		err = encoder.Encode(task.Args.Resources)
		if err != nil {
			return fmt.Errorf("failed to encode resources to file: %w", err)
		}
	}

	res.Outputs = []string{output}

	multierr.AppendInto(&err, encoder.Close())
	multierr.AppendInto(&err, file.Close())

	return err //nolint:wrapcheck
}

var Plugin = bonk.NewPlugin("resources", func(plugin *bonk.Plugin) error {
	err := plugin.RegisterExecutors(
		bonk.BoxExecutor(Executor_Resources{}),
	)
	if err != nil {
		return fmt.Errorf("failed to register Test executor: %w", err)
	}

	return nil
})

func main() {
	Plugin.Serve()
}
