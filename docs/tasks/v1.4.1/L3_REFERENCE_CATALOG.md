# L3 External Reference Catalog

This catalog is intentionally small and reused across tasks. External sources provide implementation patterns; Tally source remains authoritative.

## EXT-JSONSCHEMA — santhosh-tekuri/jsonschema — bounded declarative validation reference

- URL: https://github.com/santhosh-tekuri/jsonschema
- License: Apache-2.0
- Version/pin guidance: current repository; use as pattern only
- Use in Tally: Reference for strict declarative shape validation. Do NOT add this dependency unless dependency whitelist/ADR explicitly approves it.

## EXT-GO-HTML — Go html/template

- URL: https://pkg.go.dev/html/template
- License: BSD-3-Clause (Go project)
- Version/pin guidance: Go standard library
- Use in Tally: Context-aware escaping reference for rendering LLM-originated values as data, never raw HTML.

## EXT-GO-EXEC — Go os/exec

- URL: https://pkg.go.dev/os/exec
- License: BSD-3-Clause (Go project)
- Version/pin guidance: Go standard library
- Use in Tally: Subprocess context cancellation, Wait/Process lifecycle and command execution semantics.

## EXT-GIT-WORKTREE — Git worktree documentation

- URL: https://git-scm.com/docs/git-worktree
- License: Documentation/reference only
- Version/pin guidance: current Git manual
- Use in Tally: Disposable linked/detached worktree semantics; no code copying required.

## EXT-GIT-MERGE-TREE — Git merge-tree documentation

- URL: https://git-scm.com/docs/git-merge-tree
- License: Documentation/reference only
- Version/pin guidance: current Git manual
- Use in Tally: Preflight merge/conflict evaluation without touching working tree/index; inspect exit status and conflict records.

## EXT-GIT-MERGE — Git merge documentation

- URL: https://git-scm.com/docs/git-merge
- License: Documentation/reference only
- Version/pin guidance: current Git manual
- Use in Tally: Authoritative merge/abort/conflict behavior.

## EXT-GIT-UPDATE-REF — Git update-ref documentation

- URL: https://git-scm.com/docs/git-update-ref
- License: Documentation/reference only
- Version/pin guidance: current Git manual
- Use in Tally: Compare-and-swap style ref update semantics using expected old object IDs.

## EXT-GITHUB-RULESETS — GitHub Repository Rulesets — evaluate/rule insights pattern

- URL: https://docs.github.com/en/repositories/configuring-branches-and-merges-in-your-repository/managing-rulesets/managing-rulesets-for-a-repository
- License: Documentation/reference only
- Version/pin guidance: current GitHub Docs
- Use in Tally: Pattern for rule proposal/evaluate-before-enforce and inspecting affected historical/current actions.

## EXT-GITLAB-APPROVAL — GitLab Merge Request Approval Policies

- URL: https://docs.gitlab.com/user/application_security/policies/merge_request_approval_policies/
- License: Documentation/reference only
- Version/pin guidance: current GitLab Docs
- Use in Tally: Pattern for condition-based approval, fail-safe behavior when required evidence/scans are unavailable.

## EXT-CODEX-MULTI — OpenAI Codex spawn_agent implementation

- URL: https://github.com/openai/codex/blob/main/codex-rs/core/src/tools/handlers/multi_agents_spec.rs
- License: Apache-2.0
- Version/pin guidance: main; pin inspected commit before implementation
- Use in Tally: Reference current child model/reasoning surface; do not assume arbitrary provider routing.

## EXT-CODEX-CONFIG — OpenAI Codex runtime config

- URL: https://github.com/openai/codex/blob/main/codex-rs/core/src/config/mod.rs
- License: Apache-2.0
- Version/pin guidance: main; pin inspected commit
- Use in Tally: Reference current provider map, multi-agent enablement, subagent model/thread config.

## EXT-DSH-SAFETY — DeepSeek Harness SAFETY

- URL: https://github.com/deepseek-ai/deepseek-harness/blob/master/SAFETY.md
- License: MIT repository
- Version/pin guidance: master; developer preview
- Use in Tally: Hard constraint: do not treat DSH as sole security control; least privilege and disposable isolation.

## EXT-DSH-SUBAGENT — DeepSeek Harness subagent seam

- URL: https://github.com/deepseek-ai/deepseek-harness/blob/master/docs/subsystems/subagent.md
- License: MIT
- Version/pin guidance: master; pin commit
- Use in Tally: Named provider registry, capability preflight, child lifecycle and result semantics.

## EXT-DSH-TOOL — DeepSeek Harness tool-subagent implementation

- URL: https://github.com/deepseek-ai/deepseek-harness/blob/master/packages/subagent/tool-subagent/src/index.ts
- License: MIT
- Version/pin guidance: master; pin commit
- Use in Tally: Current provider/model/reasoning selection surface. Must verify exact installed build because developer-preview behavior has changed.

## EXT-DSH-CODEX — DeepSeek Harness Codex/Claude Code backend design note

- URL: https://github.com/deepseek-ai/deepseek-harness/blob/master/.agents/notes/implemented/feature/2026-08-04-claude-code-and-codex-subagent-backends.md
- License: MIT
- Version/pin guidance: master
- Use in Tally: Reference ownership boundaries for task text, cwd, cancellation, result settlement and process teardown.

## EXT-POSTGRES-INSERT — PostgreSQL INSERT / ON CONFLICT

- URL: https://www.postgresql.org/docs/current/sql-insert.html
- License: PostgreSQL documentation
- Version/pin guidance: current PostgreSQL docs
- Use in Tally: Reference idempotent insert patterns where a unique semantic key is appropriate; do not mutate append-only events.

## EXT-UBER-GO-TEST — Uber Go Style Guide — table-driven tests

- URL: https://github.com/uber-go/guide/blob/master/style.md
- License: CC BY 3.0 / repository terms; pattern only
- Version/pin guidance: master
- Use in Tally: Reference table-driven test structure for deterministic validation matrices.
