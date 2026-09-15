#!/usr/bin/env bash
# Explicit Local / Build Host validation carrier for Tally v1.5.2.
# Intentionally NOT wired to GitHub Actions or any CI system.
set -uo pipefail

ROOT="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"
cd "$ROOT"
OUT_ROOT="${TALLY_V152_VALIDATION_OUT:-$ROOT/.local-validation/v1.5.2}"
STAMP="$(date -u +%Y%m%dT%H%M%SZ)"
OUT="$OUT_ROOT/$STAMP"
mkdir -p "$OUT"
LOG="$OUT/validation.log"
SUMMARY="$OUT/summary.tsv"
: >"$LOG"
printf 'status\tname\tcommand\n' >"$SUMMARY"

REVISION="$(git rev-parse HEAD 2>/dev/null || printf unknown)"
{
  echo "Tally v1.5.2 explicit validation"
  echo "revision=$REVISION"
  echo "host=$(hostname 2>/dev/null || printf unknown)"
  echo "go=$(go version 2>/dev/null || printf unavailable)"
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
  blockers=$((blockers+1))
}

# Focused slices first. These are intentionally explicit so a failure points to
# the environment seam before the full regression obscures it.
run focused-runner go test ./internal/runner/...
run focused-taskexecution go test ./internal/taskexecution/...
run focused-controlplane go test ./internal/controlplane/...
run focused-dsh-runtime go test ./runtimes/dsh/...
run focused-v15server-composition go test ./cmd/v15server/...
run architecture-tests go test ./test/arch/...
run full-regression go test ./...

# Transport-level external probes. These prove the selected environment can
# resolve the pinned workspace and bind a generic EnvironmentRuntime. They do
# not pretend to be a production Agent claim.
if [[ -n "${TALLY_V152_SSH_HOST:-}" && -n "${TALLY_V152_SSH_USER:-}" && -n "${TALLY_V152_SSH_WORKSPACE:-}" && -n "${TALLY_V152_SSH_BASE_REVISION:-}" && ( -n "${TALLY_V152_SSH_KEY_FILE:-}" || -n "${TALLY_V152_SSH_PASSWORD:-}" ) ]]; then
  run real-ssh-transport go test ./internal/taskexecution -run '^TestV152ExternalSSHEnvironment$' -count=1
else
  record_env_blocked real-ssh-transport "SSH target/user/workspace/revision/auth not configured"
fi

if command -v docker >/dev/null 2>&1 && docker info >/dev/null 2>&1 && [[ -n "${TALLY_V152_DOCKER_IMAGE:-}" && -n "${TALLY_V152_DOCKER_WORKSPACE_SOURCE:-}" && -n "${TALLY_V152_DOCKER_BASE_REVISION:-}" ]]; then
  run real-docker-transport go test ./internal/taskexecution -run '^TestV152ExternalDockerEnvironment$' -count=1
else
  record_env_blocked real-docker-transport "Docker daemon/profile/workspace/revision not configured"
fi

{
  echo "finished_at=$(date -u +%Y-%m-%dT%H:%M:%SZ)"
  echo "failures=$failures"
  echo "environment_blockers=$blockers"
  echo "summary=$SUMMARY"
} | tee -a "$LOG"

if (( failures > 0 )); then exit 1; fi
if (( blockers > 0 )); then exit 3; fi
exit 0
