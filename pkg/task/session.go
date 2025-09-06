// Copyright © 2025 Colden Cullen
// SPDX-License-Identifier: MIT

package task

import (
	"context"

	"github.com/google/uuid"
	"github.com/spf13/afero"
)

// SessionId is a unique identifier per-session.
type SessionId = uuid.UUID

// A session defines a context in which tasks are invoked.
type Session interface {
	// ID() returns a unique identifier per-session.
	ID() SessionId
	// FS() returns a filesystem describing the root of the project consumed by the session.
	FS() afero.Fs
}

// LocalSession is a session that is being executed on the local machine.
type LocalSession interface {
	Session

	// LocalPath() returns the absolute path to the root of the project on disk.
	LocalPath() string
}

// DefaultSession is a default implementation of Session that stores its parameters in members.
type DefaultSession struct {
	Id SessionId
	Fs afero.Fs
}

type localSession struct {
	id        SessionId
	localPath string
}

var (
	_ Session      = (*DefaultSession)(nil)
	_ LocalSession = (*localSession)(nil)
)

func (ds *DefaultSession) ID() SessionId {
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

func (ls *localSession) ID() SessionId {
	return ls.id
}

func (ls *localSession) FS() afero.Fs {
	return afero.NewBasePathFs(afero.NewOsFs(), ls.localPath)
}

func (ls *localSession) LocalPath() string {
	return ls.localPath
}

// Executors may optionally implement this interface to be alerted when session statuses change.
type SessionManager interface {
	OpenSession(ctx context.Context, session Session) error
	CloseSession(ctx context.Context, sessionId SessionId)
}
