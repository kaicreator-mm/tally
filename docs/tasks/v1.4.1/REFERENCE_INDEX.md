# L2 External Reference Index

## Standards / official docs

- Git worktree: https://git-scm.com/docs/git-worktree
- JSON Schema 2020-12: https://json-schema.org/draft/2020-12
- GitHub Repository Rulesets / Rule Insights / Evaluate mode:
  https://docs.github.com/en/repositories/configuring-branches-and-merges-in-your-repository/managing-rulesets/managing-rulesets-for-a-repository
- GitLab Merge Request Approval Rules:
  https://docs.gitlab.com/user/project/merge_requests/approvals/rules/
- GitLab Merge Request Approval Policies:
  https://docs.gitlab.com/user/application_security/policies/merge_request_approval_policies/

## Codex implementation/failure evidence

- spawn_agent source:
  https://github.com/openai/codex/blob/main/codex-rs/core/src/tools/handlers/multi_agents_spec.rs
- native subagent model_provider override issue:
  https://github.com/openai/codex/issues/40858
- Multi-Agent V2 external-provider agent_message issue:
  https://github.com/openai/codex/issues/33551

## DeepSeek Harness

- Safety:
  https://github.com/deepseek-ai/deepseek-harness/blob/master/SAFETY.md
- Subagent seam:
  https://github.com/deepseek-ai/deepseek-harness/blob/master/docs/subsystems/subagent.md
- Model/provider discovery:
  https://github.com/deepseek-ai/deepseek-harness/blob/master/packages/subagent/tool-subagent/src/list-models.ts
- Tool-subagent:
  https://github.com/deepseek-ai/deepseek-harness/blob/master/packages/subagent/tool-subagent/src/index.ts
- Codex / Claude Code subagent backends:
  https://github.com/deepseek-ai/deepseek-harness/blob/master/.agents/notes/implemented/feature/2026-08-04-claude-code-and-codex-subagent-backends.md

## How to use references

These references justify architecture patterns and constraints. They are not instructions to copy implementations blindly.

Task-level code reuse/reference extraction belongs to L3.
