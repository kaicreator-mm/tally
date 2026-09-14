# Tally v1.4 — L3 Reference Pack Index

Batch 1 targets Architecture Freeze blockers and the highest-risk closed-loop implementation tasks.

| Task | Pack | Main references |
|---|---|---|
| `A0-02` | `reference-packs/A0-02/README.md` | openai/codex, openai/codex, openai/codex |
| `P0-03` | `reference-packs/P0-03/README.md` | openai/codex, openai/codex, openai/codex |
| `A0-03` | `reference-packs/A0-03/README.md` | WecoAI/SpecBench, JonathanGabor/evilgenie_inspect |
| `A0-04` | `reference-packs/A0-04/README.md` | SWE-agent/SWE-agent, temporalio/sdk-go, WecoAI/SpecBench, JonathanGabor/evilgenie_inspect |
| `P0-01` | `reference-packs/P0-01/README.md` | slsa-framework/slsa, in-toto/attestation |
| `P0-02` | `reference-packs/P0-02/README.md` | openai/codex, openai/codex, slsa-framework/slsa, in-toto/attestation |
| `P2-03` | `reference-packs/P2-03/README.md` | temporalio/sdk-go |
| `P2-05` | `reference-packs/P2-05/README.md` | SWE-agent/SWE-agent, in-toto/attestation |
| `P2-07` | `reference-packs/P2-07/README.md` | SWE-agent/SWE-agent, temporalio/sdk-go |
| `P2-08` | `reference-packs/P2-08/README.md` | SWE-agent/SWE-agent, temporalio/sdk-go, in-toto/attestation |

## Policy

- Local TaskContract / Decision Locks / current Tally source are authoritative.
- External implementations are pinned when repository code is used.
- No whole external repository is copied into this pack.
- Re-run L3 only when a Task is blocked by an uncovered implementation question.
