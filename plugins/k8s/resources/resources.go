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

	"go.bonk.build/pkg/executor/plugin"
	"go.bonk.build/pkg/task"
)

const output = "resources.yaml"

type Params struct {
	Resources any `cue:"[...]" json:"resources"`
}

type Executor_Resources struct {
	task.NoopSessionManager
}

func (Executor_Resources) Execute(
	ctx context.Context,
	task *task.Task,
	args *Params,
	res *task.Result,
) error {
	if len(task.Inputs) > 0 {
		return errors.New("resources task does not accept inputs")
	}

	file, err := task.OutputFS().Create(output)
	if err != nil {
		return fmt.Errorf("failed to create resources yaml: %w", err)
	}

	encoder := yaml.NewEncoder(file)

	switch value := reflect.ValueOf(args.Resources); value.Type().Kind() {
	// Slices and arrays need to be written over multiple calls to force them into separate documents.
	case reflect.Slice, reflect.Array:
		for _, val := range value.Seq2() {
			multierr.AppendInto(&err, encoder.Encode(val.Interface()))
		}
		if err != nil {
			return fmt.Errorf("failed to encode resources as yaml: %w", err)
		}

	default:
		err = encoder.Encode(args.Resources)
		if err != nil {
			return fmt.Errorf("failed to encode resources to file: %w", err)
		}
	}

	res.Outputs = []string{output}

	multierr.AppendInto(&err, encoder.Close())
	multierr.AppendInto(&err, file.Close())

	return err //nolint:wrapcheck
}

var Plugin = plugin.NewPlugin("resources",
	plugin.WithExecutor("Resources", Executor_Resources{}),
)

func main() {
	Plugin.Serve()
}
