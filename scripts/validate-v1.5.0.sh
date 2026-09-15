#!/usr/bin/env bash
# Explicit Local / Build Host validation carrier for Tally v1.5.0.
# This script is intentionally NOT wired to GitHub Actions or any CI system.
set -uo pipefail

ROOT="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"
cd "$ROOT"

OUT_ROOT="${TALLY_V15_VALIDATION_OUT:-$ROOT/.local-validation/v1.5.0}"
STAMP="$(date -u +%Y%m%dT%H%M%SZ)"
OUT="$OUT_ROOT/$STAMP"
mkdir -p "$OUT"
LOG="$OUT/validation.log"
SUMMARY="$OUT/summary.tsv"
: >"$LOG"
printf 'status\tname\tcommand\n' >"$SUMMARY"

REVISION="$(git rev-parse HEAD 2>/dev/null || printf unknown)"
GO_VERSION="$(go version 2>/dev/null || printf unavailable)"
{
  echo "Tally v1.5.0 explicit validation"
  echo "revision=$REVISION"
  echo "host=$(hostname 2>/dev/null || printf unknown)"
  echo "go=$GO_VERSION"
  echo "started_at=$(date -u +%Y-%m-%dT%H:%M:%SZ)"
  echo "carrier=Local/Build Host; no GitHub Actions; no CI"
} | tee -a "$LOG"

failures=0
run() {
  local name="$1"; shift
  local printable
  printable="$(printf '%q ' "$@")"
  echo >>"$LOG"
  echo "=== $name ===" | tee -a "$LOG"
  echo "+ $printable" | tee -a "$LOG"
  if "$@" >>"$LOG" 2>&1; then
    printf 'PASS\t%s\t%s\n' "$name" "$printable" | tee -a "$SUMMARY"
  else
    printf 'FAIL\t%s\t%s\n' "$name" "$printable" | tee -a "$SUMMARY"
    failures=$((failures+1))
  fi
}

record_not_run() {
  local name="$1" reason="$2"
  printf 'NOT_RUN(%s)\t%s\t-\n' "$reason" "$name" | tee -a "$SUMMARY"
  echo "NOT_RUN $name: $reason" >>"$LOG"
}

# Focused v1.5 domain/contracts/control-plane slices.
run engineering-domain go test ./internal/engineering/...
run intelligence-execution go test ./internal/intelligence/... ./internal/taskexecution/...
run control-plane go test ./internal/controlplane/...

# Brownfield authorities that v1.5 reuses rather than replaces.
run brownfield-safety go test ./internal/changeset/... ./internal/workspace/... ./internal/closedloop/... ./internal/workflow/...
run runtime-adapters go test ./runtimes/...
run runtime-conformance go test ./test/runtimeconformance/...
run architecture-tests go test ./test/arch/...

# Static/compiler quality gates.
run build go build ./...
run vet go vet ./...
run race-focused go test -race ./internal/engineering/... ./internal/intelligence/... ./internal/taskexecution/... ./internal/controlplane/...

# Full visible regression. This is distinct from Release PASS and from Hidden Validation.
run full-regression go test ./...

# Integration/E2E requires an explicit database/environment. Absence is never PASS.
if [[ -n "${DATABASE_URL:-}" ]]; then
  run integration-e2e go test -tags=integration ./test/e2e/...
else
  record_not_run integration-e2e "DATABASE_URL not configured"
fi

{
  echo "finished_at=$(date -u +%Y-%m-%dT%H:%M:%SZ)"
  echo "failures=$failures"
  echo "summary=$SUMMARY"
} | tee -a "$LOG"

if (( failures > 0 )); then
  exit 1
fi
exit 0
