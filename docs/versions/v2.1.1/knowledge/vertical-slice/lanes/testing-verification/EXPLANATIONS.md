# SEK-06C Explanations — Testing, Verification and Validation Lane

Narrative companion to `records.yaml`. Each section identifies canonical
record IDs and interprets them; structured field values remain authoritative
in the YAML and are not duplicated here. All records are
PENDING_REVIEW/PROPOSED and nothing here is accepted corpus knowledge.

## How the lane models "test execution is not correctness; validation is not acceptance"

The parent issue (#324) forbids two conflations, and the lane enforces them in
several places rather than in one note:

- `se:CON-TVV-VERIFICATION-VALIDATION` treats the V&V distinction as a
  **boundary concept**: the two questions interrogate different failure modes
  (disagreement with a specification versus building the wrong thing
  altogether), so no single activity — least of all running a test suite —
  answers both. Its scope explicitly separates the curriculum's "validation"
  test kind from "acceptance", which the curriculum itself lists separately in
  its learning-outcome comparison set.
- `se:CLM-TVV-0001` carries the limitation that the two-question formulation
  does not imply passing tests prove correctness or that validation equals
  acceptance. The claim stays at what the source states: two questions, and an
  expectation that students can tell them apart.
- `se:CLM-TVV-0004` records NIST's own framing: executable code testing is
  presented as *finding vulnerabilities*, not as verifying compliance
  wholesale or proving the software correct.
- `se:ANT-TVV-COVERAGE-THEATER` names the failure shape that arises when these
  distinctions collapse — a green, high-coverage suite read as correctness
  and as completed validation.

Reviewer attention: none of the three sources states "tests prove nothing";
the lane's position is the weaker, defensible one — the sources never warrant
the equivalence, and two of them actively structure their content around
testing as discovery (of defects, of vulnerabilities, of evolution breakage).

## Taxonomy placement

`se:VIEW-CS2023-SE` (shared) holds the curriculum lens. Within it,
`se:NODE-CS2023-SE-VALIDATION` is the V&V knowledge unit — the lane's primary
curriculum anchor — and `se:NODE-CS2023-SE-TOOLS` records where tooling sits
in the curriculum: testing tools live in the Tools and Environments unit,
while the V&V unit's own topics enumerate what test tooling covers (coverage
tooling, static analysis, fuzzing, dynamic analysis). Keeping two nodes
prevents a single blurred "testing and tools" blob.

`se:VIEW-SSDF-SECURITY` is deliberately a cross-cutting lens, not a parallel
discipline taxonomy. Its first node here, `se:NODE-SSDF-RV`, exists because
the RV group's content (ongoing code review/analysis/test, risk-based
disposition, root causes) genuinely intersects curriculum V&V topics — an
intersection the lane expresses as a typed edge
(`se:REL-TVV-0020`, CROSS_CUTS) rather than by merging nodes across views.
The RV.1.2 cross-reference to PW.7/PW.8 inside the NIST text itself is what
makes this intersection source-grounded rather than asserted.

## The evidence chain and its two velocities

Fifteen evidence records (`se:EVD-TVV-0001`..`se:EVD-TVV-0015`) split into
three provenance styles, all pending review:

- Registry-anchored paraphrases (overview, core-hours, SE-Tools, NIST Section
  1 risk guidance) — facts the source dossier had already verified.
- Heading-anchored CS2023 SE-Validation topic paraphrases — verified against
  the 2026-11-06 full-text extraction; no page numbers are asserted for
  KU-level locators beyond the registry's section-span anchor, because the
  extraction could not independently confirm printed pages.
- Page-level NIST paraphrases (PW.7, PW.8, RV group) — verified against the
  locally downloaded official PDF with exact PDF indices and printed pages.

