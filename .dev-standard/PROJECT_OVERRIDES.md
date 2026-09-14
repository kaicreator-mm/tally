# Tally Project Overrides

## v1.5.0 verification carrier

For Tally v1.5.0, **do not use GitHub Actions and do not use any CI system**.

This explicit project requirement overrides generic workflow text that names CI as the validation carrier. It does not waive or weaken any test, review, closure, hidden-validation, critical-journey, packaging or release-qualification requirement.

Rules:
- Do not create, modify, enable, trigger, wait for or depend on `.github/workflows/**` for v1.5.0.
- Do not substitute hosted CI, self-hosted CI or another continuous-integration service.
- GitHub is used for repository state, Issues, branches, commits, PRs, reviews, merges, tags/releases and provenance.
- Focused tests, integration tests, full regression, Hidden Validation, Critical Journeys and packaging are executed explicitly in the developer/agent environment or a designated Local/Build Host.
- Verification evidence records environment, revision, command, result and relevant failure output.
- Status vocabulary: `PASS`, `FAIL`, `NOT_RUN(reason)`, `ENV-BLOCKED(reason/evidence)`.
- An unexecuted command is never PASS. Release Qualification is never inferred from PR merge.

Refs: #5
