// Copyright © 2025 Colden Cullen
// SPDX-License-Identifier: MIT

package task

import (
	"bytes"
	"crypto/sha256"
	"encoding/json"
	"fmt"
	"hash"
	"io"
	"os"
	"reflect"

	"go.uber.org/multierr"

	"cuelang.org/go/cue"
)

const StateFile = "state.json"

type state struct {
	// Cache provided executor & outputs
	Executor string      `json:"executor,omitempty"`
	Inputs   []string    `json:"inputs,omitempty"`
	Result   *TaskResult `json:"result,omitempty"`

	ParamsChecksum []byte `json:"paramsChecksum,omitempty"`
	InputsChecksum []byte `json:"inputChecksum,omitempty"`
	ResultChecksum []byte `json:"resultChecksum,omitempty"`
}

func NewState(
	executor string,
	params cue.Value,
	root *os.Root,
	inputs []string,
	result *TaskResult,
) (*state, error) {
	var err error
	state := &state{}
	state.Executor = executor
	state.Inputs = inputs
	state.Result = result

	hasher := sha256.New()

	// Hash the parameters
	state.ParamsChecksum, err = hashCueValue(hasher, params)
	if err != nil {
		return state, err
	}

	// Hash the input files
	state.InputsChecksum, err = hashFiles(hasher, nil, inputs)
	if err != nil {
		return state, err
	}

	// Hash the input files
	state.ResultChecksum, err = hashResult(hasher, root, result)
	if err != nil {
		return state, err
	}

	return state, err
}

func LoadState(root *os.Root) (*state, error) {
	file, err := root.Open(StateFile)
	if err != nil {
		return nil, fmt.Errorf("failed to open state file %s: %w", StateFile, err)
	}
	encoder := json.NewDecoder(file)

	state := &state{}
	err = encoder.Decode(state)
	if err != nil {
		return nil, fmt.Errorf("failed to decode state file %s: %w", StateFile, err)
	}

	return state, nil
}

func (s *state) Save(root *os.Root) error {
	file, err := root.Create(StateFile)
	if err != nil {
		return fmt.Errorf("failed to open state file %s: %w", StateFile, err)
	}
	encoder := json.NewEncoder(file)

	err = encoder.Encode(s)
	if err != nil {
		return fmt.Errorf("failed to encode state file %s: %w", StateFile, err)
	}

	return nil
}

func (s *state) DetectMismatches(
	executor string,
	params cue.Value,
	root *os.Root,
	inputs []string,
) []string {
	var mismatches []string
	hasher := sha256.New()

	if executor != s.Executor {
		mismatches = append(mismatches, "executor")
	}

	paramsChecksum, err := hashCueValue(hasher, params)
	if err != nil || !bytes.Equal(paramsChecksum, s.ParamsChecksum) {
		mismatches = append(mismatches, "params-checksum")
	}
	hasher.Reset()

	if !reflect.DeepEqual(inputs, s.Inputs) {
		mismatches = append(mismatches, "inputs")
	}
	inputsChecksum, err := hashFiles(hasher, nil, inputs)
	if err != nil || !bytes.Equal(inputsChecksum, s.InputsChecksum) {
		mismatches = append(mismatches, "inputs-checksum")
	}
	hasher.Reset()

	resultChecksum, err := hashResult(hasher, root, s.Result)
	if err != nil || !bytes.Equal(resultChecksum, s.ResultChecksum) {
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

func hashFiles(hasher hash.Hash, root *os.Root, files []string) ([]byte, error) {
	var err error
	for _, fileName := range files {
		var file *os.File
		var openErr error
		if root != nil {
			file, openErr = root.Open(fileName)
		} else {
			file, openErr = os.Open(fileName)
		}
		if multierr.AppendInto(&err, openErr) {
			return nil, fmt.Errorf("failed to open input file %s: %w", fileName, err)
		}

		_, hashErr := io.Copy(hasher, file)
		if multierr.AppendInto(&err, hashErr) {
			return nil, fmt.Errorf("failed to hash input file %s: %w", fileName, err)
		}
	}

	result := hasher.Sum(nil)

	return result, err
}

func hashResult(hasher hash.Hash, root *os.Root, result *TaskResult) ([]byte, error) {
	if result == nil {
		return nil, nil
	}

	var err error

	_, hashErr := hashFiles(hasher, root, result.Outputs)
	multierr.AppendInto(&err, hashErr)

	// Convert the followups to json for easy hashing
	bytes, hashErr := json.Marshal(result.FollowupTasks)
	if multierr.AppendInto(&err, hashErr) {
		hasher.Write(bytes)
	}

	if err != nil {
		return nil, fmt.Errorf("failed to hash task result: %w", err)
	}

	return hasher.Sum(nil), nil
}
