#!/usr/bin/env bash
set -euo pipefail

# Tally v1.5.4 focused validation. Local/Build-Host only: no GitHub Actions/CI.

echo '== v1.5.4 focused: multi-agent core =='
go test ./internal/multiagent -count=1

echo '== v1.5.4 focused: control-plane integration =='
go test ./internal/controlplane -run 'MultiAgent|RestartQueuedMultiAgent' -count=1

echo '== v1.5.4 focused: truthful runtime binding =='
go test ./internal/taskexecution -run 'RuntimeExecutor|RuntimeReference' -count=1

echo '== v1.5.4 focused: production composition =='
go test ./cmd/server -run 'V15.*MultiAgent|V15MultiAgent' -count=1

echo '== v1.5.4 focused: production binaries compile =='
go build ./cmd/server ./cmd/v15server

echo 'PASS: v1.5.4 focused validation'
