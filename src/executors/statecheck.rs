// Copyright © 2025 Colden Cullen
// SPDX-License-Identifier: MIT

use std::{
    collections::HashSet,
    hash::Hasher,
};

use fnv::FnvHasher;
use lazy_errors::{ErrorStash, OrStash, OrWrap};
use serde::{Deserialize, Serialize};
use vfs::VfsPath;

use crate::{
    Executor, ExecutorId, Session, SessionHandler, SessionId, SessionResult, Task, TaskOutput,
    TaskResult,
};

/// An executor that avoids re-running tasks if they are already up to date.
pub struct StateChecker<'child> {
    child: &'child mut dyn Executor,

    sessions: HashSet<Session>,
}

impl<'child> StateChecker<'child> {
    fn new(child: &'child mut dyn Executor) -> Self {
        Self {
            child,
            sessions: Default::default(),
        }
    }
}

impl Executor for StateChecker<'_> {
    fn execute(&mut self, session: &SessionId, executor: &ExecutorId, task: &Task) -> TaskResult {
        let session = self
            .sessions
            .get(session)
            .ok_or(lazy_errors::err!("session not opened"))?;
        let cached_output = TaskState::load_and_check(session, executor, task);

        if let Ok(cached_output) = cached_output {
            Ok(cached_output)
        } else {
            self.child
                .execute(session.id(), executor, task)
                .and_then(|output| {
                    TaskState::save(session, executor, task, &output).map(|()| output)
                })
        }
    }
}

impl SessionHandler for StateChecker<'_> {
    fn open_session(&mut self, session: &Session) -> SessionResult {
        self.sessions.insert(session.clone());

        Ok(())
    }

    fn close_session(&mut self, session_id: &SessionId) -> SessionResult {
        self.sessions.remove(session_id);

        Ok(())
    }
}

#[derive(Debug, thiserror::Error)]
enum TaskStateMismatch {
    #[error("saved state not found")]
    StateMissing,

    #[error("different executor requested")]
    Executor,
    #[error("different arguments passed")]
    Argumnets,
    #[error("different inputs passed")]
    Inputs,
    #[error("outputs changed or are missing ondisk")]
    Outputs,
}

const STATE_FILE: &str = "state.json";

#[derive(Serialize, Deserialize)]
struct TaskState {
    executor: ExecutorId,
    // This needs to serialize as a Value to avoid cloning the object before serializing
    output: serde_json::Value,

    arguments_checksum: u64,
    inputs_checksum: u64,
    output_checksum: u64,
}

impl TaskState {
    fn save(
        session: &Session,
        executor: &ExecutorId,
        task: &Task,
        output: &TaskOutput,
    ) -> lazy_errors::Result<()> {
        let output_fs = session.output_fs(task.id());
        output_fs.create_dir_all().unwrap();

        // @TODO: needs hashable argument
        let arguments_checksum = 0;

        let inputs_checksum = hash_files(session.input_fs(), task.inputs())
            .ok_or(lazy_errors::err!("failed to hash input files"))?;
        let output_checksum = hash_files(&output_fs, output.outputs())
            .ok_or(lazy_errors::err!("failed to hash output files"))?;

        let state = Self {
            executor: executor.clone(),
            output: serde_json::to_value(output).or_wrap()?,

            arguments_checksum,
            inputs_checksum,
            output_checksum,
        };

        // Save the file.
        let output = output_fs
            .join(STATE_FILE)
            .or_wrap()?
            .create_file()
            .or_wrap()?;
        serde_json::to_writer(output, &state).or_wrap()?;

        Ok(())
    }

