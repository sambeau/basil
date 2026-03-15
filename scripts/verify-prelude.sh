#!/bin/bash
# verify-prelude.sh - Verification script for FEAT-143 prelude components
# Tests Parsley syntax fixes and validates component files parse correctly

set -e

PARS="./pars"
PASS=0
FAIL=0

echo "=== Verifying Prelude Components (FEAT-143) ==="
echo ""

# Helper function for syntax checks
check_syntax() {
    local name="$1"
    local file="$2"

    printf "%-25s" "$name:"
    if $PARS --check "$file" 2>/dev/null; then
        echo "OK"
        PASS=$((PASS + 1))
    else
        echo "FAIL"
        FAIL=$((FAIL + 1))
    fi
}

# Helper function for expression tests
check_expr() {
    local name="$1"
    local expr="$2"
    local pattern="$3"

    printf "%-25s" "$name:"
    if $PARS -e "$expr" 2>/dev/null | grep -q "$pattern"; then
        echo "OK"
        PASS=$((PASS + 1))
    else
        echo "FAIL (expected: $pattern)"
        FAIL=$((FAIL + 1))
    fi
}

echo "--- Syntax Checks (Phase 4 fixes) ---"
echo ""

# Verify all component files parse correctly
for f in server/prelude/components/*.pars; do
    name=$(basename "$f" .pars)
    check_syntax "$name" "$f"
done

echo ""
echo "--- Language Construct Tests ---"
echo ""

# Test for-loop ordering: (idx, item) not (item, idx)
check_expr "for-loop ordering" \
    'for (idx, item in ["a", "b", "c"]) { {i: idx, v: item} }' \
    'i: 0'

# Test that first index is 0
check_expr "for-loop index starts at 0" \
    'for (i, x in [10, 20, 30]) { i }' \
    '0'

# Test range operator is inclusive
check_expr "range inclusive (1..5)" \
    'for (n in 1..5) { n }' \
    '5'

# Test range with variables
check_expr "range with variables" \
    'let start = 3; let end = 7; for (n in start..end) { n }' \
    '7'

# Test spread syntax in tag attributes (this tests the parser accepts it)
check_expr "spread syntax parses" \
    'let attrs = {class: "test"}; <div ...attrs/>' \
    'class='

echo ""
echo "--- Component Logic Tests ---"
echo ""

# Test accordion first-open logic pattern
check_expr "accordion first-open logic" \
    'let items = [{t: "Q1"}, {t: "Q2"}]; for (i, item in items) { {idx: i, open: (i == 0)} }' \
    'open: true'

# Test breadcrumb position (1-based)
check_expr "breadcrumb position (1-based)" \
    'let items = ["Home", "About", "Contact"]; for (idx, item in items) { idx + 1 }' \
    '1, 2, 3'

# Test unique ID generation pattern (using string conversion)
check_expr "unique ID generation" \
    'let name = "size"; for (idx, opt in ["S", "M", "L"]) { "field-" + name + "-" + idx.fmt() }' \
    'field-size-0'

# Test pagination range calculation
check_expr "pagination window range" \
    'let {max, min} = import @std/math; let current = 5; let pageWindow = 2; let totalPages = 10; let start = max(1, current - pageWindow); let end = min(totalPages, current + pageWindow); for (n in start..end) { n }' \
    '3, 4, 5, 6, 7'

echo ""
echo "=== Summary ==="
echo "Passed: $PASS"
echo "Failed: $FAIL"
echo ""

if [ $FAIL -eq 0 ]; then
    echo "✓ All verification checks passed"
    exit 0
else
    echo "✗ Some checks failed"
    exit 1
fi
