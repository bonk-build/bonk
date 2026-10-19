// Copyright © 2025 Colden Cullen
// SPDX-License-Identifier: MIT

use std::any::Any;
use std::fmt::Debug;

macro_rules! maybe_trait {
    {#[cfg($meta:meta)] pub trait $trait:ident: $requirement:path {}} => {
        #[cfg(not($meta))]
        pub trait $trait: std::any::Any {}
        #[cfg($meta)]
        pub trait $trait: $requirement {}

        #[cfg(not($meta))]
        impl<T: std::any::Any> $trait for T {}
        #[cfg($meta)]
        impl<T: $requirement> $trait for T {}
    };
}

maybe_trait! {
    #[cfg(feature = "clone")]
    pub trait MaybeClone: dyn_clone::DynClone {}
}

maybe_trait! {
    #[cfg(feature = "eq")]
    pub trait MaybeEq: dyn_eq::DynEq {}
}

maybe_trait! {
    #[cfg(feature = "hash")]
    pub trait MaybeHash: dyn_hash::DynHash {}
}

maybe_trait! {
    #[cfg(feature = "serde-ser")]
    pub trait MaybeSerialize: erased_serde::Serialize {}
}

maybe_trait! {
    #[cfg(feature = "serde-de")]
    pub trait MaybeDeerialize: serde::de::DeserializeOwned {}
}

pub trait SuperAny
where
    Self: Any + Debug + MaybeClone + MaybeEq + MaybeHash + MaybeSerialize,
{
}
impl<T> SuperAny for T where T: Any + Debug + MaybeClone + MaybeEq + MaybeHash + MaybeSerialize {}

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
        T: SuperAny + MaybeDeerialize,
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

#[cfg(feature = "clone")]
dyn_clone::clone_trait_object!(SuperAny);

#[cfg(feature = "eq")]
mod eq {
    use super::*;
    use dyn_eq::DynEq;

    dyn_eq::eq_trait_object!(SuperAny);

    /// Forwarding impl for types that implement SuperAny
    impl<T> PartialEq<T> for dyn SuperAny
    where
        T: SuperAny + 'static,
    {
        fn eq(&self, other: &T) -> bool {
            self.eq(other as &dyn SuperAny)
        }
    }
}

#[cfg(feature = "hash")]
dyn_hash::hash_trait_object!(SuperAny);

#[cfg(feature = "serde-ser")]
erased_serde::serialize_trait_object!(SuperAny);

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
    pub struct DeferredDeserialize(
        Format,
        #[cfg_attr(feature = "serde-ser", serde(skip))] OnceCell<Box<dyn SuperAny>>,
    );

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
                    Box::<T>::new(match self.0 {
                        #[cfg(feature = "serde-de-json")]
                        Format::Json(ref value) => serde_json::from_value(value.clone()).unwrap(),
                    })
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
    use super::*;
    use std::ops::AddAssign;

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
        let val = 8i8;
        let any: Box<dyn SuperAny> = Box::new(val);
        let any = any.clone();

        assert_eq!(*any.downcast_ref::<i8>().unwrap(), val);
        assert_ne!(*any.downcast_ref::<i8>().unwrap(), 7);
    }

    #[cfg(feature = "eq")]
    #[test]
    fn eq() {
        let val = 8;
        let any = &val as &dyn SuperAny;

        assert_eq!(any, &val);
        assert_ne!(any, &7);
    }

    #[cfg(feature = "hash")]
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
