# SEK-07F Explanations — Maintenance, Evolution, Refactoring & Technical Debt

Narrative companion to `records.yaml`. Per the authoring format, this file links
canonical IDs and gives reading context only; every editable structured fact lives in
the YAML records. Nothing here overrides a record's scope, lifecycle or evidence chain.

## Why this lane reads maintenance as a post-delivery concern

The lane's anchor definition comes from SWEBOK KA7 (`se:EVD-07F-SWEBOK-MAINT-DEFINITION`
behind `se:CLM-07F-SWEBOK-MAINT-DEFINITION` and `se:CON-07F-SOFTWARE-MAINTENANCE`):
maintenance is the support of software already in operation, and it starts before
delivery. That framing is why the lane's lifecycle lens (`se:VIEW-07F-MAINTENANCE-EVOLUTION`)
sits apart from curriculum classification (`se:VIEW-CS2023-SE`) — the two lenses answer
different questions about the same subject, and `se:REL-07F-030` keeps that separation
explicit rather than merging the taxonomies.

## How the records fit together

- **What maintenance work looks like.** The five standardized request categories
  (`se:CLM-07F-SWEBOK-MAINT-CATEGORIES`) classify the demand side; the change-control
  chain (`se:CLM-07F-SWEBOK-KA8-SCR-PROCESS`, `se:CLM-07F-SWEBOK-IMPACT-ANALYSIS-ROLE`)
  describes how a request becomes a controlled change. The practice
  `se:PRA-07F-CHANGE-IMPACT-ANALYSIS` operationalizes that chain, and
  `se:NODE-07F-CHANGE-IMPACT-AND-CONTROL` is the lane's node for the concern.
- **Why systems keep changing.** The evolution-laws record
  (`se:CLM-07F-SWEBOK-EVOLUTION-LAWS`) is deliberately typed EMPIRICAL_FINDING *as
  reported by SWEBOK* — the primary Lehman research is not a registered source here
  (`se:GAP-07F-NO-CASE-EVIDENCE` records that limit). The complexity principle
  (`se:PRI-07F-MANAGE-EVOLUTION-COMPLEXITY`) is derived from it and inherits that
  secondhand character, which its limitations state.
- **Debt: definition, pathology, remedy, decision.** The effort-based definition
  (`se:CON-07F-TECHNICAL-DEBT`, `se:CLM-07F-SWEBOK-TECH-DEBT-DEFINITION`), the
  accumulation pathology (`se:ANT-07F-UNMANAGED-TECH-DEBT`), the remediation technique
  (`se:CON-07F-REFACTORING`, `se:PRA-07F-REFACTORING-WITH-TOOLS`) and the prioritization
  question (`se:DEC-07F-DEBT-VERSUS-FEATURES`) form one thread. The decision model is
  honest about its own origin: its single-procedure shape is the lane synthesis
  `se:EVD-07F-SYNTHESIS-DEBT-PRIORITIZATION`, not a sentence in SWEBOK, and the
  organizational feature-pressure it must survive qualifies it via `se:REL-07F-029`.
- **Changing interfaces safely.** The CS2023 records contribute the relational view of
  compatibility (`se:CLM-07F-CS2023-COMPAT-RELATIONSHIP`), the curriculum entry point
  for Hyrum's Law (`se:CON-07F-HYRUMS-LAW`, curriculum scope only — the law's full
  statement is not asserted) and the dual-API migration pattern
  (`se:PAT-07F-DUAL-API-LARGE-SCALE-REFACTORING`, an elective Non-core topic, which its
  scope_description says plainly). `se:REL-07F-031` records why the pattern and the
  implicit-dependency concept belong together.

## Reading discipline

Every claim-to-evidence trace is a `SUPPORTED_BY` edge in `relationship_assertions`;
records with `evidence_required` predicates carry their own `evidence_refs`. One
evidence record is marked `evidence_type: SYNTHESIS` (`se:EVD-07F-SYNTHESIS-DEBT-PRIORITIZATION`)
and everything derived from it is labeled as synthesis-derived. No record in this lane
is ACCEPTED; the observation file (`docs/versions/v2.1.1/methods/observations/SEK-07F.md`)
records locator verification, corrections and the Claim/CAS readback.
