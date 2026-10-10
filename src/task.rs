// Copyright © 2025 Colden Cullen
// SPDX-License-Identifier: MIT

//! `task` contains the core structures needed define a bonk pipeline.
//! For more information, see [Task], and [Executor].

use std::hash::Hash;

use camino::Utf8PathBuf;
use derive_more::{Debug, Display, From};
use serde::{Deserialize, Serialize};

use crate::Argument;

#[derive(PartialEq, Eq, Hash, From, Debug, Display, Clone, Serialize, Deserialize)]
#[debug("{_0:?}")]
#[display("{_0}")]
#[from(forward)]
pub struct TaskId(pub(crate) Utf8PathBuf);

impl TaskId {
    pub fn new(parts: &[impl AsRef<str>]) -> Self {
        Self(Utf8PathBuf::from_iter(parts.iter().map(AsRef::as_ref)))
    }

    pub fn into_child(self, parts: &[&str]) -> Self {
        Self(parts.iter().fold(self.0, |result, part| result.join(part)))
    }

    pub fn make_child(&self, parts: &[&str]) -> Self {
        self.clone().into_child(parts)
    }
}

#[cfg(test)]
impl TaskId {
    /// Helper for creating a TaskId for use in test cases where the id doesn't matter.
    pub fn for_testing() -> Self {
        use std::sync::atomic::AtomicUsize;

        static TESTING_COUNTER: AtomicUsize = AtomicUsize::new(0);
        Self::new(&[
            "testing",
            TESTING_COUNTER
                .fetch_add(1, std::sync::atomic::Ordering::AcqRel)
                .to_string()
                .as_str(),
        ])
    }
}

// Task represents a unit of work to be executed.
#[non_exhaustive]
#[derive(PartialEq, Eq, Debug, Hash, bon::Builder, Serialize, Deserialize)]
#[builder(on(Vec<_>, into))]
#[debug("{id:?}")]
pub struct Task {
    /// Describes how this task is addressed.
    #[builder(start_fn)]
    id: TaskId,

    /// Describes any files that may be consumed by this task (relative to [Session.SourceFS]).
    #[builder(default)]
    inputs: Vec<String>,
    /// A list of tasks which must be completed before this task can run.
    #[builder(default)]
    dependencies: Vec<TaskId>,
    /// Any arguments that may be passed to the executor.
    #[serde(skip)]
    args: Option<Box<dyn Argument>>,
}

impl Task {
    #[inline]
    pub fn new(id: TaskId) -> Self {
        Self::builder(id).build()
    }

    pub fn id(&self) -> &TaskId {
        &self.id
    }

    pub fn inputs(&self) -> &[String] {
        self.inputs.as_slice()
    }

    pub fn dependencies(&self) -> &[TaskId] {
        self.dependencies.as_slice()
    }

    pub fn args(&self) -> Option<&dyn Argument> {
        self.args.as_deref()
    }
}

#[derive(PartialEq, Debug, Default, bon::Builder, Serialize, Deserialize)]
pub struct TaskOutput {
    /// outputs describes any files that have been emitted by the task relative to [Session.OutputFS].
    #[builder(default, into)]
    outputs: Vec<String>,

    /// followupTasks is a list of tasks to be executed after this task completes.
    #[builder(default)]
    followup_tasks: Vec<Task>,
}

impl TaskOutput {
    pub fn outputs(&self) -> &[String] {
        &self.outputs
    }

    pub fn followup_tasks(&self) -> &[Task] {
        &self.followup_tasks
    }
}

pub type TaskResult = lazy_errors::Result<TaskOutput>;

#[cfg(test)]
pub fn has_id(id: TaskId) -> impl mockall::Predicate<Task> {
    return mockall::predicate::function(move |task: &Task| task.id() == &id);
}

#[cfg(test)]
mod tests {
    use super::*;
    #[test]
    fn new() {
        let task = Task::builder(TaskId::new(&["root", "child"])).build();

        assert_eq!(task.id, TaskId::new(&["root/child"]));
        assert!(task.inputs.is_empty());
        assert!(task.dependencies.is_empty());
        assert!(task.args.is_none());
    }

    #[test]
    fn new_with_inputs() {
        let task = Task::builder(TaskId::new(&["root", "child"]))
            .inputs(["InputA".to_owned()])
            .build();

        assert_eq!(task.id, TaskId::new(&["root/child"]));
        assert_eq!(task.inputs.len(), 1);
        assert_eq!(task.inputs[0], "InputA");
        assert!(task.dependencies.is_empty());
        assert!(task.args.is_none());
    }

    #[test]
    fn new_with_dependencies() {
        let task = Task::builder(TaskId::new(&["root", "child"]))
            .dependencies(&[TaskId::new(&["root", "sibling"])])
            .build();

        assert_eq!(task.id, TaskId::new(&["root/child"]));
        assert!(task.inputs.is_empty());
        assert_eq!(task.dependencies.len(), 1);
        assert_eq!(task.dependencies[0], TaskId::new(&["root/sibling"]));
        assert!(task.args.is_none());
    }

    #[test]
    fn new_with_args() {
        let task = Task::builder(TaskId::new(&["root", "child"]))
            .args(Box::new(5i8))
            .build();

        assert_eq!(task.id, TaskId::new(&["root/child"]));
        assert!(task.inputs.is_empty());
        assert!(task.dependencies.is_empty());
        assert_eq!(*task.args.unwrap().downcast_ref::<i8>().unwrap(), 5);
    }
}