One evidence record is explicitly different: `se:EVD-TVV-0014` is marked
SYNTHESIS. It exists so the lane's one interpretive leap — the coverage-theater
anti-pattern — has an auditable, separable basis. The DERIVED_FROM edge
(`se:REL-TVV-0019`) points the anti-pattern at that synthesis rather than
pretending a source named the anti-pattern.

## Claims worth a reviewer's first pass

- `se:CLM-TVV-0002` is the lane's most literal record: CS2023's topic list
  (unit, integration, validation, system) and its learning-outcome list
  (regression, unit, integration, systems, acceptance) do not match. The claim
  preserves the discrepancy instead of harmonizing it, because harmonizing
  would invent a taxonomy neither source states.
- `se:CLM-TVV-0004` (NIST PW.8) is qualified twice in the lane: by its own
  limitations (notional examples; no ordering implication) and structurally by
  `se:REL-TVV-0016` (QUALIFIED_BY the risk/cost/feasibility/applicability
  evidence `se:EVD-TVV-0015`). The recommendation is recorded as a
  recommendation, never promoted to universal practice.
- `se:CLM-TVV-0005` records the RV group's shape. Note what the claim does not
  say: it does not serialize RV.1→RV.2→RV.3 into a workflow, since the source
  disclaims ordering.

## From claims to actionable records

`se:PRI-TVV-DAMP-OVER-DRY` anchors test-code style to the curriculum's CS
Core topic; because the source states the rule without a rationale, the
principle's rationale field is written as offered-for-review interpretation,
and the record says so in its limitations.

The two practices diverge intentionally in authority. 
`se:PRA-TVV-CONTEXT-GATED-CODE-TESTING` is NIST-grounded: its first step is
the decision itself (whether to test at all, and with what), which is the
point of PW.8.1; its preconditions include knowing the software stage and
prior review/analysis results, mirroring how PW.8.1 scopes new testing against
previous findings (with PW.7.1 as the parallel human-readable-code decision).
`se:PRA-TVV-DEFECT-TRACKING-REPRODUCIBILITY` braids a curriculum topic
(defect tracking and prioritization, reproducibility) with the NIST
record-and-triage tasks; `se:REL-TVV-0018` (IMPLEMENTS) records that it
operationalizes only the record/triage portion of `se:CLM-TVV-0004`, not the
gating decision — a deliberate qualifier to block reading one practice as
fulfilling the whole recommendation.

`se:PAT-TVV-REGRESSION-SAFETY-NET` is the lane's candidate pattern: recorded
defect knowledge becomes executable regression checks. Its grounding is a
notional example in NIST plus the curriculum's evolution rationale for
testing; it is therefore a candidate, and `se:GAP-TVV-NO-EMPIRICAL-CASES`
records exactly the missing evidence class (real cases) that would be needed
before promotion. Its DEPENDS_ON edge (`se:REL-TVV-0017`) ties it to
`se:CON-TVV-TEST-COVERAGE`, since a growing regression suite is precisely
where the coverage-versus-throughput tradeoff bites.

`se:DM-TVV-PRE-POST-COMMIT-TESTING` turns the curriculum's pre/post-commit
and tradeoff topics into a reusable decision frame. It structures judgment —
which tests gate a commit — without deciding for any team, matching the
DecisionModel contract that a model structures rather than decides.

## What the gaps say about the lane

The four gap records (`se:GAP-TVV-ISO-29119-2-METADATA-ONLY`,
`se:GAP-TVV-SWEBOK-KA5-HOLD`, `se:GAP-TVV-NO-EMPIRICAL-CASES`, and
`se:GAP-TVV-CS2023-RIGHTS-CLEARANCE`) are first-class results. They mark the
unavailable normative test-process source, the professional-BOK hold, the lack
of empirical cases, and the unresolved permission scope for using CS2023 in a
general engineering knowledge graph. No educational-use permission is
presumed to cover this project's broader purpose. Until these gaps close, the
lane remains a curriculum-plus-recommendation view with explicit authority and
rights limits.
