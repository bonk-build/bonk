// Copyright © 2025 Colden Cullen
// SPDX-License-Identifier: MIT

//! `executor` provides the [Executor] trait, which is an abstraction for "object which may execute a task."
//!
//! Subpackages provide useful executors for building task-executing heirarchies.
//! Each package exports just a few objects conforming to the [Executor] interface,
//! and optionally a few helpers.

use camino::Utf8PathBuf;
use derive_more::From;
use serde::{Deserialize, Serialize};

use crate::{SessionId, Task, TaskResult};

#[derive(Clone, PartialEq, Eq, Hash, Debug, From, Serialize, Deserialize)]
#[from(forward)]
pub struct ExecutorId(Utf8PathBuf);

impl ExecutorId {
    pub fn new(parts: &[impl AsRef<str>]) -> Self {
        Self(Utf8PathBuf::from_iter(parts.iter().map(AsRef::as_ref)))
    }

    /// Split the first component and the rest.
    pub fn cut(&self) -> (&str, Option<Self>) {
        let mut iter = self.0.iter();
        let first = iter.next().unwrap();
        let rest = iter.as_path();
        if rest.as_str().is_empty() {
            (first, None)
        } else {
            (first, Some(Self(rest.to_path_buf())))
        }
    }
}

/// Executor is the trait required to execute tasks.
#[cfg_attr(test, mockall::automock)]
pub trait Executor {
    /// Execute is given a task to execute and expected to populate result with the outcome.
    fn execute(&mut self, session: &SessionId, executor: &ExecutorId, task: &Task) -> TaskResult;
}
