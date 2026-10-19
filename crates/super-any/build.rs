// Copyright © 2025 Colden Cullen
// SPDX-License-Identifier: MIT

fn maybe_set_feature(set: bool, value: &str) {
    const KEY: &str = "feature";

    build_rs::output::rustc_check_cfg_values(KEY, &[value]);
    if set {
        build_rs::output::rustc_cfg_value(KEY, value);
    }
}

fn main() {
    // Enable serde-de if any serde-de-* is set
    maybe_set_feature(
        build_rs::input::cargo_cfg_feature()
            .iter()
            .any(|feat| feat.starts_with("serde-de-")),
        "serde-de",
    );
}
