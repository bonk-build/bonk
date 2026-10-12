// Copyright © 2025 Colden Cullen
// SPDX-License-Identifier: MIT

#![allow(unused_imports)]

use std::any::Any;
use std::fmt::Debug;

#[cfg(feature = "eq")]
use dyn_eq::DynEq;
#[cfg(feature = "hash")]
use dyn_hash::DynHash;

cfg_select! {
    all(feature = "eq", feature = "hash", feature = "serde-ser") => {
        pub trait SuperAny: Any + Debug + DynEq + DynHash + erased_serde::Serialize {}
        impl<T> SuperAny for T where T: Any + Debug + DynEq + DynHash + erased_serde::Serialize {}
    }

    all(feature = "eq", feature = "hash") => {
        pub trait SuperAny: Any + Debug + DynHash + DynEq {}
        impl<T> SuperAny for T where T: Any + Debug + DynHash + DynEq {}
    }

    all(feature = "eq") => {
        pub trait SuperAny: Any + Debug + DynEq {}
        impl<T> SuperAny for T where T: Any + Debug + DynEq {}
    }

    all(feature = "hash") => {
        pub trait SuperAny: Any + Debug + DynHash {}
        impl<T> SuperAny for T where T: Any + Debug + DynHash {}
    }

    _ => {
        pub trait SuperAny: Any {}
        impl<T> SuperAny for T where T: Any {}
    }
}

#[cfg(feature = "eq")]
dyn_eq::eq_trait_object!(SuperAny);

#[cfg(feature = "hash")]
dyn_hash::hash_trait_object!(SuperAny);

#[cfg(feature = "serde-ser")]
erased_serde::serialize_trait_object!(SuperAny);

/// Forwarding impl for types that implement SuperAny
#[cfg(feature = "eq")]
impl<T> PartialEq<T> for dyn SuperAny
where
    T: SuperAny + 'static,
{
    fn eq(&self, other: &T) -> bool {
        self.eq(other as &dyn SuperAny)
    }
}

// Implement some necessary forwards to Any
impl dyn SuperAny {
    pub fn is<T: Any>(&self) -> bool {
        <dyn Any>::is::<T>(self)
    }
    pub fn downcast_ref<T: Any>(&self) -> Option<&T> {
        <dyn Any>::downcast_ref(self)
    }
    pub fn downcast_mut<T: Any>(&mut self) -> Option<&mut T> {
        <dyn Any>::downcast_mut(self)
    }
}

#[cfg(test)]
mod tests {
    use std::ops::AddAssign;

    use super::*;

    #[test]
    fn type_id() {
        let val = 8i8;
        let any = &val as &dyn SuperAny;

        assert_eq!(val.type_id(), any.type_id());
    }

    #[test]
    fn is() {
        let val = 8i8;
        let any = &val as &dyn SuperAny;

        assert!(any.is::<i8>());
    }

    #[test]
    fn downcast() {
        let val = 8i8;
        let any = &val as &dyn SuperAny;

        assert_eq!(&val, any.downcast_ref::<i8>().unwrap());
    }

    #[test]
    fn downcast_mut() {
        let mut val = 8i8;

        {
            let any = &mut val as &mut dyn SuperAny;
            any.downcast_mut::<i8>().unwrap().add_assign(1);
            assert_eq!(&9, any.downcast_ref::<i8>().unwrap());
        }

        assert_eq!(9, val);
    }

    #[test]
    fn debug() {
        let val = 8;
        let any = &val as &dyn SuperAny;

        assert_eq!(format!("{val:?}"), format!("{any:?}"));
    }

    #[cfg(feature = "eq")]
    #[test]
    fn eq() {
        let val = 8;
        let any = &val as &dyn SuperAny;

        assert_eq!(any, &val);
        assert_ne!(any, &7);
    }

    #[cfg(feature = "eq")]
    #[test]
    fn hash() {
        fn hash_object<T: std::hash::Hash>(val: &T) -> u64 {
            use std::hash::{DefaultHasher, Hasher};

            let mut hasher = DefaultHasher::new();
            val.hash(&mut hasher);
            hasher.finish()
        }

        let val = 8;
        let any = &val as &dyn SuperAny;

        assert_eq!(hash_object(&val), hash_object(&any));
        assert_ne!(hash_object(&7), hash_object(&any));
    }

    #[cfg(feature = "serde-ser")]
    #[test]
    fn serde_ser() {
        let val = 8;
        let any = &val as &dyn SuperAny;

        let val_json = serde_json::to_string(&val).unwrap();
        let any_json = serde_json::to_string(&any).unwrap();

        assert_eq!(val_json, any_json);

        let other_json = serde_json::to_string(&7).unwrap();
        assert_ne!(other_json, any_json);
    }
}
