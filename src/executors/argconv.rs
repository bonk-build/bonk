// Copyright © 2025 Colden Cullen
// SPDX-License-Identifier: MIT

use super_any::SuperAny;

use crate::{Executor, ExecutorId, SessionId, Task, TaskResult};

pub trait TypedExecutor {
    type ArgType: SuperAny + serde::de::DeserializeOwned;

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
    ArgType: SuperAny + serde::de::DeserializeOwned,
{
    fn execute(&mut self, session: &SessionId, executor: &ExecutorId, task: &Task) -> TaskResult {
        TypedExecutor::execute(self, session, executor, task, unwrap(task)?)
    }
}

fn unwrap<ArgType>(task: &Task) -> lazy_errors::Result<&ArgType>
where
    ArgType: SuperAny + serde::de::DeserializeOwned,
{
    task.args()
        .ok_or(lazy_errors::err!("no args provided"))?
        .downcast_ref()
        .ok_or(lazy_errors::err!("invalid type conversion"))
}

#[cfg(test)]
mod tests {
    use super_any::DeferredDeserialize;

    use super::*;
    use crate::TaskId;
    use std::{assert_matches, cell::LazyCell};

    #[derive(Clone, Debug, PartialEq, Eq, Hash, serde::Serialize, serde::Deserialize)]
    struct Args {
        val1: String,
        val2: i32,
    }

    const DEFAULT_ARGS: LazyCell<Box<Args>> = LazyCell::new(|| {
        Box::new(Args {
            val1: "test string".into(),
            val2: 69420,
        })
    });

    #[test]
    fn straight_conversion() {
        let task = Task::builder(TaskId::for_testing())
            .args(Box::new(Args::clone(DEFAULT_ARGS.as_ref())))
            .build();

        let args = unwrap::<Args>(&task);
        assert_matches!(args, Ok(_));
        assert_eq!(args.unwrap(), DEFAULT_ARGS.as_ref());
    }

    #[test]
    fn json_conversion() {
        let json = serde_json::to_value(DEFAULT_ARGS.as_ref()).unwrap();

        let task = Task::builder(TaskId::for_testing())
            .args(Box::new(DeferredDeserialize::from(json)))
            .build();

        let args = unwrap::<Args>(&task);
        assert_matches!(args, Ok(_));
        assert_eq!(args.unwrap(), DEFAULT_ARGS.as_ref());
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
