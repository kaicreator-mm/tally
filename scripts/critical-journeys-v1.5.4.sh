#!/usr/bin/env bash
set -euo pipefail

# Critical Journeys for v1.5.4. Local/Build-Host only: no GitHub Actions/CI.

echo '== CJ1: implement -> review -> verify -> accept =='
go test ./internal/multiagent -run '^TestEngineSequentialHappyPath$' -count=1
go test ./internal/controlplane -run '^TestMultiAgentDispatchPersistsWorkflowAndUsesCompletionContract$' -count=1

echo '== CJ2: review finding -> repair -> review -> verify =='
go test ./internal/multiagent -run '^TestEngineReviewerRepairLoop$|^TestEngineVerifierDisagreementReturnsToRepair$' -count=1

echo '== CJ3: bounded parallel collaboration without collisions =='
go test ./internal/multiagent -run '^TestEngineEnforcesGlobalConcurrencyLimit$|Scheduler' -count=1
go test ./internal/controlplane -run 'Parallel|Conflict' -count=1

echo '== CJ4: restart/cancel keeps semantic state recoverable =='
go test ./internal/multiagent -run 'RecoversInFlightReviewer|CancelCollaborationQueued' -count=1
go test ./internal/controlplane -run '^TestRestartQueuedMultiAgentKeepsTaskReadyAndPersistsInterruptedCollaboration$' -count=1

echo '== CJ5: production server composes existing runtime executor =='
go test ./cmd/server -run '^TestV15ProductionCompositionEnablesMultiAgentOnlyWithExecutionRuntime$' -count=1

echo 'PASS: v1.5.4 critical journeys'
