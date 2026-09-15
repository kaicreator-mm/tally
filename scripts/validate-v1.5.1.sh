#!/usr/bin/env bash
# Explicit Local / Build Host validation carrier for Tally v1.5.1.
# Intentionally NOT wired to GitHub Actions or any CI system.
set -uo pipefail

ROOT="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"
cd "$ROOT"

OUT_ROOT="${TALLY_V151_VALIDATION_OUT:-$ROOT/.local-validation/v1.5.1}"
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
  echo "Tally v1.5.1 explicit validation"
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

# v1.5.1 focused persistence/scheduling/production composition.
run persistence go test ./internal/persistence/...
run control-plane go test ./internal/controlplane/...
run architecture-tests go test ./test/arch/...
# The binary goes to the ignored bin/ so the build never dirties the worktree
# (export candidate-identity checks refuse HEAD exports from a dirty tree).
run core-server-build go build -o bin/v15server ./cmd/v15server
run race-focused go test -race ./internal/persistence/... ./internal/controlplane/...

# Full visible regression remains distinct from Hidden Validation and Release PASS.
run full-regression go test ./...

# PostgreSQL is optional at runtime but adapter parity is required release evidence.
if [[ -n "${TALLY_TEST_POSTGRES_URL:-}" ]]; then
  run postgres-conformance go test ./internal/persistence/postgres/...
else
  record_env_blocked postgres-conformance "TALLY_TEST_POSTGRES_URL not configured"
fi

# Brownfield integration/E2E remains a separate compatibility surface. It may
# require the legacy DATABASE_URL even though v1.5.1 Core does not.
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
