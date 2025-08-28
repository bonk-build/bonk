// Copyright © 2025 Colden Cullen
// SPDX-License-Identifier: MIT

package bonk // import "go.bonk.build/api/go"

import (
	"context"
	"fmt"
	"os"

	"go.uber.org/multierr"

	"google.golang.org/grpc"

	"cuelang.org/go/cue"
	"cuelang.org/go/cue/cuecontext"
	"cuelang.org/go/encoding/gocode/gocodec"

	goplugin "github.com/hashicorp/go-plugin"
	slogctx "github.com/veqryn/slog-context"

	bonkv0 "go.bonk.build/api/go/proto/bonk/v0"
)

var cuectx = cuecontext.New()

// The inputs passed to a task executor.
type TaskParams[Params any] struct {
	Inputs []string
	Params Params
	OutDir string
}

// Represents a executor capable of performing tasks.
type BonkExecutor struct {
	Name         string
	Outputs      []string
	ParamsSchema cue.Value
	Exec         func(context.Context, TaskParams[cue.Value]) ([]string, error)
}

// Factory to create a new task executor.
func NewExecutor[Params any](
	name string,
	exec func(context.Context, *TaskParams[Params]) ([]string, error),
) BonkExecutor {
	zero := new(Params)

	schema := cuectx.EncodeType(*zero)
	if schema.Err() != nil {
		panic(schema.Err())
	}

	return BonkExecutor{
		Name:         name,
		ParamsSchema: schema,
		Exec: func(ctx context.Context, paramsCue TaskParams[cue.Value]) ([]string, error) {
			params := new(TaskParams[Params])
			params.Inputs = paramsCue.Inputs
			params.OutDir = paramsCue.OutDir
			err := paramsCue.Params.Decode(&params.Params)
			if err != nil {
				return nil, fmt.Errorf("failed to decode task parameters: %w", err)
			}

			return exec(ctx, params)
		},
	}
}

type ExecutorServer struct {
	goplugin.NetRPCUnsupportedPlugin
	goplugin.GRPCPlugin

	Executors map[string]BonkExecutor
}

func (p *ExecutorServer) GRPCServer(_ *goplugin.GRPCBroker, s *grpc.Server) error {
	bonkv0.RegisterExecutorServiceServer(s, &executorGRPCServer{
		decodeCodec: gocodec.New(cuectx, &gocodec.Config{}),
		executors:   p.Executors,
	})

	return nil
}

func (p *ExecutorServer) GRPCClient(
	_ context.Context,
	_ *goplugin.GRPCBroker,
	c *grpc.ClientConn,
) (any, error) {
	return bonkv0.NewExecutorServiceClient(c), nil
}

// PRIVATE

// Here is the gRPC server that GRPCClient talks to.
type executorGRPCServer struct {
	bonkv0.UnimplementedExecutorServiceServer

	decodeCodec *gocodec.Codec
	executors   map[string]BonkExecutor
}

func (s *executorGRPCServer) DescribeExecutors(
	ctx context.Context,
	req *bonkv0.DescribeExecutorsRequest,
) (*bonkv0.DescribeExecutorsResponse, error) {
	respBuilder := bonkv0.DescribeExecutorsResponse_builder{
		Executors: make(
			map[string]*bonkv0.DescribeExecutorsResponse_ExecutorDescription,
			len(s.executors),
		),
	}

	for name := range s.executors {
		respBuilder.Executors[name] = bonkv0.DescribeExecutorsResponse_ExecutorDescription_builder{}.Build()
	}

	return respBuilder.Build(), nil
}

func (s *executorGRPCServer) ExecuteTask(
	ctx context.Context,
	req *bonkv0.ExecuteTaskRequest,
) (*bonkv0.ExecuteTaskResponse, error) {
	executor, ok := s.executors[req.GetExecutor()]
	if !ok {
		return nil, fmt.Errorf("executor %s is not registered to this plugin", req.GetExecutor())
	}

	params := TaskParams[cue.Value]{
		Params: cue.Value{},
		Inputs: req.GetInputs(),
		OutDir: req.GetOutDirectory(),
	}

	err := os.MkdirAll(req.GetOutDirectory(), 0o750)
	if err != nil {
		return nil, fmt.Errorf("failed to create output directory: %w", err)
	}

	root, err := os.OpenRoot(req.GetOutDirectory())
	if err != nil {
		return nil, fmt.Errorf("failed to open fs root in output directory: %w", err)
	}

	err = s.decodeCodec.Validate(executor.ParamsSchema, req.GetParameters())
	if err != nil {
		return nil, fmt.Errorf(
			"params %s don't match required schema %s",
			req.GetParameters(),
			executor.ParamsSchema,
		)
	}

	params.Params, err = s.decodeCodec.Decode(req.GetParameters())
	if err != nil {
		return nil, fmt.Errorf("failed to decode parameters: %w", err)
	}

	execCtx, cleanup, err := getTaskLoggingContext(ctx, root)
	if err != nil {
		return nil, err
	}

	// Append executor information
	execCtx = slogctx.Append(execCtx, "executor", req.GetExecutor())

	outputs, err := executor.Exec(execCtx, params)
	multierr.AppendFunc(&err, cleanup)
	if err != nil {
		return nil, fmt.Errorf("failed to execute task: %w", err)
	}

	return bonkv0.ExecuteTaskResponse_builder{
		Output: outputs,
	}.Build(), nil
}
