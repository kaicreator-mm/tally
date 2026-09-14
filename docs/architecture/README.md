# Tally Architecture

## Current

1. `CURRENT_CODEBASE.md` — current source facts.
2. `V1.4.1_TARGET.md` — current authorized target.
3. `../implementation/V1.4.1_GAP_MAP.md` — source-to-target gaps.
4. approved `adr/**`.
5. `../tasks/v1.4.1/L2_ARCHITECTURE_EVIDENCE.md` — architecture evidence.

## Historical

- `V1.4_TARGET.md` describes the completed v1.4 target and remains historical evidence.

## Current runtime principle

Tally embeds the Harness protocol, not Harness implementations.

`internal/agentruntime.Runtime` remains the single core runtime protocol; Codex and DeepSeek Harness are out-of-process adapters.
