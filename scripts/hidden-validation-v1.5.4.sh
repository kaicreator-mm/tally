#!/usr/bin/env bash
set -euo pipefail

# Hidden Validation for v1.5.4. Local/Build-Host only; never a CI gate.
# Covers: reviewer PASS/finding, repair/failure, verifier disagreement,
# unavailable execution, independence violation, parallel collision,
# dependency race, restart/cancel, evidence provenance and concurrency bounds.

echo '== v1.5.4 hidden validation: role lifecycle and negative paths =='
go test ./internal/multiagent -count=1

echo '== v1.5.4 hidden validation: control-plane persistence/restart/acceptance =='
go test ./internal/controlplane -run 'MultiAgent|RestartQueuedMultiAgent' -count=1

echo '== v1.5.4 hidden validation: runtime identity cannot be spoofed =='
go test ./internal/taskexecution -run 'RuntimeExecutorRejects.*RuntimeReference' -count=1

echo 'PASS: v1.5.4 hidden validation'
