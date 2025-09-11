// Copyright © 2025 Colden Cullen
// SPDX-License-Identifier: MIT

package task

import (
	"github.com/google/uuid"
	"github.com/spf13/afero"
)

type testSession struct {
	memmapFs afero.MemMapFs
}

func NewTestSession() Session {
	return &testSession{
		memmapFs: afero.MemMapFs{},
	}
}

func (ts *testSession) ID() SessionId {
	return uuid.Nil
}

func (ts *testSession) SourceFS() afero.Fs {
	return &ts.memmapFs
}

func (ts *testSession) OutputFS() afero.Fs {
	return afero.NewBasePathFs(ts.SourceFS(), ".bonk-test")
}
