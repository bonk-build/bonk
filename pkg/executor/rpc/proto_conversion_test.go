// Copyright © 2025 Colden Cullen
// SPDX-License-Identifier: MIT

package rpc_test

import (
	"testing"

	"google.golang.org/protobuf/types/known/structpb"

	"github.com/stretchr/testify/require"

	"go.bonk.build/pkg/executor/rpc"
)

func Test_Nil(t *testing.T) {
	t.Parallel()

	proto, err := rpc.ToProtoValue(nil)
	require.NoError(t, err)
	require.IsType(t, (*structpb.Value_NullValue)(nil), proto.GetKind())
}
