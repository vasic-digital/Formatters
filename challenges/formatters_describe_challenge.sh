#!/usr/bin/env bash
# formatters_describe_challenge.sh — Round-255 paired-mutation gate.
#
# Wraps challenges/runner with:
#   1. Baseline run (en + sr locales) → MUST exit 0
#   2. Eight paired-mutation runs (one per check) → EACH MUST exit 99
#
# A baseline-failure or any mutation-not-flipping invariant fails this gate.
# CONST-035 / Article XI §11.9 / §1.1: a check that cannot be broken on
# demand cannot be trusted to detect real regressions.
#
# Exit codes:
#   0 — baseline PASS + all 8 mutations triggered exit 99
#   1 — baseline failed OR a mutation failed to flip its invariant
set -euo pipefail

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
SUBMODULE_ROOT="$(cd "${SCRIPT_DIR}/.." && pwd)"
cd "${SUBMODULE_ROOT}"

PASS=0
FAIL=0

echo "=== Formatters describe Challenge (round 255 — paired mutation) ==="
echo "submodule_root=${SUBMODULE_ROOT}"
echo

# Build the runner binary once — `go run` masks user exit codes >1 (returns 1)
# so paired-mutation exit-99 detection requires a real binary on disk.
RUNNER_BIN="$(mktemp -d)/formatters_round255_runner"
trap 'rm -rf "$(dirname "${RUNNER_BIN}")"' EXIT
echo "[step 0] building runner binary at ${RUNNER_BIN}"
go build -o "${RUNNER_BIN}" ./challenges/runner

# --- Step 1: baselines for both locales -------------------------------------
for loc in en sr; do
    echo "[step 1.${loc}] baseline run lang=${loc}"
    if "${RUNNER_BIN}" -lang="${loc}" >/tmp/formatters_round255_${loc}.log 2>&1; then
        echo "  PASS: baseline ${loc} exited 0"
        PASS=$((PASS+1))
    else
        rc=$?
        echo "  FAIL: baseline ${loc} exited ${rc} (expected 0)"
        cat /tmp/formatters_round255_${loc}.log
        FAIL=$((FAIL+1))
    fi
done
echo

# --- Step 2: paired mutations -----------------------------------------------
# Each entry must flip exactly one check from PASS → FAIL → exit 99.
MUTATIONS=(
    "skip_registry"
    "corrupt_roundtrip"
    "no_format_call"
    "skip_steps"
    "cache_miss"
    "constructor_drift"
    "detect_drift"
    "accept_ambiguous"
)

for m in "${MUTATIONS[@]}"; do
    echo "[step 2.${m}] paired-mutation run"
    set +e
    MUTATE="${m}" "${RUNNER_BIN}" -lang=en \
        >/tmp/formatters_round255_mut_${m}.log 2>&1
    rc=$?
    set -e
    if [ "${rc}" = "99" ]; then
        echo "  PASS: mutation '${m}' flipped invariant (exit 99 as expected)"
        PASS=$((PASS+1))
    else
        echo "  FAIL: mutation '${m}' produced exit ${rc} (expected 99)"
        cat /tmp/formatters_round255_mut_${m}.log
        FAIL=$((FAIL+1))
    fi
done
echo

# --- Summary ----------------------------------------------------------------
TOTAL=$((PASS + FAIL))
echo "=== describe Challenge summary: ${PASS}/${TOTAL} passed (${FAIL} failed) ==="
if [ "${FAIL}" -ne 0 ]; then
    exit 1
fi
exit 0
