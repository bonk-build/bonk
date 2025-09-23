// Copyright © 2025 Colden Cullen
// SPDX-License-Identifier: MIT

package task

import (
	"github.com/google/uuid"
	"github.com/spf13/afero"
)

// SessionId is a unique identifier per-session.
type SessionId = uuid.UUID

// NewSessionId creates a new unique session identifier which may be sorted in order of creation time.
func NewSessionId() SessionId {
	return uuid.Must(uuid.NewV7())
}

// Session defines a context in which tasks are invoked.
type Session interface {
	// ID() returns a unique identifier per-session.
	ID() SessionId
	// SourceFS() returns a filesystem describing the root of the project consumed by the session.
	SourceFS() afero.Fs
	// OutputFS() returns a filesystem where output files may be written.
	OutputFS() afero.Fs
}

// LocalSession is a session that is being executed on the local machine.
type LocalSession interface {
	Session

	// LocalPath() returns the absolute path to the root of the project on disk.
	LocalPath() string
}

// DefaultSession is a default implementation of Session that stores its parameters in members.
type DefaultSession struct {
	Id       SessionId
	SourceFs afero.Fs
	OutputFs afero.Fs
}

var (
	_ Session      = (*DefaultSession)(nil)
	_ LocalSession = (*localSession)(nil)
)

func (ds *DefaultSession) ID() SessionId {
	return ds.Id
}

func (ds *DefaultSession) SourceFS() afero.Fs {
	return ds.SourceFs
}

// OutputFS implements Session.
func (ds *DefaultSession) OutputFS() afero.Fs {
	return ds.OutputFs
}

type localSession struct {
	id        SessionId
	localPath string

	sourceFs afero.Fs
	outputFs afero.Fs
}

// NewLocalSession creates a session describing a project source on the current local machine.
func NewLocalSession(id SessionId, localPath string) LocalSession {
	sessionRoot := afero.NewBasePathFs(afero.NewOsFs(), localPath)

	return &localSession{
		id:        id,
		localPath: localPath,

		sourceFs: afero.NewReadOnlyFs(sessionRoot),
		outputFs: afero.NewBasePathFs(sessionRoot, ".bonk"),
	}
}

func (ls *localSession) ID() SessionId {
	return ls.id
}

func (ls *localSession) SourceFS() afero.Fs {
	return ls.sourceFs
}

func (ls *localSession) OutputFS() afero.Fs {
	return ls.outputFs
}

func (ls *localSession) LocalPath() string {
	return ls.localPath
}