    fn load_and_check(
        session: &Session,
        executor: &ExecutorId,
        task: &Task,
    ) -> lazy_errors::Result<TaskOutput> {
        let output_fs = session.output_fs(task.id());

        let saved = output_fs
            .join(STATE_FILE)
            .unwrap()
            .open_file()
            .map_err(|_| TaskStateMismatch::StateMissing)
            .or_wrap()?;

        let saved: Self = serde_json::from_reader(saved).or_wrap()?;
        let saved_value: TaskOutput = serde_json::from_value(saved.output).or_wrap()?;

        let mut mismatches = ErrorStash::new(|| "cached task state doesn't match incoming state");

        executor
            .eq(&saved.executor)
            .ok_or(TaskStateMismatch::Executor)
            .or_stash(&mut mismatches);

        // @TODO: args
        0.eq(&saved.arguments_checksum)
            .ok_or(TaskStateMismatch::Argumnets)
            .or_stash(&mut mismatches);

        hash_files(session.input_fs(), task.inputs())
            .is_some_and(|cs| cs == saved.inputs_checksum)
            .ok_or(TaskStateMismatch::Inputs)
            .or_stash(&mut mismatches);

        hash_files(&output_fs, saved_value.outputs())
            .is_some_and(|cs| cs == saved.output_checksum)
            .ok_or(TaskStateMismatch::Outputs)
            .or_stash(&mut mismatches);

        mismatches.into_result().map(|()| saved_value)
    }
}

#[must_use]
fn hash_files(fs: &VfsPath, files: &[impl AsRef<str>]) -> Option<u64> {
    let mut hasher = IoWrapper(FnvHasher::default());

    // @TODO: rayon parallel?
    for file in files {
        let file = fs.join(file).ok()?;
        let mut file = file.open_file().ok()?;
        std::io::copy(&mut file, &mut hasher).ok()?;
    }

    Some(hasher.0.finish())
}

struct IoWrapper<T>(pub T);

impl<T: Hasher> std::io::Write for IoWrapper<T> {
    #[inline]
    fn write(&mut self, buf: &[u8]) -> std::io::Result<usize> {
        Hasher::write(&mut self.0, buf);
        Ok(buf.len())
    }

    #[inline]
    fn flush(&mut self) -> std::io::Result<()> {
        Ok(())
    }
}

#[cfg(test)]
mod tests {
    use super::*;
    use crate::{MockExecutor, TaskId};
    use std::assert_matches;

    #[test]
    fn exec_save_state() {
        let mut exec = MockExecutor::new();
        exec.expect_execute()
            .once()
            .return_once_st(|_, _, _| Ok(TaskOutput::default()));

        let mut checker = StateChecker::new(&mut exec);
        let session = Session::for_testing();
        let executor = ExecutorId::from("exec");
        let task = Task::new(TaskId::for_testing());

        checker.open_session(&session).unwrap();

        let res1 = checker.execute(session.id(), &executor, &task);
        assert_matches!(res1, Ok(_));

        assert!(
            session
                .output_fs(task.id())
                .join(STATE_FILE)
                .unwrap()
                .is_file()
                .unwrap()
        );

        let res2 = checker.execute(session.id(), &executor, &task);
        assert_matches!(res2, Ok(_));
        assert_eq!(res1.unwrap(), res2.unwrap());

        checker.close_session(session.id()).unwrap();
    }

    #[test]
    fn exec_failure() {
        let mut exec = MockExecutor::new();
        exec.expect_execute()
            .once()
            .return_once_st(|_, _, _| Err(lazy_errors::err!("test error")));

        let mut checker = StateChecker::new(&mut exec);
        let session = Session::for_testing();
        let task = Task::new(TaskId::for_testing());

        checker.open_session(&session).unwrap();

        let res1 = checker.execute(session.id(), &ExecutorId::from("exec"), &task);
        assert_matches!(res1, Err(_));

        assert!(
            !session
                .output_fs(task.id())
                .join(STATE_FILE)
                .unwrap()
                .exists()
                .unwrap()
        );
    }

    #[test]
    fn exec_state_mismatches() {
        let mut exec = MockExecutor::new();
        exec.expect_execute()
            .times(2)
            .returning_st(|_, _, _| Ok(TaskOutput::default()));

        let mut checker = StateChecker::new(&mut exec);
        let session = Session::for_testing();
        let task = Task::builder(TaskId::for_testing())
            .args(Box::new(12))
            .build();

        checker.open_session(&session).unwrap();

        let res1 = checker.execute(session.id(), &ExecutorId::from("exec/1"), &task);
        assert_matches!(res1, Ok(_));

        assert!(
            session
                .output_fs(task.id())
                .join(STATE_FILE)
                .unwrap()
                .is_file()
                .unwrap()
        );

        let res2 = checker.execute(session.id(), &ExecutorId::from("exec/2"), &task);
        assert_matches!(res2, Ok(_));
        assert_eq!(res1.unwrap(), res2.unwrap());

        checker.close_session(session.id()).unwrap();
    }

