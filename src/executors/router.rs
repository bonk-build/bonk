// Copyright © 2025 Colden Cullen
// SPDX-License-Identifier: MIT

#![cfg(false)]

use std::{
    collections::HashMap,
    sync::{Arc, Mutex},
};

use crate::{
    Executor, ExecutorId, Session, SessionHandler, SessionId, SessionResult, Task, TaskResult,
};

#[derive(Debug, thiserror::Error)]
pub enum RouterError {
    #[error("no executor found")]
    NoExecutorFound,
}

#[derive(Default)]
pub struct Router {
    children: HashMap<String, Box<dyn Executor>>,
    session_handlers: HashMap<String, Arc<Mutex<dyn SessionHandler>>>,
}

impl Router {
    fn new() -> Self {
        Default::default()
    }

    fn add_child(&mut self, name: ExecutorId, child: impl Executor + 'static) {
        let (first, rest) = name.cut();
        if let Some(rest) = rest {
            if self.children.contains_key(first) {
                panic!("duplicate!");
            };

            let mut child_router = Router::new();
            child_router.add_child(rest, child);
            self.children
                .insert(first.to_owned(), Arc::new(Mutex::new(child_router)));
        } else {
            self.children
                .insert(first.to_owned(), Arc::new(Mutex::new(child)));
        }
    }

    fn remove_child(&mut self, name: &ExecutorId) {
        let (first, _) = name.cut();
        self.children.remove(first);
        self.session_handlers.remove(first);
    }
}

impl Executor for Router {
    fn execute(&mut self, session: &SessionId, executor: &ExecutorId, task: &Task) -> TaskResult {
        let (child, rest) = executor.cut();
        let mut child = self.children.get_mut(child).unwrap().lock().unwrap();
        if let Some(rest) = rest {
            child.execute(session, &rest, task)
        } else {
            child.execute(session, executor, task)
        }
    }
}

impl SessionHandler for Router {
    fn open_session(&mut self, session: &Session) -> SessionResult {
        self.session_handlers
            .iter_mut()
            .try_for_each(|(_, handler)| handler.lock().unwrap().open_session(session))
    }

    fn close_session(&mut self, session_id: &SessionId) -> SessionResult {
        self.session_handlers
            .iter_mut()
            .try_for_each(|(_, handler)| handler.lock().unwrap().close_session(session_id))
    }
}

#[cfg(test)]
mod tests {
    use std::{assert_matches, ops::SubAssign};

    use mockall::predicate::eq;

    use super::*;
    use crate::TaskId;

    mockall::mock!(
        Child {}
        impl Executor for Child {
            fn execute(&mut self, session: &SessionId, executor: &ExecutorId, task: &Task) -> TaskResult;
        }
        impl SessionHandler for Child {
            fn open_session(&mut self, session: &crate::Session) -> SessionResult;
            fn close_session(&mut self, session_id: &SessionId) -> SessionResult;
        }
    );

