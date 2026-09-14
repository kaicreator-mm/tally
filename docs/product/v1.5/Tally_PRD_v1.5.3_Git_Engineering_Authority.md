# Tally PRD v1.5.3 — Git Engineering Authority

- Status: **FROZEN EXECUTABLE SCOPE**
- Implementation dependency: **v1.5.0 EngineeringNode / CompletionContract / Authority contracts only**
- Release baseline dependency: v1.5.2
- Parallel lane: may be implemented concurrently with v1.5.1, v1.5.2, v1.5.4 and v1.5.5.

## Goal

Make Git/GitHub lifecycle operations formal, traceable Engineering Harness actions linked to engineering nodes, contracts, evidence and authority.

## In scope

1. Git Engineering Artifacts: Issue, Branch, Commit, Pull Request, Tag and Release.
2. Git Engineering Actions: create/update/close issue, create branch, commit, push, open/update/review/approve/request-changes/merge PR, create tag and publish release.
3. `GitActionRecord` links Project, PlanningRevision, TaskNode, TaskContract, Run, ChangeSet, Verification, Review, Acceptance and Release where applicable.
4. Git Authority controls when actions are permitted and whether human authority is required.
5. v1.5.0 CompletionContract may require specific Git actions.
6. Task state distinguishes code-complete/accepted/awaiting-merge/done where merge is required.
7. Merge result revision becomes explicit downstream provenance.
8. Tag/Release must point to the accepted release baseline.
9. GitHub is the first provider without making GitHub semantics the domain model.

## Contract ownership

This version extends v1.5.0 node/action/authority abstractions with Git-specific adapters and records. It must not create a second workflow/state authority.

## Canonical task trace

```text
TaskNode → Frozen TaskContract → Issue → Branch/Workspace → Agent Run
→ Commit/Push → Pull Request → Verification → Independent Review
→ Acceptance → Merge → Resulting Revision → Task DONE
```

## Out of scope

Arbitrary SCM providers beyond the extension boundary, repository administration, organization-wide policy management and unrelated release-distribution systems.

## Acceptance criteria

- Every material Git action is attributable to engineering node and actor/authority.
- Policy can block PR creation/merge/tag/release.
- Task cannot be DONE when CompletionContract requires an unperformed merge.
- Verification/review evidence links to PR/accepted revision.
- Merge result revision updates downstream provenance.
- Release tag cannot qualify when pointing to a non-accepted baseline.

## Hidden Validation

Wrong base branch, stale head, push failure, missing PR, merge without verification, unauthorized merge, request-changes→repair→re-review, tag mismatch and release provenance mismatch.

## Release qualification

Issue/PR/Merge/Tag/Release traceability PASS; Git Authority PASS; CompletionContract Git requirements PASS; accepted baseline provenance PASS.