// Copyright © 2025 Colden Cullen
// SPDX-License-Identifier: MIT

use std::any::Any;
use std::fmt::Debug;

#[cfg(feature = "clone")]
use dyn_clone::DynClone;
#[cfg(feature = "eq")]
use dyn_eq::DynEq;
#[cfg(feature = "hash")]
use dyn_hash::DynHash;

cfg_select! {
    all(feature = "clone", feature = "eq", feature = "hash", feature = "serde-ser") => {
        pub trait SuperAny: Any + Debug + DynClone + DynEq + DynHash + erased_serde::Serialize {}
        impl<T> SuperAny for T where T: Any + Debug + DynClone + DynEq + DynHash + erased_serde::Serialize {}
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

#[cfg(feature = "clone")]
dyn_clone::clone_trait_object!(SuperAny);

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
    pub fn is<T>(&self) -> bool
    where
        T: SuperAny,
    {
        <dyn Any>::is::<T>(self)
    }

    pub fn downcast_ref<T>(&self) -> Option<&T>
    where
        T: SuperAny + serde::de::DeserializeOwned,
    {
        #[cfg(feature = "serde-de")]
        if let Some(deferred) = <dyn Any>::downcast_ref::<serde_de::DeferredDeserialize>(self) {
            return deferred.deserialize();
        }

        <dyn Any>::downcast_ref(self)
    }

    pub fn downcast_mut<T>(&mut self) -> Option<&mut T>
    where
        T: SuperAny,
    {
        <dyn Any>::downcast_mut(self)
    }
}

#[cfg(feature = "serde-de")]
mod serde_de {
    use std::cell::OnceCell;

    use super::*;

    #[derive(Debug)]
    #[cfg_attr(feature = "clone", derive(Clone))]
    #[cfg_attr(feature = "eq", derive(PartialEq, Eq))]
    #[cfg_attr(feature = "hash", derive(Hash))]
    #[cfg_attr(feature = "serde-ser", derive(serde::Serialize))]
    enum Format {
        #[cfg(feature = "serde-de-json")]
        Json(serde_json::Value),
    }

    #[derive(Debug)]
    #[cfg_attr(feature = "clone", derive(Clone))]
    #[cfg_attr(feature = "eq", derive(PartialEq, Eq))]
    #[cfg_attr(feature = "serde-ser", derive(serde::Serialize))]
    pub struct DeferredDeserialize(Format, #[serde(skip)] OnceCell<Box<dyn SuperAny>>);

    #[cfg(feature = "hash")]
    impl std::hash::Hash for DeferredDeserialize {
        fn hash<H: std::hash::Hasher>(&self, state: &mut H) {
            if let Some(de) = self.1.get() {
                de.hash(state);
            } else {
                self.0.hash(state);
            }
        }
    }

    impl DeferredDeserialize {
        pub fn deserialize<T>(&self) -> Option<&T>
        where
            T: SuperAny + serde::de::DeserializeOwned,
        {
            self.1
                .get_or_init(|| {
                    let result: T = match self.0 {
                        #[cfg(feature = "serde-de-json")]
                        Format::Json(ref value) => serde_json::from_value(value.clone()).unwrap(),

                        #[allow(unreachable_patterns)]
                        _ => panic!(),
                    };

                    Box::new(result)
                })
                .downcast_ref()
        }
    }

    #[cfg(feature = "serde-de-json")]
    impl From<serde_json::Value> for DeferredDeserialize {
        fn from(value: serde_json::Value) -> Self {
            Self(Format::Json(value), OnceCell::new())
        }
    }
}

#[cfg(feature = "serde-de")]
pub use serde_de::*;

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

    #[cfg(feature = "clone")]
    #[test]
    fn clone() {
        let val = 8;
        let any: Box<dyn SuperAny> = Box::new(val);
        let any = any.clone();

        assert_eq!(*any, val);
        assert_ne!(*any, 7);
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

    #[cfg(feature = "serde-de-json")]
    #[test]
    fn serde_json() {
        let val = 8i8;
        let json = serde_json::to_value(val).unwrap();
        let any = &DeferredDeserialize::from(json) as &dyn SuperAny;

        assert_eq!(&val, any.downcast_ref::<i8>().unwrap());
    }
}
