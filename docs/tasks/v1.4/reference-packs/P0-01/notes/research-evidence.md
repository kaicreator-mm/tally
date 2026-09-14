# External Research Evidence

External references are study material. Local TaskContract/architecture remains authoritative.

## REF-SLSA-PROV — slsa-framework/slsa
- Pin: `spec v1.1/v1.2 family`
- License: `Study-only; verify exact spec terms before copying`
- URL: https://github.com/slsa-framework/slsa/blob/main/spec/build-provenance.md
- Relevant paths:
  - `spec/build-provenance.md`
- Why relevant: Separates builder identity from per-invocation metadata such as invocationId/start/finish; useful for observed runtime identity versus desired configuration.
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
