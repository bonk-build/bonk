// Copyright © 2025 Colden Cullen
// SPDX-License-Identifier: MIT

package statecheck

import (
	"encoding/json"
	"fmt"
	"hash"
	"hash/fnv"
	"io"
	"log/slog"
	"reflect"

	"github.com/gohugoio/hashstructure"
	"github.com/spf13/afero"

	"go.bonk.build/pkg/task"
)

const StateFile = "state.json"

type state struct {
	// Cache provided executor & outputs
	Executor string       `json:"executor,omitempty"`
	Inputs   []string     `json:"inputs,omitempty"`
	Result   *task.Result `json:"result,omitempty"`

	ArgumentsChecksum uint64 `json:"argumentsChecksum,omitempty"`
	InputsChecksum    uint64 `json:"inputChecksum,omitempty"`
	OutputChecksum    uint64 `json:"resultChecksum,omitempty"`
	FollowupChecksum  uint64 `json:"followupChecksum,omitempty"`
}

func SaveState[Params any](task *task.Task[Params], result *task.Result) error {
	err := task.OutputFS().MkdirAll("", 0o750)
	if err != nil {
		return fmt.Errorf("failed to create task directory: %w", err)
	}

	file, err := task.OutputFS().Create(StateFile)
	if err != nil {
		return fmt.Errorf("failed to open state file %s: %w", StateFile, err)
	}
	encoder := json.NewEncoder(file)

	state := state{
		Executor: task.Executor,
		Inputs:   task.Inputs,
		Result:   result,
	}

	hasher := fnv.New64()

	// Hash the parameters
	state.ArgumentsChecksum, err = hashAnyValue(hasher, task.Args)
	if err != nil {
		return err
	}
	hasher.Reset()

	// Hash the input files
	state.InputsChecksum, err = hashFiles(hasher, task.Session.SourceFS(), task.Inputs)
	if err != nil {
		return err
	}
	hasher.Reset()

	// Hash the output files
	state.OutputChecksum, err = hashFiles(hasher, task.OutputFS(), result.Outputs)
	if err != nil {
		return err
	}

	state.FollowupChecksum, err = hashAnyValue(hasher, result.FollowupTasks)
	if err != nil {
		return err
	}
	hasher.Reset()

	err = encoder.Encode(state)
	if err != nil {
		return fmt.Errorf("failed to encode state file %s: %w", StateFile, err)
	}

	return nil
}

func DetectStateMismatches[Params any](task *task.Task[Params]) []string {
	file, err := task.OutputFS().Open(StateFile)
	if err != nil {
		return []string{"<state missing>"}
	}
	encoder := json.NewDecoder(file)

	state := state{}
	err = encoder.Decode(&state)
	if err != nil {
		slog.Error("failed to decode json state", "error", err)

		return []string{"<state decode failed>"}
	}

	var mismatches []string
	hasher := fnv.New64()

	if task.Executor != state.Executor {
		mismatches = append(mismatches, "executor")
	}

	argsChecksum, err := hashAnyValue(hasher, task.Args)
	if err != nil || argsChecksum != state.ArgumentsChecksum {
		mismatches = append(mismatches, "arguments-checksum")
	}
	hasher.Reset()

	if !reflect.DeepEqual(task.Inputs, state.Inputs) {
		mismatches = append(mismatches, "inputs")
	}
	inputsChecksum, err := hashFiles(hasher, task.Session.SourceFS(), task.Inputs)
	if err != nil || inputsChecksum != state.InputsChecksum {
		mismatches = append(mismatches, "inputs-checksum")
	}
	hasher.Reset()

	outputChecksum, err := hashFiles(hasher, task.OutputFS(), state.Result.Outputs)
	if err != nil {
		mismatches = append(mismatches, "!output-checksum-failed!")
	} else if outputChecksum != state.OutputChecksum {
		mismatches = append(mismatches, "output-checksum")
	}
	hasher.Reset()

	followupChecksum, err := hashAnyValue(hasher, state.Result.FollowupTasks)
	if err != nil {
		mismatches = append(mismatches, "!followup-checksum-failed!")
	} else if followupChecksum != state.FollowupChecksum {
		mismatches = append(mismatches, "followup-checksum")
	}
	hasher.Reset()

	return mismatches
}

func hashAnyValue(hasher hash.Hash64, params any) (uint64, error) {
	if params == nil {
		return 0, nil
	}

	return hashstructure.Hash(params, &hashstructure.HashOptions{ //nolint:wrapcheck
		Hasher: hasher,
	})
}

func hashFiles(hasher hash.Hash64, root afero.Fs, files []string) (uint64, error) {
	for _, pattern := range files {
		matches, err := afero.Glob(root, pattern)
		if err != nil {
			return 0, fmt.Errorf("failed to expand glob '%s': %w", pattern, err)
		}

		for _, fileName := range matches {
			file, err := root.Open(fileName)
			if err != nil {
				return 0, fmt.Errorf("failed to open input file %s: %w", fileName, err)
			}

			_, err = io.Copy(hasher, file)
			if err != nil {
				return 0, fmt.Errorf("failed to hash input file %s: %w", fileName, err)
			}
		}
	}

	return hasher.Sum64(), nil
}
