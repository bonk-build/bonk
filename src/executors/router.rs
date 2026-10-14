// Copyright © 2025 Colden Cullen
// SPDX-License-Identifier: MIT

use std::{any::Any, collections::HashMap};

use lazy_errors::OrWrap;

use crate::{Executor, ExecutorId, ExecutorRef, SessionId, Task, TaskResult};

/// Signifies that this path component is meant to act as a fallback.
pub const WILDCARD: &str = "*";

#[derive(Debug, thiserror::Error)]
pub enum RouterError {
    #[error("no executor found")]
    NoExecutorFound,
}

#[derive(Default)]
pub struct Router {
    children: HashMap<String, ExecutorRef>,
}

impl Router {
    fn new() -> Self {
        Default::default()
    }

    /// Adds a child executor to the router.
    ///
    /// If `name` consists of multiple components, they will be split and inserted as a heirarchy of routers.
    ///
    /// If an existing executor is overwritten, it is returned.
    fn add_child(&mut self, name: ExecutorId, child: ExecutorRef) -> Option<ExecutorRef> {
        let (first, rest) = name.cut();
        if let Some(rest) = rest {
            // In this case, the child at `first` must be a router.

            if let Some(existing_child_ref) = self.children.get(first) {
                if let Some(existing_child) =
                    <dyn Any>::downcast_mut::<Router>(&mut *existing_child_ref.lock().unwrap())
                {
                    existing_child.add_child(rest, child)
                } else {
                    // If the child exists but isn't a router, replace it.
                    self.children.insert(first.to_owned(), child)
                }
            } else {
                // In this case, the child doesn't exist but more are coming, so make a new router.

                let mut child_router = Router::new();
                child_router.add_child(rest, child);
                self.children
                    .insert(first.to_owned(), child_router.into_ref())
            }
        } else {
            // Handle the "child doesn't exist" case.
            self.children.insert(first.to_owned(), child)
        }
    }

    fn remove_child(&mut self, name: &ExecutorId) {
        let (first, rest) = name.cut();

        let remove = if let Some(child) = self.children.get_mut(first) {
            if let Some(child) = <dyn Any>::downcast_mut::<Router>(&mut *child.lock().unwrap())
                && let Some(rest) = rest
            {
                child.remove_child(&rest);
                child.children.len() == 0
            } else {
                true
            }
        } else {
            false
        };

        if remove {
            self.children.remove(first);
        }
    }
}

impl Executor for Router {
    fn execute(&mut self, session: &SessionId, mut task: Task) -> TaskResult {
        let (child, rest) = task.executor().cut();
        let child = if let Some(child) = self.children.get_mut(child) {
            child
        } else if let Some(child) = self.children.get_mut(WILDCARD) {
            child
        } else {
            return Err(RouterError::NoExecutorFound).or_wrap()?;
        };

        if let Some(rest) = rest {
            task = task.with_executor(rest);
        }

        child.lock().unwrap().execute(session, task)
    }
}

#[cfg(test)]
mod tests {
    use std::assert_matches;

    use super::*;
    use crate::{ExecutorRef, Session, SessionHandler, SessionId, SessionResult, TaskId};

    mockall::mock!(
        Child {}
        impl Executor for Child {
            fn execute(&mut self, session: &SessionId, task: Task) -> TaskResult;
        }
        impl SessionHandler for Child {
            fn open_session(&mut self, session: &Session) -> SessionResult;
            fn close_session(&mut self, session_id: &SessionId) -> SessionResult;
        }
    );

    #[test]
    fn add_replace() {
        let mut rtr = Router::new();

        let existing = rtr.add_child("testing/*".into(), MockChild::new().into_ref());
        assert!(existing.is_none());

        let existing = rtr.add_child("testing".into(), MockChild::new().into_ref());
        assert!(existing.is_some());
        assert!(<dyn Any>::is::<Router>(&*existing.unwrap().lock().unwrap()))
    }

    #[test]
    fn call() {
        let session = SessionId::new();
        let exec_id = ExecutorId::from("testing/child/abc");
        let task = Task::new(TaskId::for_testing(), exec_id.clone());

        let mut exec = MockChild::new();
        exec.expect_execute()
            .once()
            .return_once_st(|_, _| Ok(Default::default()));

        let mut router = Router::new();

        let existing = router.add_child(exec_id, exec.into_ref());
        assert!(existing.is_none());

        let res = router.execute(&session, task);
        assert_matches!(res, Ok(_));
    }

    #[test]
    fn call_wildcard() {
        let mut exec = MockChild::new();
        exec.expect_execute()
            .once()
            .return_once_st(|_, _| Ok(Default::default()));

        let mut router = Router::new();

        let existing = router.add_child(ExecutorId::from("testing/*/abc"), exec.into_ref());
        assert!(existing.is_none());

        let session = SessionId::new();
        let task = Task::new(TaskId::for_testing(), ExecutorId::from("testing/child/abc"));
        let res = router.execute(&session, task);
        assert_matches!(res, Ok(_));
    }

    #[test]
    fn call_fail() {
        let session = SessionId::new();
        let task = Task::new(TaskId::for_testing(), ExecutorId::from("testing/child/abc"));

        let mut router = Router::new();

        let res = router.execute(&session, task);
        assert_matches!(res, Err(_));
        let err = res.unwrap_err();
        assert_eq!(1, err.children().len());
        let err = &err.children()[0];
        assert!(err.is::<RouterError>());
        let err = err.downcast_ref::<RouterError>().unwrap();
        assert_matches!(err, RouterError::NoExecutorFound);
    }

    #[test]
    fn routing() {
        let executors = Vec::<(ExecutorId, ExecutorRef<MockChild>)>::from([
            ("testing/*".into(), MockChild::new().into_ref()),
            ("testing/child/abc".into(), MockChild::new().into_ref()),
            ("testing/child/*".into(), MockChild::new().into_ref()),
            ("testing/sibling".into(), MockChild::new().into_ref()),
            ("unrelated".into(), MockChild::new().into_ref()),
            ("super/*".into(), MockChild::new().into_ref()),
            ("*/*".into(), MockChild::new().into_ref()),
        ]);

        let task_routings = Vec::<(ExecutorId, ExecutorId)>::from([
            ("testing/child/abc".into(), "testing/child/abc".into()),
            ("testing/child/def".into(), "testing/child/*".into()),
            // Test sending to the single wildcard
            ("super/testing".into(), "super/*".into()),
            // Test sending to the double-wildcard
            ("other/testing".into(), "*/*".into()),
        ]);

        let mut rtr = Router::new();
        let session = Session::for_testing();

        // Validate expected successful registrations
        for (id, exec) in executors.iter() {
            let existing = rtr.add_child(id.clone(), exec.clone());
            assert!(existing.is_none());
        }

        assert_eq!(4, rtr.children.len());

        let mut executors: HashMap<ExecutorId, ExecutorRef<MockChild>> =
            executors.into_iter().collect();

        // Validate task delivery
        for (sent, receive) in task_routings {
            eprintln!("sending message from {sent:?} to {receive:?}");

            let task = Task::new(TaskId::for_testing(), sent);

            let exec: &mut ExecutorRef<MockChild> = executors.get_mut(&receive).unwrap();
            exec.lock()
                .unwrap()
                .expect_execute()
                .once()
                .return_once_st(|_, _| Ok(Default::default()));

            let res = rtr.execute(session.id(), task);
            assert_matches!(res, Ok(_), "failed to find correct handler");
        }

        // Validate unregistration
        for (id, _) in executors.iter() {
            rtr.remove_child(id);
        }
        assert_eq!(0, rtr.children.len());
    }
}
