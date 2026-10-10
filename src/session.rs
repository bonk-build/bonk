// Copyright © 2025 Colden Cullen
// SPDX-License-Identifier: MIT

use derive_more::{Debug, Display, PartialEq};
use serde::{Deserialize, Serialize};
use vfs::VfsPath;

use crate::TaskId;

#[derive(PartialEq, Eq, Hash, Debug, Display, Clone, Copy, Serialize, Deserialize)]
#[debug("{_0:?}")]
#[display("{_0}")]
pub struct SessionId(uuid::Uuid);

impl SessionId {
    pub fn new() -> Self {
        Self(uuid::Uuid::now_v7())
    }
}

impl Default for SessionId {
    fn default() -> Self {
        Self::new()
    }
}

#[derive(PartialEq, Eq, Debug, Clone)]
#[non_exhaustive]
pub struct Session {
    id: SessionId,

    #[partial_eq(skip)]
    source_fs: VfsPath,

    #[partial_eq(skip)]
    output_fs: VfsPath,
}

impl Session {
    pub fn new(id: SessionId, source_dir: impl AsRef<std::path::Path>) -> Self {
        let source_fs = VfsPath::new(vfs::PhysicalFS::new(source_dir));
        let output_fs = source_fs.join(".bonk").unwrap();
        Self {
            id,
            source_fs,
            output_fs,
        }
    }

    pub fn id(&self) -> &SessionId {
        &self.id
    }

    pub fn input_fs(&self) -> &VfsPath {
        &self.source_fs
    }

    pub fn output_fs(&self, task: &TaskId) -> VfsPath {
        self.output_fs.join(task.0.as_str()).unwrap()
    }

    #[cfg(test)]
    pub fn for_testing() -> Self {
        Self {
            id: SessionId::new(),
            source_fs: VfsPath::new(vfs::MemoryFS::new()),
            output_fs: VfsPath::new(vfs::MemoryFS::new()),
        }
    }
}

impl std::hash::Hash for Session {
    fn hash<H: std::hash::Hasher>(&self, state: &mut H) {
        self.id.hash(state);
    }
}

impl std::borrow::Borrow<SessionId> for Session {
    fn borrow(&self) -> &SessionId {
        &self.id
    }
}

pub type SessionResult = lazy_errors::Result<()>;

#[cfg_attr(test, mockall::automock)]
pub trait SessionHandler {
    /// OpenSession is called before any tasks are executed, and can be used to do things such as
    /// initializing caches, etc.
    fn open_session(&mut self, session: &Session) -> SessionResult;

    /// CloseSession shuts down and frees all resources created over the course of a session.
    /// After this call, no outstanding goroutines should be running.
    fn close_session(&mut self, session_id: &SessionId) -> SessionResult;
}
