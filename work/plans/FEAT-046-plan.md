# Implementation Plan: FEAT-046 Path Pattern Matching

## Overview
Add a `match(path, pattern)` function that extracts named parameters from URL paths using Express-style patterns (`:param` and `*glob`).

## Implementation Steps

### Step 1: Add match builtin (pkg/parsley/evaluator/builtins.go)
Add `match(path, pattern)` function:
- Takes two string arguments
- Returns Dictionary on match, NULL on no match
- Supports `:name` for single segment capture
- Supports `*name` for rest/glob capture (returns Array)
- Supports literal segments

### Step 2: Tests (server/match_test.go)
- Basic parameter extraction: `/users/:id`
- Multiple parameters: `/users/:userId/posts/:postId`
- Glob capture: `/files/*path`
- Literal matching: `/users`
- Edge cases: trailing slashes, no match, extra segments
- Case sensitivity

### Step 3: Documentation
- Update reference.md with match() function docs
- Update CHEATSHEET.md with pattern matching section

### Step 4: Update spec
- Mark acceptance criteria
- Add implementation notes

## Progress Log
- [ ] Step 1: match builtin
- [ ] Step 2: tests
- [ ] Step 3: documentation
- [ ] Step 4: spec update