    #[test]
    fn state_save() {
        let session = Session::for_testing();
        let exec = ExecutorId::from("exec");
        let task = Task::new(TaskId::for_testing());
        let output = TaskOutput::default();

        let res = TaskState::save(&session, &exec, &task, &output);
        assert_matches!(res, Ok(()));

        assert!(
            session
                .output_fs(task.id())
                .join(STATE_FILE)
                .unwrap()
                .is_file()
                .unwrap()
        );
    }

    // @TODO: args
    const INPUT_FILE_NAME: &str = "input-file";

    #[test]
    fn state_mismatches_inputs() {
        let session = Session::for_testing();
        let exec = ExecutorId::from("exec");
        let task = Task::new(TaskId::for_testing());
        let output = TaskOutput::default();

        let res = TaskState::save(&session, &exec, &task, &output);
        assert_matches!(res, Ok(()));

        let res = TaskState::load_and_check(&session, &exec, &task);
        assert_matches!(res, Ok(_));

        let task = Task::builder(task.id().clone())
            .inputs([INPUT_FILE_NAME.into()])
            .build();

        let res = TaskState::load_and_check(&session, &exec, &task);
        assert_matches!(res, Err(_));
        let unwrap_err = res.unwrap_err();
        let mismatches = unwrap_err.children();
        assert_eq!(mismatches.len(), 1);
        assert_matches!(
            mismatches[0].downcast_ref::<TaskStateMismatch>().unwrap(),
            TaskStateMismatch::Inputs
        );
    }

    #[test]
    fn state_mismatches_inputs_checksum() {
        const INPUT_FILE_CONTENTS1: &str = "This if the first iteration of the file";
        const INPUT_FILE_CONTENTS2: &str = "This if the second iteration of the file";
        let session = Session::for_testing();
        let exec = ExecutorId::from("exec");
        let task = Task::builder(TaskId::for_testing())
            .inputs([INPUT_FILE_NAME.into()])
            .build();
        let output = TaskOutput::default();

        let input_file_path = session.input_fs().join(INPUT_FILE_NAME).unwrap();

        let mut input_file = input_file_path.create_file().unwrap();
        input_file
            .write_all(INPUT_FILE_CONTENTS1.as_bytes())
            .unwrap();
        input_file.flush().unwrap();

        let res = TaskState::save(&session, &exec, &task, &output);
        assert_matches!(res, Ok(()));

        let res = TaskState::load_and_check(&session, &exec, &task);
        assert_matches!(res, Ok(_));

        let mut input_file = input_file_path.create_file().unwrap();
        input_file
            .write_all(INPUT_FILE_CONTENTS2.as_bytes())
            .unwrap();
        input_file.flush().unwrap();

        let res = TaskState::load_and_check(&session, &exec, &task);
        assert_matches!(res, Err(_));
        let unwrap_err = res.unwrap_err();
        let mismatches = unwrap_err.children();
        assert_eq!(mismatches.len(), 1);
        assert_matches!(
            mismatches[0].downcast_ref::<TaskStateMismatch>().unwrap(),
            TaskStateMismatch::Inputs
        );
    }

    #[test]
    fn state_mismatches_executor() {
        let session = Session::for_testing();
        let exec = ExecutorId::from("exec/1");
        let task = Task::new(TaskId::for_testing());
        let output = TaskOutput::default();

        let res = TaskState::save(&session, &exec, &task, &output);
        assert_matches!(res, Ok(()));

        let res = TaskState::load_and_check(&session, &exec, &task);
        assert_matches!(res, Ok(_));

        let res = TaskState::load_and_check(&session, &ExecutorId::from("exec/2"), &task);
        assert_matches!(res, Err(_));
        let unwrap_err = res.unwrap_err();
        let mismatches = unwrap_err.children();
        assert_eq!(mismatches.len(), 1);
        assert_matches!(
            mismatches[0].downcast_ref::<TaskStateMismatch>().unwrap(),
            TaskStateMismatch::Executor
        );
    }
}
