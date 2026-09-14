#!/usr/bin/env bash
# drive-bare-arm.sh — P4-03 arm A (bare) orchestration driver.
#
# This script is trial tooling; it executes ONLY the documented bare-runner
# commands (scripts/v14_trial/bare run|judge|event — P4-02, RUN_PROTOCOL §2).
# It never imports or wires any Tally loop package and it never modifies the
# bare runner, the frozen set, or any evaluation asset.
#
# Procedure per task (RUN_PROTOCOL §2):
#   attempt 1: bare run (provisions a fresh disposable workspace), then the
#              arm-neutral judge on that attempt's workspace;
#   if the judge does not accept and attempts remain, the operator journals a
#   prompt_reissue event (the human re-issues the same verbatim intent), the
#   attempt-1 workspace state is continued (byte-identical copy into the next
#   attempt's workspace directory — "the human continues where they left
#   off"), bare run --attempt N, and the judge again.
#   Max 3 attempts = first attempt + 2 retries (the same bounded shape the
#   Tally arm's repair budget gives arm B).
#
# The final judge verdict for a task is the one recorded in its LAST attempt
# directory (judge-result.json).
set -u

REPO="${REPO:-C:/Dev/Code/tally}"
CODEX_BIN="${CODEX_BIN:-C:/Users/15549/AppData/Local/OpenAI/Codex/bin/9ba750cce02d5e5c/codex.exe}"
MODEL="${MODEL:-deepseek-v4-flash}"
SANDBOX="${SANDBOX:-danger-full-access}"
TIMEOUT="${TIMEOUT:-30m}"
BARE_ROOT="${BARE_ROOT:-$TMP/tally-v14-trial/bare}"
OWNER_HOME="${OWNER_HOME:-$TMP/tally-v14-trial/bare-owner-home}"
PSQLSH="${PSQLSH:-$TMP/tally-v14-trial/tally/tools/psqlsh.exe}"
PG_URI="${PG_URI:-postgres://postgres:test@localhost:5433}"
AUTH_FILE="${AUTH_FILE:-$REPO/.e2e/deepseek-auth.json}"
MAX_ATTEMPTS="${MAX_ATTEMPTS:-3}"

TASKS="${TASKS:-bf-01-token-bucket-offbyone bf-02-logout-session-invalidation bf-03-pagination-partial-page bf-04-date-validation-calendar bf-05-currency-rounding-halfup bf-06-email-dedupe-casefold db-01-orders-total-backfill db-02-account-email-unique db-03-rename-breaks-downstream db-04-orders-fk-orphan db-05-drop-column-data-loss db-06-total-type-narrowing}"

mkdir -p "$OWNER_HOME"
if [ ! -f "$OWNER_HOME/auth.json" ]; then
  cp "$AUTH_FILE" "$OWNER_HOME/auth.json" && chmod 600 "$OWNER_HOME/auth.json"
fi

RUN_FLAGS=(--codex-bin "$CODEX_BIN" --model "$MODEL" --sandbox "$SANDBOX" --timeout "$TIMEOUT" \
  --codex-home "$OWNER_HOME" --provider-id custom --provider-name deepseek \
  --provider-base-url https://api.deepseek.com --provider-wire-api responses \
  --provider-requires-openai-auth --runs-root "$BARE_ROOT")

JUDGE_FLAGS=(--pg "$PG_URI" --psql-bin "$PSQLSH" --runs-root "$BARE_ROOT")

accepted() { # task attempt -> 0 if judge accepted
  grep -q '"accepted": true' "$BARE_ROOT/$1/attempt-$2/judge-result.json" 2>/dev/null
}

for task in $TASKS; do
  echo "=== bare arm: $task ==="
  for att in $(seq 1 "$MAX_ATTEMPTS"); do
    if [ "$att" -gt 1 ]; then
      prev=$((att - 1))
      if [ ! -d "$BARE_ROOT/$task/attempt-$prev/ws" ]; then
        echo "$task: attempt-$prev workspace missing; stopping at attempt $prev" >&2
        break
      fi
      go run "$REPO/scripts/v14_trial/bare" event --task "$task" --attempt "$att" --type prompt_reissue \
        --note "operator re-issued the same verbatim frozen intent after the previous attempt was not accepted; workspace state continued from attempt $prev" \
        || exit 1
      rm -rf "$BARE_ROOT/$task/attempt-$att/ws"
      mkdir -p "$BARE_ROOT/$task/attempt-$att"
      cp -R "$BARE_ROOT/$task/attempt-$prev/ws" "$BARE_ROOT/$task/attempt-$att/ws"
      echo "$task: attempt $att (continued workspace)"
    fi
    go run "$REPO/scripts/v14_trial/bare" run --task "$task" --attempt "$att" "${RUN_FLAGS[@]}" || { echo "$task: run failed" >&2; break; }
    go run "$REPO/scripts/v14_trial/bare" judge --task "$task" --attempt "$att" "${JUDGE_FLAGS[@]}" || { echo "$task: judge failed" >&2; break; }
    if accepted "$task" "$att"; then
      echo "$task: ACCEPTED at attempt $att"
      break
    fi
    echo "$task: not accepted at attempt $att"
  done
done
echo "bare arm: all tasks processed"
