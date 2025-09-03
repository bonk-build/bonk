// Copyright © 2025 Colden Cullen
// SPDX-License-Identifier: MIT

package scheduler

import (
	"bytes"
	"crypto/sha256"
	"encoding/json"
	"errors"
	"fmt"
	"hash"
	"io"
	"log/slog"
	"reflect"

	"cuelang.org/go/cue"

	"github.com/spf13/afero"

	"go.bonk.build/pkg/task"
)

const StateFile = "state.json"

type state struct {
	// Cache provided executor & outputs
	Executor string       `json:"executor,omitempty"`
	Inputs   []string     `json:"inputs,omitempty"`
	Result   *task.Result `json:"result,omitempty"`

	ParamsChecksum []byte `json:"paramsChecksum,omitempty"`
	InputsChecksum []byte `json:"inputChecksum,omitempty"`
	ResultChecksum []byte `json:"resultChecksum,omitempty"`
}

func SaveState(task *task.Task, result *task.Result) error {
	file, err := task.OutputFs.Create(StateFile)
	if err != nil {
		return fmt.Errorf("failed to open state file %s: %w", StateFile, err)
	}
	encoder := json.NewEncoder(file)

	state := state{
		Executor: task.Executor(),
		Inputs:   task.Inputs,
		Result:   result,
	}

	hasher := sha256.New()

	// Hash the parameters
	state.ParamsChecksum, err = hashCueValue(hasher, task.Params)
	if err != nil {
		return err
	}
	hasher.Reset()

	// Hash the input files
	state.InputsChecksum, err = hashFiles(hasher, task.ProjectFs, task.Inputs)
	if err != nil {
		return err
	}
	hasher.Reset()

	// Hash the result
	state.ResultChecksum, err = hashResult(hasher, task.OutputFs, result)
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

func DetectStateMismatches(task *task.Task) []string {
	file, err := task.OutputFs.Open(StateFile)
	if err != nil {
		return []string{"<state missing>"}
	}
	encoder := json.NewDecoder(file)

	state := &state{}
	err = encoder.Decode(state)
	if err != nil {
		slog.Error("failed to decode json state", "error", err)

		return []string{"<state decode failed>"}
	}

	var mismatches []string
	hasher := sha256.New()

	if task.Executor() != state.Executor {
		mismatches = append(mismatches, "executor")
	}

	paramsChecksum, err := hashCueValue(hasher, task.Params)
	if err != nil || !bytes.Equal(paramsChecksum, state.ParamsChecksum) {
		mismatches = append(mismatches, "params-checksum")
	}
	hasher.Reset()

	if !reflect.DeepEqual(task.Inputs, state.Inputs) {
		mismatches = append(mismatches, "inputs")
	}
	inputsChecksum, err := hashFiles(hasher, task.ProjectFs, task.Inputs)
	if err != nil || !bytes.Equal(inputsChecksum, state.InputsChecksum) {
		mismatches = append(mismatches, "inputs-checksum")
	}
	hasher.Reset()

	resultChecksum, err := hashResult(hasher, task.OutputFs, state.Result)
	if err != nil {
		mismatches = append(mismatches, "!result-checksum-failed!")
	} else if !bytes.Equal(resultChecksum, state.ResultChecksum) {
		mismatches = append(mismatches, "result-checksum")
	}
	hasher.Reset()

	return mismatches
}

func hashCueValue(hasher hash.Hash, params cue.Value) ([]byte, error) {
	paramsJSON, err := params.MarshalJSON()
	if err != nil {
		return nil, fmt.Errorf("failed to marshal params to json: %w", err)
	}

	return hasher.Sum(paramsJSON), nil
}

func hashFiles(hasher hash.Hash, root afero.Fs, files []string) ([]byte, error) {
	for _, fileName := range files {
		file, err := root.Open(fileName)
		if err != nil {
			return nil, fmt.Errorf("failed to open input file %s: %w", fileName, err)
		}

		_, err = io.Copy(hasher, file)
		if err != nil {
			return nil, fmt.Errorf("failed to hash input file %s: %w", fileName, err)
		}
	}

	result := hasher.Sum(nil)

	return result, nil
}

func hashResult(hasher hash.Hash, root afero.Fs, result *task.Result) ([]byte, error) {
	_, err := hashFiles(hasher, root, result.Outputs)
	if err != nil {
		return nil, errors.New("failed to hash output files")
	}

	// Convert the followups to json for easy hashing
	bytes, err := json.Marshal(result.FollowupTasks)
	if err != nil {
		return nil, errors.New("failed to hash followup tasks")
	}
	hasher.Write(bytes)

	return hasher.Sum(nil), nil
}
