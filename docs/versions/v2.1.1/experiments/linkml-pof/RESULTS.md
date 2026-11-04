# SEK-04 — Initial LinkML Proof-of-fit Results

**Status:** DESIGN_ONLY / CLI_NOT_RUN / NOT_ADOPTED  
**Candidate pin:** LinkML v1.12.1  
**As of:** 2026-11-04

## 1. What has been verified

Documentation-level research:
- LinkML supports identifier slots and non-inlined references by identifier.
- LinkML supports multivalued inlined lists and non-inlined reference fields.
- LinkML can generate JSON-LD contexts and RDF serializations supported by RDFLib.
- The JSON-LD documentation explicitly notes that @type information is not generated at all levels.
- The JSON Schema generator documentation lists unsupported LinkML constraints, including unique_keys.
- The official CLI provides schema metamodel validation, data validation and linting.

Real-data preparation:
- SEK-03 provides 3 source references, 8 locators, 7 evidence records, 7 claims and 7 typed support relationships.
- The initial vertical-slice YAML parses successfully with PyYAML; all IDs are unique and all declared references resolve.
- The three format fixtures encode the same SE-CLM-0001 claim and evidence chain.

## 2. What has not been executed

| Test | Status | Reason |
|---|---|---|
| LinkML schema metamodel validation | NOT_RUN | LinkML CLI unavailable in current sandbox |
| LinkML data validation | NOT_RUN | Same |
| LinkML lint | NOT_RUN | Same |
| JSON Schema generation | NOT_RUN | Same |
| JSON-LD context/frame generation | NOT_RUN | Same |
| RDF/Turtle generation | NOT_RUN | Same |
| Duplicate/dangling reference adversarial tests | NOT_RUN | Requires LinkML runtime and graph validator |
| Schema evolution test | NOT_RUN | Requires executable baseline |
| Human Git Diff comparison | DESIGN_ONLY | Requires reviewers to inspect the three fixtures |

The current analysis sandbox does not have LinkML installed. A package-install attempt did not produce a usable installation. No specific DNS/network cause is asserted.

## 3. Early findings — hypotheses only

- The model can express stable identifiers and non-inlined references in principle.
- A polymorphic RelationshipAssertion with string endpoints plus entity-kind enums needs graph-level endpoint resolution.
- JSON Schema output may not enforce the SourceLocator unique key.
- JSON-LD context generation may not preserve explicit types at all nested levels without additional framing or transformation.
- Markdown front matter requires a parser/adapter before LinkML can validate the extracted data.
- Standalone YAML is structurally convenient but may create large, noisy diffs.
- YAML plus linked Markdown may provide a useful separation of machine-validated fields and human explanation, but this must be checked in actual review.

## 4. Decision

**No LinkML decision is made.** The current state is DESIGN_ONLY / NOT_RUN. It is not ADOPT, REJECT or PASS.

The next valid step is a bounded toolchain execution against the exact schema and fixtures, recording:
- LinkML version and Python version;
- schema metamodel/lint output;
- valid and invalid data validation output;
- generated JSON Schema, JSON-LD context/frame and RDF/Turtle;
- repeated-run diff stability;
- schema migration behavior;
- human review comparison.

If that execution cannot be performed, record REPEAT_EXPERIMENT rather than inferring success from documentation.
