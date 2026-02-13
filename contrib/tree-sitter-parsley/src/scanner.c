/**
 * External scanner for Parsley tree-sitter grammar
 *
 * Handles context-sensitive tokenization that the pure JS grammar cannot express:
 * 1. Raw text tags: <style> and <script> content as raw text with @{} interpolation
 *
 * The key insight is that tree-sitter's valid_symbols array tells us what tokens
 * are valid at the current parse position. If RAW_TEXT is valid, we must be inside
 * a style/script tag (since that's the only place we declared it in the grammar).
 *
 * Reference: pkg/parsley/lexer/lexer.go (nextTagContentToken, readRawTagText)
 */

#include "tree_sitter/parser.h"
#include <string.h>
#include <stdbool.h>
#include <stdlib.h>

// Token types that the external scanner can produce
// These must match the order in grammar.js externals array
enum TokenType {
    RAW_TEXT,
    RAW_TEXT_INTERPOLATION_START,
    ERROR_SENTINEL
};

// Scanner state - tracks which raw text tag we're in
typedef struct {
    // 0 = not in raw text, 1 = in <style>, 2 = in <script>
    uint8_t raw_text_mode;
} Scanner;

// Create a new scanner instance
void *tree_sitter_parsley_external_scanner_create(void) {
    Scanner *scanner = (Scanner *)calloc(1, sizeof(Scanner));
    return scanner;
}

// Destroy the scanner instance
void tree_sitter_parsley_external_scanner_destroy(void *payload) {
    Scanner *scanner = (Scanner *)payload;
    free(scanner);
}

// Serialize scanner state for tree-sitter's GLR backtracking
unsigned tree_sitter_parsley_external_scanner_serialize(
    void *payload,
    char *buffer
) {
    Scanner *scanner = (Scanner *)payload;
    buffer[0] = scanner->raw_text_mode;
    return 1;
}

// Deserialize scanner state
void tree_sitter_parsley_external_scanner_deserialize(
    void *payload,
    const char *buffer,
    unsigned length
) {
    Scanner *scanner = (Scanner *)payload;
    if (length >= 1) {
        scanner->raw_text_mode = (uint8_t)buffer[0];
    } else {
        scanner->raw_text_mode = 0;
    }
}

// Helper: advance the lexer (include character in token)
static void advance(TSLexer *lexer) {
    lexer->advance(lexer, false);
}

// Helper: check if next chars match a string (case-insensitive for tag names)
static bool lookahead_matches_string(TSLexer *lexer, const char *str) {
    for (size_t i = 0; str[i] != '\0'; i++) {
        int32_t c = lexer->lookahead;
        char expected = str[i];
        
        // Case-insensitive comparison for letters
        if (c >= 'A' && c <= 'Z') c = c + ('a' - 'A');
        if (expected >= 'A' && expected <= 'Z') expected = expected + ('a' - 'A');
        
        if (c != expected) {
            return false;
        }
        advance(lexer);
    }
    return true;
}

/**
 * Scan for raw text content in <style> or <script> tags
 *
 * In raw text mode:
 * - Everything is literal text until we see the matching </style> or </script>, or @{
 * - `@{` triggers interpolation (return RAW_TEXT for content before it)
 * - `{` and `}` are literal (NOT Parsley blocks/dicts)
 * - `//` comments are preserved (valid JS, harmless in CSS)
 * - `</` inside JS strings like '</div>' should NOT end the tag
 *
 * We detect which tag we're in by scanning backwards or by looking at what
 * the grammar tells us via valid_symbols.
 */
