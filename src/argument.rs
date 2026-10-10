// Copyright © 2025 Colden Cullen
// SPDX-License-Identifier: MIT

use std::any::Any;
use std::hash::Hash;

use derive_more::Deref;
use dyn_eq::DynEq;
use dyn_hash::DynHash;

#[typetag::serde(tag = "type")]
pub trait Argument: Any + DynHash + DynEq {}

// Implement some necessary forwards
impl dyn Argument {
    pub fn downcast_ref<T: Any>(&self) -> Option<&T> {
        <dyn Any>::downcast_ref(self)
    }
}

dyn_hash::hash_trait_object!(Argument);
dyn_eq::eq_trait_object!(Argument);

// Automatically implement
#[typetag::serialize]
impl<T> Argument for T
where
    T: Any + DynHash + DynEq + serde::Serialize,
{
    #[doc(hidden)]
    fn typetag_deserialize(&self) {}
}

#[derive(Deref, Hash)]
pub struct ArgumentHolder {
    #[deref]
    value: Box<dyn Argument>,
}

impl ArgumentHolder {
    fn new<T>(value: T) -> Self
    where
        T: Argument + 'static,
    {
        let value = Box::new(value);
        Self { value }
    }
}
