// Copyright © 2025 Colden Cullen
// SPDX-License-Identifier: MIT

package task

import (
	"github.com/google/uuid"
	"github.com/spf13/afero"
)

// SessionID is a unique identifier per-session.
type SessionID = uuid.UUID

// NewSessionID creates a new unique session identifier which may be sorted in order of creation time.
func NewSessionID() SessionID {
	return uuid.Must(uuid.NewV7())
}

// Session defines a context in which tasks are invoked.
type Session interface {
	// ID() returns a unique identifier per-session.
	ID() SessionID
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
	Id       SessionID //nolint:revive
	SourceFs afero.Fs
	OutputFs afero.Fs
}

var (
	_ Session      = (*DefaultSession)(nil)
	_ LocalSession = (*localSession)(nil)
)

// ID returns a unique identifier per-session.
func (ds *DefaultSession) ID() SessionID {
	return ds.Id
}

// SourceFS returns an [afero.Fs] referring to the sesion root.
func (ds *DefaultSession) SourceFS() afero.Fs {
	return ds.SourceFs
}

// OutputFS returns an [afero.Fs] referring to session's output directory.
func (ds *DefaultSession) OutputFS() afero.Fs {
	return ds.OutputFs
}

type localSession struct {
	id        SessionID
	localPath string

	sourceFs afero.Fs
	outputFs afero.Fs
}

// NewLocalSession creates a session describing a project source on the current local machine.
func NewLocalSession(id SessionID, localPath string) LocalSession {
	sessionRoot := afero.NewBasePathFs(afero.NewOsFs(), localPath)

	return &localSession{
		id:        id,
		localPath: localPath,

		sourceFs: afero.NewReadOnlyFs(sessionRoot),
		outputFs: afero.NewBasePathFs(sessionRoot, ".bonk"),
	}
}

func (ls *localSession) ID() SessionID {
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
