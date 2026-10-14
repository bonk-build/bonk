// Copyright © 2025 Colden Cullen
// SPDX-License-Identifier: MIT

//! `executor` provides the [Executor] trait, which is an abstraction for "object which may execute a task."
//!
//! Subpackages provide useful executors for building task-executing heirarchies.
//! Each package exports just a few objects conforming to the [Executor] interface,
//! and optionally a few helpers.

use std::sync::{Arc, Mutex};

use camino::Utf8PathBuf;
use derive_more::{Debug, From};
use serde::{Deserialize, Serialize};

use crate::{SessionId, Task, TaskResult};

#[derive(Clone, PartialEq, Eq, Hash, Debug, From, Serialize, Deserialize)]
#[debug("{_0:?}")]
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
pub trait Executor: std::any::Any {
    /// Execute is given a task to execute and expected to populate result with the outcome.
    fn execute(&mut self, session: &SessionId, task: Task) -> TaskResult;

    /// Helper to create a handle.
    fn into_ref(self) -> ExecutorRef<Self>
    where
        Self: Sized + 'static,
    {
        Arc::new(Mutex::new(self))
    }
}

#[cfg(test)]
mockall::mock!(
    pub Executor {}
    impl crate::Executor for Executor {
        fn execute(&mut self, session: &SessionId, task: Task) -> TaskResult;
    }
);

pub type ExecutorRef<E = dyn Executor> = Arc<Mutex<E>>;
