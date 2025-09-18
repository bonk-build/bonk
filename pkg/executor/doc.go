// Copyright © 2025 Colden Cullen
// SPDX-License-Identifier: MIT

// The executor package provides useful executors for building task-executing heirarchies.
// Each package exports just a few objects conforming to the [go.bonk.build/pkg/task.Executor] interface,
// and optionally a few helpers.
//
//   - [go.bonk.build/pkg/executor/statecheck]: allows using task states to avoid re-executing up-to-date tasks
//   - [go.bonk.build/pkg/executor/tree]: allows building a tree of named executors for complex routing
//   - [go.bonk.build/pkg/executor/rpc]: allows passing task invocations across an RPC boundary
//   - [go.bonk.build/pkg/executor/plugin]: uses [github.com/hashicorp/go-plugin] to launch gRPC-based sub-processes
package executor
