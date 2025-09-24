// Copyright © 2025 Colden Cullen
// SPDX-License-Identifier: MIT

package rpc

import "google.golang.org/grpc/codes"

const (
	// CodeExecErr indicates that no infrastructural errors occurred.
	// The underlying error message should be passed through unwrapped.
	CodeExecErr codes.Code = 100
)
