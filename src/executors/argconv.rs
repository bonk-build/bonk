// Copyright © 2025 Colden Cullen
// SPDX-License-Identifier: MIT

use std::any::Any;

use crate::{Executor, ExecutorId, SessionId, Task, TaskResult};

pub trait TypedExecutor {
    type ArgType: Any;

    fn execute(
        &mut self,
        session: &SessionId,
        executor: &ExecutorId,
        task: &Task,
        arg: &Self::ArgType,
    ) -> TaskResult;
}

impl<T, ArgType> Executor for T
where
    T: TypedExecutor<ArgType = ArgType>,
    ArgType: Any,
{
    fn execute(&mut self, session: &SessionId, executor: &ExecutorId, task: &Task) -> TaskResult {
        TypedExecutor::execute(self, session, executor, task, unwrap(task)?)
    }
}

fn unwrap<ArgType: 'static>(task: &Task) -> lazy_errors::Result<&ArgType> {
    task.args()
        .ok_or(lazy_errors::err!("no args provided"))?
        .downcast_ref()
        .ok_or(lazy_errors::err!("invalid type conversion"))
}

#[cfg(test)]
mod tests {
    use super::*;
    use crate::TaskId;
    use std::assert_matches;

    #[derive(Debug, PartialEq, Eq, Hash, serde::Serialize)]
    struct Args {
        val1: &'static str,
        val2: i32,
    }

    const DEFAULT_ARGS: Args = Args {
        val1: "test string",
        val2: 69420,
    };

    #[test]
    fn straight_conversion() {
        let task = Task::builder(TaskId::for_testing())
            .args(Box::new(DEFAULT_ARGS))
            .build();

        let args = unwrap::<Args>(&task);
        assert_matches!(args, Ok(_));
        assert_eq!(args.unwrap(), &DEFAULT_ARGS);
    }

    #[test]
    fn conversion_failure() {
        let task = Task::builder(TaskId::for_testing())
            .args(Box::new(()))
            .build();

        let args = unwrap::<Args>(&task);
        assert_matches!(args, Err(_));
    }
}
