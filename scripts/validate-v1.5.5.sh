#!/usr/bin/env bash
# Explicit Local / Build Host validation carrier for Tally v1.5.5.
# Intentionally NOT wired to GitHub Actions or any CI system.
set -uo pipefail

ROOT="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"
cd "$ROOT"

OUT_ROOT="${TALLY_V155_VALIDATION_OUT:-$ROOT/.local-validation/v1.5.5}"
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
  echo "Tally v1.5.5 explicit validation"
  echo "revision=$REVISION"
  echo "host=$(hostname 2>/dev/null || printf unknown)"
  echo "go=$GO_VERSION"
  echo "started_at=$(date -u +%Y-%m-%dT%H:%M:%SZ)"
  echo "carrier=Local/Build Host; no GitHub Actions; no CI"
} | tee -a "$LOG"

failures=0
blockers=0
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

record_env_blocked() {
  local name="$1" reason="$2"
  printf 'ENV-BLOCKED(%s)\t%s\t-\n' "$reason" "$name" | tee -a "$SUMMARY"
  echo "ENV-BLOCKED $name: $reason" >>"$LOG"
  blockers=$((blockers+1))
}

# Frozen v1.5.5 concerns.
run replan-domain go test ./internal/replan/...
run engineering-learning go test ./internal/engineeringlearning/...
run intelligence-contract go test ./internal/intelligence/...
run control-plane go test ./internal/controlplane/...
run policy-governance go test ./internal/policygovernance/...

# Cross-slice compatibility with current main's SQLite-first production profile.
# This specifically includes ReplanHistory round-trip/restart coverage.
run sqlite-replan-compatibility go test ./internal/persistence/sqlite/...
run architecture-tests go test ./test/arch/...
run core-server-build go build ./cmd/v15server

# Full visible regression is distinct from Hidden Validation/Release PASS.
run full-regression go test ./...

# Brownfield integration/E2E may require the legacy DATABASE_URL. Its absence is
# an environment blocker, never an implicit PASS.
if [[ -n "${DATABASE_URL:-}" ]]; then
  run integration-e2e go test -tags=integration ./test/e2e/...
else
  record_env_blocked integration-e2e "DATABASE_URL not configured for legacy integration surface"
fi

{
  echo "finished_at=$(date -u +%Y-%m-%dT%H:%M:%SZ)"
  echo "failures=$failures"
  echo "environment_blockers=$blockers"
  echo "summary=$SUMMARY"
} | tee -a "$LOG"

if (( failures > 0 )); then
  exit 1
fi
if (( blockers > 0 )); then
  exit 3
fi
exit 0