    #[test]
    fn add() {
        let mut executors = HashMap::<ExecutorId, MockChild>::from([
            ("testing/child/abc".into(), MockChild::new()),
            ("testing/child".into(), MockChild::new()),
            ("testing/sibling".into(), MockChild::new()),
            ("unrelated".into(), MockChild::new()),
            ("super/*".into(), MockChild::new()),
        ]);

        let task_routings = Vec::<(ExecutorId, ExecutorId)>::from([
            ("testing/child/abc".into(), "testing/child/abc".into()),
            ("testing/child".into(), "testing/child".into()),
            ("testing/child/def".into(), "testing/child".into()),
            ("super/testing".into(), "super/*".into()),
        ]);

        let mut rtr = Router::new();
        let session = Session::for_testing();

        // Validate expected successful registrations
        for (id, exec) in executors.iter_mut() {
            exec.expect_open_session().once();
            // .withf(|s| s.id() == session.id())
            // .return_const(Ok(()));
            exec.expect_close_session().once().with(eq(session.id()));

            rtr.add_child(id.clone(), exec);
        }

        assert_eq!(executors.len(), rtr.session_handlers.len());

        // Validate session opening
        let res = rtr.open_session(&session);
        assert_matches!(res, Ok(_));

        // Validate task delivery
        for (sent, receive) in task_routings {
            let tsk = Task::new(TaskId::for_testing());

            let exec = executors.get_mut(&receive).unwrap();
            exec.expect_execute().once().with(
                eq(session.id()),
                eq(sent),
                mockall::predicate::always(),
            );

            let err = rtr.execute(session.id(), &sent, &tsk);
            assert_matches!(err, Ok(_));
        }

        // Validate session closing
        rtr.close_session(session.id());

        // Validate unregistration
        let num_execs = rtr.session_handlers.len();
        for (id, _) in executors.iter() {
            rtr.remove_child(id);
            num_execs.sub_assign(1);

            assert_eq!(num_execs, rtr.session_handlers.len());
        }
    }

    // #[test]
    // fn call() {
    // 	const execName: &str = "testing.child.abc";
    // 	let tsk = task.Task{
    // 		Executor: execName,
    // 	}

    // 	let exec = MockChild::new();
    // 	exec.expect_execute(session, &tsk, &result);

    // 	let rtr = Router::new();

    // 	let err = rtr.add_child(execName, exec);
    // 	assert_matches!(err, Ok(_));

    // 	err = rtr.execute(session, &tsk, &result);
    // 	assert_matches!(err, Ok(_));
    // 	assert_eq!(execName, tsk.Executor);
    // }

    // #[test]
    // fn call_wildcard() {

    // 	let exec = MockChild::new();
    // 	exec.expect_execute(session, &tsk, &result);

    // 	let rtr = Router::new();

    // 	let err = rtr.add_child("testing.*.abc", exec);
    // 	assert_matches!(err, Ok(_));

    // 	err = rtr.execute(session, &tsk, &result);
    // 	assert_matches!(err, Ok(_));
    // }

    // #[test]
    // fn call_fail() {
    // 	const execName: &str = "testing.child.abc";

    // 	let exec = MockChild::new();

    // 	let rtr = Router::new();

    // 	let err = rtr.add_child("something.else", exec);
    // 	assert_matches!(err, Ok(_));

    // 	err = rtr.execute(session, &tsk, &result);
    // 	require.Error(t, err);
    // 	require.ErrorIs(t, err, router.ErrNoExecutorFound);
    // }

    // #[test]
    // fn call_overlap() {
    // 	const execNames: [&str; 2] = [
    // 		"testing.child.abc",
    // 		"testing.sibling",
    // 	];

    // 	let rtr = Router::new();

    // 	for name in execNames {
    // 		let exec = MockChild::new();

    // 		let err = rtr.add_child(name, exec);
    // 		assert_matches!(err, Ok(_));
    // 	}

    // 	let exec = MockChild::new();
    // 	exec.expect_execute(nil, mock.Anything, (*task.Result)(nil));

    // 	let err = rtr.add_child("testing.child", exec);
    // 	assert_matches!(err, Ok(_));

    // 	err = rtr.execute(nil, &task.Task{
    // 		Executor: "testing.child",
    // 	}, nil);
    // 	assert_matches!(err, Ok(_));
    // }

    // #[test]
    // fn open_close_session_error() {
    // 	const execName: &str = "testing.child.abc";

    // 	let exec = MockChild::new();
    // 	exec.expect_open_session(session).Return(assert.AnError);
    // 	exec.expect_close_session(session.ID());

    // 	let rtr = Router::new();

    // 	let err = rtr.add_child(execName, exec);
    // 	assert_matches!(err, Ok(_));

    // 	err = rtr.open_session(session);
    // 	require.ErrorIs(t, err, assert.AnError);
    // 	defer rtr.close_session(session.ID());
    // }
}
