#!/usr/bin/env bash
# P4-03 DeepSeek Harness protocol probe (spike evidence, not production code).
#
# Reproduces the probes recorded in
# docs/tasks/v1.4.1/spikes/P4-03_dsh_protocol_spike.md against a pinned dsh
# version. Reads DEEPSEEK_API_KEY from the environment only; never writes it
# anywhere. Usage:
#
#   DSH_API_KEY_ENV_ONLY=1 bash tools/dsh-spike/probe-headless.sh
#
# Requires: Node >=22.19 (or >=24), network for npx + (probe 6 only) the API.
set -u

DSH_VERSION="${DSH_VERSION:-0.1.5-rc.1}"
DSH="npx -y @deepseek-ai/dsh@${DSH_VERSION}"
WS="$(mktemp -d)"
echo "probe workspace: $WS"
echo "hello.txt" > "$WS/hello.txt"

echo "== 1. version =="
$DSH --version

echo "== 2. headless help =="
$DSH --profile headless --help

echo "== 3. keyless run (expect exit 1, MISSING_CREDENTIAL on stderr) =="
(cd "$WS" && env -u DEEPSEEK_API_KEY timeout 90 $DSH --profile headless \
  "Reply with exactly the word PONG and nothing else.")
echo "keyless exit=$?"

if [ -z "${DEEPSEEK_API_KEY:-}" ]; then
  echo "== 4. credentialed run SKIPPED: DEEPSEEK_API_KEY not set (NOT_RUN) =="
else
  echo "== 4. credentialed run (expect exit 0, stdout 'hello.txt' content) =="
  (cd "$WS" && timeout 120 $DSH --profile headless \
    "Read the file hello.txt in the current working directory and reply with exactly its contents and nothing else.")
  echo "credentialed exit=$?"
fi

echo "== cleanup =="
rm -rf "$WS"
echo done
