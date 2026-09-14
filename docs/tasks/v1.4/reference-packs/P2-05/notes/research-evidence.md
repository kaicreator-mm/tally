# External Research Evidence

External references are study material. Local TaskContract/architecture remains authoritative.

## REF-SWE-RETRY — SWE-agent/SWE-agent
- Pin: `3ea751c087f32b16e039a2233dd6eefecef325d5`
- License: `MIT`
- URL: https://github.com/SWE-agent/SWE-agent/blob/3ea751c087f32b16e039a2233dd6eefecef325d5/sweagent/agent/reviewer.py
- Relevant paths:
  - `sweagent/agent/reviewer.py`
  - `sweagent/agent/agents.py`
- Why relevant: Defines an explicit outer Review/Retry loop with max_attempts and cost limits, distinct from the inner agent step loop; each attempt is finalized before deciding to start the next.
- Copy policy: study behavior/tests/contracts; do not paste external implementation unless license/attribution is rechecked immediately before copying.

## REF-INTOTO — in-toto/attestation
- Pin: `Attestation Framework v1.2`
- License: `Apache-2.0`
- URL: https://github.com/in-toto/attestation/blob/main/spec/v1/statement.md
- Relevant paths:
  - `spec/v1/statement.md`
  - `spec/predicates/link.md`
  - `spec/predicates/test-result.md`
- Why relevant: Binds claims to immutable subjects by digest and keeps execution metadata/evidence as typed attestations; useful for ChangeSet-bound runtime/evidence provenance.
- Copy policy: study behavior/tests/contracts; do not paste external implementation unless license/attribution is rechecked immediately before copying.
