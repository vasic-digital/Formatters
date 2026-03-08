#!/usr/bin/env bash
# formatters_functionality_challenge.sh - Validates Formatters module core functionality and structure
set -euo pipefail

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
MODULE_DIR="$(cd "${SCRIPT_DIR}/../.." && pwd)"
MODULE_NAME="Formatters"

PASS=0
FAIL=0
TOTAL=0

pass() { PASS=$((PASS+1)); TOTAL=$((TOTAL+1)); echo "  PASS: $1"; }
fail() { FAIL=$((FAIL+1)); TOTAL=$((TOTAL+1)); echo "  FAIL: $1"; }

echo "=== ${MODULE_NAME} Functionality Challenge ==="
echo ""

# Test 1: Required packages exist
echo "Test: Required packages exist"
pkgs_ok=true
for pkg in formatter registry executor cache native service; do
    if [ ! -d "${MODULE_DIR}/pkg/${pkg}" ]; then
        fail "Missing package: pkg/${pkg}"
        pkgs_ok=false
    fi
done
if [ "$pkgs_ok" = true ]; then
    pass "All required packages present (formatter, registry, executor, cache, native, service)"
fi

# Test 2: Formatter interface is defined
echo "Test: Formatter interface is defined"
if grep -rq "type Formatter interface" "${MODULE_DIR}/pkg/formatter/"; then
    pass "Formatter interface is defined in pkg/formatter"
else
    fail "Formatter interface not found in pkg/formatter"
fi

# Test 3: FormatRequest struct exists
echo "Test: FormatRequest struct exists"
if grep -rq "type FormatRequest struct" "${MODULE_DIR}/pkg/formatter/"; then
    pass "FormatRequest struct is defined"
else
    fail "FormatRequest struct not found"
fi

# Test 4: FormatResult struct exists
echo "Test: FormatResult struct exists"
if grep -rq "type FormatResult struct" "${MODULE_DIR}/pkg/formatter/"; then
    pass "FormatResult struct is defined"
else
    fail "FormatResult struct not found"
fi

# Test 5: Registry implementation exists
echo "Test: Registry implementation exists"
if grep -rq "type\s\+\w*Registry\w*\s\+struct\|type\s\+\w*registry\w*\s\+struct" "${MODULE_DIR}/pkg/registry/"; then
    pass "Registry implementation exists in pkg/registry"
else
    fail "No registry implementation found in pkg/registry"
fi

# Test 6: Executor implementation exists
echo "Test: Executor implementation exists"
if grep -rq "type\s\+\w*Executor\w*\s\+struct\|type\s\+\w*executor\w*\s\+struct" "${MODULE_DIR}/pkg/executor/"; then
    pass "Executor implementation exists in pkg/executor"
else
    fail "No executor implementation found in pkg/executor"
fi

# Test 7: Native formatter providers exist
echo "Test: Native formatter providers exist"
if [ -d "${MODULE_DIR}/pkg/native" ] && [ "$(find "${MODULE_DIR}/pkg/native" -name "*.go" ! -name "*_test.go" | wc -l)" -gt 0 ]; then
    pass "Native formatter providers found"
else
    fail "No native formatter providers found"
fi

# Test 8: Service formatter support exists
echo "Test: Service formatter support exists"
if [ -d "${MODULE_DIR}/pkg/service" ] && [ "$(find "${MODULE_DIR}/pkg/service" -name "*.go" ! -name "*_test.go" | wc -l)" -gt 0 ]; then
    pass "Service formatter support found"
else
    fail "No service formatter support found"
fi

# Test 9: Format cache support
echo "Test: Format cache support exists"
if grep -rq "FormatCache\|Cache" "${MODULE_DIR}/pkg/cache/"; then
    pass "Format cache support found"
else
    fail "No format cache support found"
fi

# Test 10: FormatterType or language support
echo "Test: Formatter type/language support exists"
if grep -rq "FormatterType\|Language\|language" "${MODULE_DIR}/pkg/formatter/"; then
    pass "Formatter type/language support found"
else
    fail "No formatter type/language support found"
fi

echo ""
echo "=== Results: ${PASS}/${TOTAL} passed, ${FAIL} failed ==="
[ "${FAIL}" -eq 0 ] && exit 0 || exit 1
