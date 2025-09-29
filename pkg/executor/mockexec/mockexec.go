// Copyright © 2025 Colden Cullen
// SPDX-License-Identifier: MIT

package mockexec

import (
	"testing"

	"go.uber.org/mock/gomock"
)

//go:generate go tool mockgen -destination executor_mock.go -package mockexec -copyright_file ../../../license-header.txt -typed ../../task Executor

func New(t *testing.T) *MockExecutor {
	t.Helper()
	ctrl := gomock.NewController(t)

	return NewMockExecutor(ctrl)
}
