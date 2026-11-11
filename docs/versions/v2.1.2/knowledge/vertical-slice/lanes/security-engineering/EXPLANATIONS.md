# SEK-07C — Security Engineering lane: narrative explanations

This file links canonical record IDs and gives narrative context only. It does
not duplicate the structured facts (statements, locators, qualifiers, counts)
that live in `records.yaml`; that file is the editable source of truth for this
lane. All referenced records are PROPOSED or PENDING_REVIEW pending human review.

## Why this lane reads security as lifecycle-cross-cutting

The lane's organizing node `se:NODE-07C-SECURITY-LIFECYCLE` exists because all
three authorities converge — each in its own register — on the idea that
security is not one phase. SWEBOK reports the design-in preference as a
generally accepted belief (`se:CLM-07C-SWEBOK-SECURITY-BY-DESIGN`); SSDF turns
the same intuition into a risk-based recommendation
(`se:CLM-07C-NIST-SECURE-BY-DESIGN`); CS2023 teaches it as a mindset
(`se:CLM-07C-CS2023-PROTECTION-PRINCIPLES`). The lane's principle
`se:PRI-07C-DESIGN-SECURITY-IN` states the shared position while its
QUALIFIED_BY edge records that even the curriculum balances security against
other design goals — the principle presumes tradeoff analysis, not
maximization.

## The requirements thread

The concept `se:CON-07C-SECURITY-REQUIREMENT` and the practice
`se:PRA-07C-SECURITY-REQUIREMENTS-BASELINE` sit at the lane's center because
both SWEBOK 4.1 (`se:CLM-07C-SWEBOK-SECURITY-REQUIREMENTS-SCOPE`) and SSDF PO.1
(`se:CLM-07C-NIST-SSDF-REQUIREMENTS-CONTINUOUS`) anchor everything else to a
requirements record that exists, is maintained, and is traceable. The
verification node depends on this thread
(`se:REL-07C-023`): one of security testing's two purposes is checking the
implementation against exactly these requirements
(`se:CLM-07C-SWEBOK-SECURITY-TESTING-PURPOSES`,
`se:PRA-07C-SECURITY-TESTING-DUAL-CHECK`).

## Design, patterns and judgment

`se:NODE-07C-SECURE-DESIGN` gathers the threat-modeling practice
(`se:PRA-07C-THREAT-MODELING-AT-DESIGN`, concept
`se:CON-07C-THREAT-MODEL`), the security-pattern pattern
(`se:PAT-07C-SECURITY-PATTERN-REUSE`, concept
`se:CON-07C-SECURITY-PATTERN`) and the review-depth decision model
(`se:DM-07C-DESIGN-REVIEW-DEPTH`). The decision model is deliberately built
from two different authorities — SSDF PW.2's independence condition plus the
CS2023 tradeoff factors — so that review depth is a judgment, not a checklist
item; SSDF is risk-based guidance, not a mandatory sequence.

## The negative space

`se:ANT-07C-SECURITY-AFTERTHOUGHT` names what the positive records exclude:
security deferred until after development. It is derived from the explicitly
marked synthesis `se:EVD-07C-SYNTHESIS-AFTERTHOUGHT`, not from any source
sentence — the sources assert the positive practice; the anti-pattern framing
is the lane's inference. What the lane could not ground at all is recorded as
gaps rather than invented: no real cases exist here
(`se:GAP-07C-NO-CASE-EVIDENCE`), several SWEBOK KA13 sections remain
unadmitted (`se:GAP-07C-SWEBOK-KA13-UNINSPECTED-SECTIONS`), and the
generative-AI companion profile stays abstract-only
(`se:GAP-07C-NIST-218A-ABSTRACT-ONLY`). The shared rights condition for CS2023
reuse is carried unchanged as `se:GAP-CS2023-RIGHTS-CLEARANCE`.

## Reading order for a reviewer

1. Start at `se:PRI-07C-DESIGN-SECURITY-IN` and its two evidence anchors
   (`se:EVD-07C-SWEBOK-SECURITY-BY-DESIGN`,
   `se:EVD-07C-NIST-PW1-SECURE-BY-DESIGN`).
2. Follow the requirements thread: concept → practice → node → claims.
3. Check the four PART_OF edges under `se:NODE-07C-SECURITY-LIFECYCLE` for the
   lifecycle structure.
4. Verify each claim's SUPPORTED_BY edge lands on an Evidence record whose
   locator you can re-open in the named PDF section.
