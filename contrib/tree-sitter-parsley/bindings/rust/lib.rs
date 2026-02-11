//! This crate provides Parsley language support for the [tree-sitter][] parsing library.
//!
//! Typically, you will use the [LANGUAGE][] constant to add this language to a
//! tree-sitter [Parser][], and then use the parser to parse some code:
//!
//! ```
//! let code = r#"
//! let x = 42
//! export greeting = "Hello, world!"
//! "#;
//! let mut parser = tree_sitter::Parser::new();
//! let language = tree_sitter_parsley::LANGUAGE;
//! parser
//!     .set_language(&language.into())
//!     .expect("Error loading Parsley parser");
//! let tree = parser.parse(code, None).unwrap();
//! assert!(!tree.root_node().has_error());
//! ```
//!
//! [Parser]: https://docs.rs/tree-sitter/*/tree_sitter/struct.Parser.html
//! [tree-sitter]: https://tree-sitter.github.io/

use tree_sitter_language::LanguageFn;

extern "C" {
    fn tree_sitter_parsley() -> *const ();
}

/// The tree-sitter [`LanguageFn`] for Parsley.
///
/// [LanguageFn]: https://docs.rs/tree-sitter-language/*/tree_sitter_language/struct.LanguageFn.html
pub const LANGUAGE: LanguageFn = unsafe { LanguageFn::from_raw(tree_sitter_parsley) };

/// The content of the [`queries/highlights.scm`][] file.
pub const HIGHLIGHTS_QUERY: &str = include_str!("../../queries/highlights.scm");

/// The tree-sitter node types as JSON.
pub const NODE_TYPES: &str = include_str!("../../src/node-types.json");

#[cfg(test)]
mod tests {
    #[test]
    fn test_can_load_grammar() {
        let mut parser = tree_sitter::Parser::new();
        parser
            .set_language(&super::LANGUAGE.into())
            .expect("Error loading Parsley parser");
    }
}
