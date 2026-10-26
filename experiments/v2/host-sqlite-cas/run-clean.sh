#!/usr/bin/env bash
# Tally v2 Issue #100: isolated public DH Host-node SQLite experiment.
# This script runs real packages; success requires process EXIT=0, not a source review.
set -Eeuo pipefail

work=''
cleanup() {
  rc=$?
  printf 'WCX_EXPERIMENT_EXIT=%s\n' "$rc"
  if [ -n "$work" ]; then
    if [ "${KEEP_WCX_WORK:-0}" = '1' ]; then
      printf 'WCX_WORK_DIR=%s\n' "$work"
    else
      rm -rf -- "$work"
    fi
  fi
}
trap cleanup EXIT

DH=${DH:?Set DH to clean DomainHarness v0.10.2 exact candidate checkout}
TALLY=${TALLY:?Set TALLY to the Tally PR #101 exact checkout}
DH=$(cd "$DH" && pwd -P)
TALLY=$(cd "$TALLY" && pwd -P)
expected_dh='11ab7954f547fde44c003e2b227d672dc8af0a79'
test "$(git -C "$DH" rev-parse HEAD)" = "$expected_dh" || { echo 'DH_SHA_MISMATCH' >&2; exit 2; }
git -C "$TALLY" rev-parse --verify HEAD
test "$(node -p 'process.versions.node.split(".")[0]')" = '24' || { echo 'NODE24_REQUIRED' >&2; exit 2; }
test -f "$TALLY/experiments/v2/host-sqlite-cas/host-cas.test.mjs"
work=$(mktemp -d)
pack="$work/tarballs"
consumer="$work/consumer"
mkdir -p "$pack" "$consumer"
printf 'DH_SHA=%s\nTALLY_SHA=%s\n' "$(git -C "$DH" rev-parse HEAD)" "$(git -C "$TALLY" rev-parse HEAD)"
printf 'NODE=%s\nNPM=%s\n' "$(node --version)" "$(npm --version)"
cd "$DH"
npm ci
npm run build
npm pack --pack-destination "$pack" -w @kaicreator/domain-harness-contracts
npm pack --pack-destination "$pack" -w @kaicreator/domain-harness-microkernel
npm pack --pack-destination "$pack" -w @kaicreator/domain-harness-host-node
tarballs=(
  "$pack"/kaicreator-domain-harness-contracts-*.tgz
  "$pack"/kaicreator-domain-harness-microkernel-*.tgz
  "$pack"/kaicreator-domain-harness-host-node-*.tgz
)
for file in "${tarballs[@]}"; do
  test -f "$file" || { printf 'MISSING_TARBALL=%s\n' "$file" >&2; exit 3; }
done
printf 'TARBALL_SHA256\n'
sha256sum "${tarballs[@]}"
cp "$TALLY/experiments/v2/host-sqlite-cas/host-cas.test.mjs" "$consumer/"
cp "$TALLY/experiments/v2/host-sqlite-cas/package.json" "$consumer/"
cp "$TALLY/experiments/v2/host-sqlite-cas/wal-open-child.mjs" "$consumer/"
cp "$TALLY/experiments/v2/host-sqlite-cas/wal-concurrent-open.test.mjs" "$consumer/"
cd "$consumer"
npm install --save-exact "${tarballs[@]}" 'better-sqlite3@^12'
npm ls --depth=0
node --test host-cas.test.mjs
if [ "${WCX_WAL_STRESS:-0}" = "1" ]; then
  printf "WCX_WAL_STRESS=ENABLED; rounds=%s; child_processes_per_round=4\n" "${WCX_WAL_ROUNDS:-5}"
  node --test wal-concurrent-open.test.mjs
else
  printf "WCX_WAL_STRESS=NOT_RUN (set WCX_WAL_STRESS=1 to opt in)\n"
fi