static bool scan_raw_text(Scanner *scanner, TSLexer *lexer, const bool *valid_symbols) {
    bool has_content = false;
    
    // Determine which close tag to look for
    // We need to detect this from context. The grammar structure tells us:
    // - If we're in style_tag rule, look for </style>
    // - If we're in script_tag rule, look for </script>
    // 
    // Since both use the same external token, we look for either.
    // The close tag literal in the grammar will handle the final match.
    
    while (true) {
        // EOF
        if (lexer->eof(lexer)) {
            break;
        }
        
        int32_t c = lexer->lookahead;
        
        // Check for @{ interpolation start
        if (c == '@') {
            // Mark position before @
            lexer->mark_end(lexer);
            advance(lexer);
            
            if (lexer->lookahead == '{') {
                // Found @{ - if we have content, return it first
                if (has_content) {
                    lexer->result_symbol = RAW_TEXT;
                    return true;
                }
                // Otherwise, return the @{ as interpolation start
                advance(lexer); // consume {
                lexer->mark_end(lexer);
                lexer->result_symbol = RAW_TEXT_INTERPOLATION_START;
                return true;
            }
            // Not @{, this @ is part of raw text - continue
            has_content = true;
            lexer->mark_end(lexer);
            continue;
        }
        
        // Check for closing tag </style> or </script>
        if (c == '<') {
            // Mark position before <
            lexer->mark_end(lexer);
            
            // Save position to potentially backtrack
            advance(lexer);
            
            if (lexer->lookahead == '/') {
                advance(lexer);
                
                // Check if this is </style>, </script>, or </SQL>
                // We need to peek ahead without consuming if it doesn't match
                
                // Check for 'SQL' (case-sensitive, uppercase only)
                if (lexer->lookahead == 'S') {
                    advance(lexer);
                    if (lexer->lookahead == 'Q') {
                        advance(lexer);
                        if (lexer->lookahead == 'L') {
                            advance(lexer);
                            // Check for end of tag name
                            if (lexer->lookahead == '>' || lexer->lookahead == ' ' ||
                                lexer->lookahead == '\t' || lexer->lookahead == '\n' ||
                                lexer->lookahead == '\r') {
                                // Found </SQL>!
                                if (has_content) {
                                    // Return accumulated content, let grammar handle close tag
                                    lexer->result_symbol = RAW_TEXT;
                                    return true;
                                }
                                // No content - decline and let grammar handle </SQL>
                                return false;
                            }
                        }
                    }
                    // Not SQL, but started with 'S' - could still be 'style' or 'script'
                    // Check for 'style' (already consumed 'S')
                    if (lexer->lookahead == 't' || lexer->lookahead == 'T') {
                        // Likely 'style'
                        advance(lexer);
                        if ((lexer->lookahead == 'y' || lexer->lookahead == 'Y')) {
                            advance(lexer);
                            if ((lexer->lookahead == 'l' || lexer->lookahead == 'L')) {
                                advance(lexer);
                                if ((lexer->lookahead == 'e' || lexer->lookahead == 'E')) {
                                    advance(lexer);
                                    // Check for end of tag name
                                    if (lexer->lookahead == '>' || lexer->lookahead == ' ' ||
                                        lexer->lookahead == '\t' || lexer->lookahead == '\n' ||
                                        lexer->lookahead == '\r') {
                                        // Found </style>!
                                        if (has_content) {
                                            lexer->result_symbol = RAW_TEXT;
                                            return true;
                                        }
                                        return false;
                                    }
                                }
                            }
                        }
                    } else if (lexer->lookahead == 'c' || lexer->lookahead == 'C') {
                        // Likely 'script'
                        advance(lexer);
                        if ((lexer->lookahead == 'r' || lexer->lookahead == 'R')) {
                            advance(lexer);
                            if ((lexer->lookahead == 'i' || lexer->lookahead == 'I')) {
                                advance(lexer);
                                if ((lexer->lookahead == 'p' || lexer->lookahead == 'P')) {
                                    advance(lexer);
                                    if ((lexer->lookahead == 't' || lexer->lookahead == 'T')) {
                                        advance(lexer);
                                        // Check for end of tag name
                                        if (lexer->lookahead == '>' || lexer->lookahead == ' ' ||
                                            lexer->lookahead == '\t' || lexer->lookahead == '\n' ||
                                            lexer->lookahead == '\r') {
                                            // Found </script>!
                                            if (has_content) {
                                                lexer->result_symbol = RAW_TEXT;
                                                return true;
                                            }
                                            return false;
                                        }
                                    }
                                }
                            }
                        }
                    }
                } else if (lexer->lookahead == 's') {
                    // Lowercase 's' - check for 'style' or 'script' (case-insensitive)
                    // Could be style or script
                    advance(lexer);
                    
                    if (lexer->lookahead == 't' || lexer->lookahead == 'T') {
                        // Likely 'style'
                        advance(lexer);
                        if ((lexer->lookahead == 'y' || lexer->lookahead == 'Y')) {
                            advance(lexer);
                            if ((lexer->lookahead == 'l' || lexer->lookahead == 'L')) {
                                advance(lexer);
                                if ((lexer->lookahead == 'e' || lexer->lookahead == 'E')) {
                                    advance(lexer);
                                    // Check for end of tag name
                                    if (lexer->lookahead == '>' || lexer->lookahead == ' ' ||
                                        lexer->lookahead == '\t' || lexer->lookahead == '\n' ||
                                        lexer->lookahead == '\r') {
                                        // Found </style>!
                                        if (has_content) {
                                            // Return accumulated content, let grammar handle close tag
                                            lexer->result_symbol = RAW_TEXT;
                                            return true;
                                        }
                                        // No content - decline and let grammar handle </style>
                                        return false;
                                    }
                                }
                            }
                        }
                    } else if (lexer->lookahead == 'c' || lexer->lookahead == 'C') {
                        // Likely 'script'
                        advance(lexer);
                        if ((lexer->lookahead == 'r' || lexer->lookahead == 'R')) {
                            advance(lexer);
                            if ((lexer->lookahead == 'i' || lexer->lookahead == 'I')) {
                                advance(lexer);
                                if ((lexer->lookahead == 'p' || lexer->lookahead == 'P')) {
                                    advance(lexer);
                                    if ((lexer->lookahead == 't' || lexer->lookahead == 'T')) {
                                        advance(lexer);
                                        // Check for end of tag name
                                        if (lexer->lookahead == '>' || lexer->lookahead == ' ' ||
                                            lexer->lookahead == '\t' || lexer->lookahead == '\n' ||
                                            lexer->lookahead == '\r') {
                                            // Found </script>!
                                            if (has_content) {
                                                // Return accumulated content, let grammar handle close tag
                                                lexer->result_symbol = RAW_TEXT;
                                                return true;
                                            }
                                            // No content - decline and let grammar handle </script>
                                            return false;
                                        }
                                    }
                                }
                            }
                        }
                    }
                }
                
                // Not </style>, </script>, or </SQL>, this is raw text content
                // (like '</div>' or '</li>' in JavaScript strings)
                has_content = true;
                lexer->mark_end(lexer);
                continue;
            }
            
            // Not </, this < is part of raw text (like < operator in JS)
            has_content = true;
            lexer->mark_end(lexer);
            continue;
        }
        
        // Any other character is raw text
        advance(lexer);
        has_content = true;
        lexer->mark_end(lexer);
    }
    
    // Return accumulated content at EOF
    if (has_content) {
        lexer->result_symbol = RAW_TEXT;
        return true;
    }
    
    return false;
}

/**
 * Main scan function called by tree-sitter
 *
 * @param payload The scanner state
 * @param lexer The tree-sitter lexer interface
 * @param valid_symbols Array indicating which tokens are valid in current parse state
 * @return true if a token was produced, false to fall back to grammar rules
 */
bool tree_sitter_parsley_external_scanner_scan(
    void *payload,
    TSLexer *lexer,
    const bool *valid_symbols
) {
    Scanner *scanner = (Scanner *)payload;
    
    // If RAW_TEXT or RAW_TEXT_INTERPOLATION_START is valid, we're inside a 
    // style/script tag and should scan for raw text content
    if (valid_symbols[RAW_TEXT] || valid_symbols[RAW_TEXT_INTERPOLATION_START]) {
        return scan_raw_text(scanner, lexer, valid_symbols);
    }
    
    // For all other cases, decline and let the grammar handle it
    return false;
}