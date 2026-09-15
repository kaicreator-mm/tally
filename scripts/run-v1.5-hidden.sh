#!/usr/bin/env bash
# Executes a v1.5.0 held-out validation pack that lives OUTSIDE this repository.
# The pack owns hidden fixtures/oracles. This wrapper only pins revision,
# enforces separation, and records the sanitized result location.
set -euo pipefail

ROOT="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"
cd "$ROOT"

HIDDEN_ROOT="${TALLY_V15_HIDDEN_ROOT:-}"
if [[ -z "$HIDDEN_ROOT" ]]; then
  echo "ENV-BLOCKED: TALLY_V15_HIDDEN_ROOT is not configured" >&2
  exit 3
fi
if [[ ! -d "$HIDDEN_ROOT" ]]; then
  echo "ENV-BLOCKED: hidden root does not exist: $HIDDEN_ROOT" >&2
  exit 3
fi

ROOT_REAL="$(cd "$ROOT" && pwd -P)"
HIDDEN_REAL="$(cd "$HIDDEN_ROOT" && pwd -P)"
case "$HIDDEN_REAL" in
  "$ROOT_REAL"|"$ROOT_REAL"/*)
    echo "FAIL: held-out pack must live outside the implementation repository" >&2
    exit 2
    ;;
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

OUT_ROOT="${TALLY_V15_HIDDEN_OUT:-$ROOT/.local-validation/v1.5.0-hidden}"
STAMP="$(date -u +%Y%m%dT%H%M%SZ)"
OUT="$OUT_ROOT/$STAMP"
mkdir -p "$OUT"
SUMMARY="$OUT/summary.txt"

{
  echo "revision=$REVISION"
  echo "hidden_pack=$HIDDEN_REAL"
  echo "manifest=$MANIFEST"
  echo "started_at=$(date -u +%Y-%m-%dT%H:%M:%SZ)"
  echo "carrier=Local/Build Host; external held-out pack; no GitHub Actions; no CI"
} >"$SUMMARY"

# The external pack must verify that its manifest is compatible with this
# revision and must not write hidden fixture/oracle content into the repo.
set +e
bash "$RUNNER" --repo "$ROOT_REAL" --revision "$REVISION" --result-dir "$OUT"
code=$?
set -e

case "$code" in
  0)
    status="PASS"
    ;;
  3)
    status="ENV-BLOCKED"
    ;;
  *)
    status="FAIL"
    ;;
esac
{
  echo "status=$status"
  echo "exit_code=$code"
  echo "finished_at=$(date -u +%Y-%m-%dT%H:%M:%SZ)"
} >>"$SUMMARY"

echo "$status: v1.5.0 hidden validation; summary=$SUMMARY"
exit "$code"
