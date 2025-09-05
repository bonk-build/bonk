// Copyright © 2025 Colden Cullen
// SPDX-License-Identifier: MIT

package task

import (
	"github.com/google/uuid"
	"github.com/spf13/afero"
)

type Session interface {
	ID() uuid.UUID
	FS() afero.Fs
}

type LocalSession interface {
	Session

	LocalPath() string
}

type DefaultSession struct {
	Id uuid.UUID
	Fs afero.Fs
}

type localSession struct {
	id        uuid.UUID
	localPath string
}

var (
	_ Session      = (*DefaultSession)(nil)
	_ LocalSession = (*localSession)(nil)
)

func (ds *DefaultSession) ID() uuid.UUID {
	return ds.Id
}

func (ds *DefaultSession) FS() afero.Fs {
	return ds.Fs
}

func NewLocalSession(localPath string) LocalSession {
	return &localSession{
		id:        uuid.Must(uuid.NewV7()),
		localPath: localPath,
	}
}

func (ls *localSession) ID() uuid.UUID {
	return ls.id
}

func (ls *localSession) FS() afero.Fs {
	return afero.NewBasePathFs(afero.NewOsFs(), ls.localPath)
}

func (ls *localSession) LocalPath() string {
	return ls.localPath
}
