// Copyright © 2025 Colden Cullen
// SPDX-License-Identifier: MIT

use derive_more::Debug;
use lazy_errors::prelude::Error;

use crate::{Executor, ExecutorId, SessionId, TaskId, TaskResult};

/// Describes the new status of a task.
#[derive(Debug)]
pub enum TaskStatus<'msg> {
    /// Running means that the task has begun executing.
    Running,
    /// Success means the task has completed successfully.
    Success,
    /// Error means the task has returned an error.
    Error(&'msg Error),
}

/// TaskStatusMsg signifies a task's change in status.
pub struct TaskStatusMsg<'msg> {
    /// The task that this event is referring to.
    pub task_id: &'msg TaskId,
    /// The new status for the task.
    pub status: TaskStatus<'msg>,
}

pub type Observer = Box<dyn FnMut(&TaskStatusMsg)>;

/// Executor which forwards events to a child and emits events about status changes.
pub struct Observable<'child> {
    child: &'child mut dyn Executor,

    listeners: Vec<Observer>,
}

impl<'child> Observable<'child> {
    /// Create a new Observable exector.
    pub fn new(child: &'child mut dyn Executor) -> Self {
        Self {
            child,
            listeners: Default::default(),
        }
    }

    /// Adds a listener to be called when a task's status changes.
    pub fn add_listener(&mut self, listener: Observer) {
        self.listeners.push(listener);
    }

    /// Sends `msg` to all listeners.
    fn broadcast(&mut self, msg: &TaskStatusMsg) {
        self.listeners.iter_mut().for_each(|listener| listener(msg));
    }
}

impl Executor for Observable<'_> {
    fn execute(
        &mut self,
        session: &SessionId,
        executor: &ExecutorId,
        task: &crate::Task,
    ) -> TaskResult {
        self.broadcast(&TaskStatusMsg {
            task_id: task.id(),
            status: TaskStatus::Running,
        });

        let result = self.child.execute(session, executor, task);

        match result {
            Ok(_) => self.broadcast(&TaskStatusMsg {
                task_id: task.id(),
                status: TaskStatus::Success,
            }),
            Err(ref err) => self.broadcast(&TaskStatusMsg {
                task_id: task.id(),
                status: TaskStatus::Error(err),
            }),
        }

        result
    }
}

#[cfg(test)]
mod tests {
    use std::{assert_matches, ops::AddAssign, sync::Arc};

    use mockall::predicate::eq;

    use super::*;
    use crate::{MockExecutor, Session, Task, TaskOutput};

    #[test]
    fn pass() {
        let task_id = TaskId::for_testing();
        let exec_id = ExecutorId::from("exec");
        let session = Session::for_testing();

        let mut exec = MockExecutor::new();

        exec.expect_execute()
            .once()
            .with(
                eq(session.id().clone()),
                eq(exec_id.clone()),
                crate::has_id(task_id.clone()),
            )
            .return_once(|_, _, _| Ok(TaskOutput::default()));

        let mut obs = Observable::new(&mut exec);

        let mut call_count = 0;
        let listener_task_id = task_id.clone();
        obs.add_listener(Box::new(move |status| {
            call_count.add_assign(1);
            assert_eq!(*status.task_id, listener_task_id);
            match call_count {
                1 => assert_matches!(status.status, TaskStatus::Running),
                2 => assert_matches!(status.status, TaskStatus::Success),
                _ => unreachable!(),
            }
        }));

        let res = obs.execute(
            session.id(),
            &exec_id,
            &Task::builder(task_id.clone()).build(),
        );
        assert_matches!(res, Ok(_));
    }

    #[test]
    fn fail() {
        let task_id = TaskId::for_testing();
        let exec_id = ExecutorId::from("exec");
        let session = Session::for_testing();

        let mut exec = MockExecutor::new();

        exec.expect_execute()
            .once()
            .with(
                eq(session.id().clone()),
                eq(exec_id.clone()),
                crate::has_id(task_id.clone()),
            )
            .return_once(|_, _, _| Err(lazy_errors::err!("test error")));

        let mut obs = Observable::new(&mut exec);

        let mut call_count = 0;
        let listener_task_id = task_id.clone();
        obs.add_listener(Box::new(move |status| {
            call_count.add_assign(1);
            assert_eq!(*status.task_id, listener_task_id);
            match call_count {
                1 => assert_matches!(status.status, TaskStatus::Running),
                2 => assert_matches!(status.status, TaskStatus::Error(_)),
                _ => unreachable!(),
            }
        }));

        let res = obs.execute(
            session.id(),
            &exec_id,
            &Task::builder(task_id.clone()).build(),
        );
        assert_matches!(res, Err(_));
    }
}
