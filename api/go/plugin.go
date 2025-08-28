// Copyright © 2025 Colden Cullen
// SPDX-License-Identifier: MIT

package bonk // import "go.bonk.build/api/go"

import (
	"context"
	"fmt"
	"log/slog"
	"os"

	"go.uber.org/multierr"

	"google.golang.org/grpc"

	"cuelang.org/go/cue"
	"cuelang.org/go/cue/cuecontext"
	"cuelang.org/go/encoding/gocode/gocodec"

	"github.com/ValerySidorin/shclog"

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

// Call from main() to start the plugin gRPC server.
func Serve(executors ...BonkExecutor) {
	executorMap := make(map[string]BonkExecutor, len(executors))
	for _, executor := range executors {
		executorMap[executor.Name] = executor
	}

	goplugin.Serve(&goplugin.ServeConfig{
		HandshakeConfig: Handshake,
		Plugins: map[string]goplugin.Plugin{
			PluginType: &BonkPluginServer{
				Executors: executorMap,
			},
		},
		GRPCServer: goplugin.DefaultGRPCServer,
		Logger:     shclog.New(slog.Default()),
	})
}

var Handshake = goplugin.HandshakeConfig{
	ProtocolVersion:  0,
	MagicCookieKey:   "BONK_PLUGIN",
	MagicCookieValue: "executor",
}

const PluginType = "bonk"

type BonkPluginServer struct {
	goplugin.NetRPCUnsupportedPlugin
	goplugin.GRPCPlugin

	Executors map[string]BonkExecutor
}

func (p *BonkPluginServer) GRPCServer(_ *goplugin.GRPCBroker, s *grpc.Server) error {
	bonkv0.RegisterBonkPluginServiceServer(s, &grpcServer{
		decodeCodec: gocodec.New(cuectx, &gocodec.Config{}),
		executors:   p.Executors,
	})

	return nil
}

func (p *BonkPluginServer) GRPCClient(
	_ context.Context,
	_ *goplugin.GRPCBroker,
	c *grpc.ClientConn,
) (any, error) {
	return bonkv0.NewBonkPluginServiceClient(c), nil
}

// PRIVATE

// Here is the gRPC server that GRPCClient talks to.
type grpcServer struct {
	bonkv0.UnimplementedBonkPluginServiceServer

	decodeCodec *gocodec.Codec
	executors   map[string]BonkExecutor
}

func (s *grpcServer) ConfigurePlugin(
	ctx context.Context,
	req *bonkv0.ConfigurePluginRequest,
) (*bonkv0.ConfigurePluginResponse, error) {
	respBuilder := bonkv0.ConfigurePluginResponse_builder{
		Features: []bonkv0.ConfigurePluginResponse_FeatureFlags{
			bonkv0.ConfigurePluginResponse_FEATURE_FLAGS_STREAMING_LOGGING,
		},
		Executors: make(
			map[string]*bonkv0.ConfigurePluginResponse_ExecutorDescription,
			len(s.executors),
		),
	}

	for name := range s.executors {
		respBuilder.Executors[name] = bonkv0.ConfigurePluginResponse_ExecutorDescription_builder{}.Build()
	}

	return respBuilder.Build(), nil
}

func (s *grpcServer) PerformTask(
	ctx context.Context,
	req *bonkv0.PerformTaskRequest,
) (*bonkv0.PerformTaskResponse, error) {
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

	return bonkv0.PerformTaskResponse_builder{
		Output: outputs,
	}.Build(), nil
}
