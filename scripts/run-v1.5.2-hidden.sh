#!/usr/bin/env bash
# Executes the v1.5.2 held-out validation pack outside this repository.
# Hidden fixtures/oracles stay external; only sanitized results are retained.
set -euo pipefail

ROOT="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"
cd "$ROOT"
HIDDEN_ROOT="${TALLY_V152_HIDDEN_ROOT:-}"
if [[ -z "$HIDDEN_ROOT" || ! -d "$HIDDEN_ROOT" ]]; then
  echo "ENV-BLOCKED: TALLY_V152_HIDDEN_ROOT is not configured or does not exist" >&2
  exit 3
fi
ROOT_REAL="$(cd "$ROOT" && pwd -P)"
HIDDEN_REAL="$(cd "$HIDDEN_ROOT" && pwd -P)"
case "$HIDDEN_REAL" in
  "$ROOT_REAL"|"$ROOT_REAL"/*) echo "FAIL: held-out pack must live outside the implementation repository" >&2; exit 2 ;;
esac
RUNNER="$HIDDEN_REAL/run.sh"
MANIFEST="$HIDDEN_REAL/manifest.json"
if [[ ! -f "$RUNNER" || ! -f "$MANIFEST" ]]; then
  echo "ENV-BLOCKED: hidden pack requires run.sh and manifest.json" >&2
  exit 3
fi
if [[ -n "$(git status --porcelain)" ]]; then
  echo "FAIL: hidden validation requires a clean frozen candidate" >&2
  exit 2
fi
REVISION="$(git rev-parse HEAD)"
OUT_ROOT="${TALLY_V152_HIDDEN_OUT:-${TMPDIR:-/tmp}/tally-v1.5.2-hidden-results}"
mkdir -p "$OUT_ROOT"
OUT_ROOT_REAL="$(cd "$OUT_ROOT" && pwd -P)"
case "$OUT_ROOT_REAL" in
  "$ROOT_REAL"|"$ROOT_REAL"/*) echo "FAIL: hidden result directory must live outside the implementation repository" >&2; exit 2 ;;
esac
STAMP="$(date -u +%Y%m%dT%H%M%SZ)"
OUT="$OUT_ROOT_REAL/$STAMP"
mkdir -p "$OUT"
SUMMARY="$OUT/summary.txt"
printf 'revision=%s\nmanifest=%s\nstarted_at=%s\n' "$REVISION" "$MANIFEST" "$(date -u +%Y-%m-%dT%H:%M:%SZ)" >"$SUMMARY"
set +e
bash "$RUNNER" --repo "$ROOT_REAL" --revision "$REVISION" --version v1.5.2 --result-dir "$OUT"
code=$?
set -e
case "$code" in 0) status=PASS ;; 3) status=ENV-BLOCKED ;; *) status=FAIL ;; esac
printf 'status=%s\nexit_code=%s\nfinished_at=%s\n' "$status" "$code" "$(date -u +%Y-%m-%dT%H:%M:%SZ)" >>"$SUMMARY"
echo "$status: v1.5.2 hidden validation; summary=$SUMMARY"
exit "$code"
