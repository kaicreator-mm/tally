# External Research Evidence

External references are study material. Local TaskContract/architecture remains authoritative.

## REF-CODEX-INSTR — openai/codex
- Pin: `553df1c691fe8bf7747e50da22f1342984495ae0`
- License: `Apache-2.0`
- URL: https://github.com/openai/codex/blob/553df1c691fe8bf7747e50da22f1342984495ae0/codex-rs/codex-home/src/instructions/mod.rs
- Relevant paths:
  - `codex-rs/codex-home/src/instructions/mod.rs`
- Why relevant: Codex loads global AGENTS.md / AGENTS.override.md from CODEX_HOME; CODEX_HOME is observable instruction state, not merely a cache directory.
- Copy policy: study behavior/tests/contracts; do not paste external implementation unless license/attribution is rechecked immediately before copying.

## REF-CODEX-AUTH — openai/codex
- Pin: `553df1c691fe8bf7747e50da22f1342984495ae0`
- License: `Apache-2.0`
- URL: https://github.com/openai/codex/blob/553df1c691fe8bf7747e50da22f1342984495ae0/codex-rs/login/src/auth/storage.rs
- Relevant paths:
  - `codex-rs/login/src/auth/storage.rs`
- Why relevant: File auth lives at CODEX_HOME/auth.json; keyring storage derives a key from the codex_home path, so per-run CODEX_HOME can change authentication behavior.
- Copy policy: study behavior/tests/contracts; do not paste external implementation unless license/attribution is rechecked immediately before copying.

## REF-CODEX-ROLLOUT — openai/codex
- Pin: `553df1c691fe8bf7747e50da22f1342984495ae0`
- License: `Apache-2.0`
- URL: https://github.com/openai/codex/blob/553df1c691fe8bf7747e50da22f1342984495ae0/codex-rs/rollout/src/lib.rs
- Relevant paths:
  - `codex-rs/rollout/src/lib.rs`
  - `codex-rs/rollout/src/metadata.rs`
- Why relevant: Active and archived session/rollout data are rooted under CODEX_HOME/sessions and CODEX_HOME/archived_sessions.
- Copy policy: study behavior/tests/contracts; do not paste external implementation unless license/attribution is rechecked immediately before copying.
