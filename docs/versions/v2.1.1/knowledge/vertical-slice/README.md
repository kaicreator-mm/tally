# Initial Software Engineering Knowledge Vertical Slice

**Status:** PROPOSED / PENDING HUMAN REVIEW  
**Task:** SEK-03 #304  
**Purpose:** Real records for early authoring-format and LinkML Proof-of-fit.

This slice intentionally contains only a small set of verifiable facts from SWEBOK v4.0a and CS2023. It tests the traceability chain:

~~~text
Claim → Evidence → SourceLocator → Source
~~~

The YAML file includes:
- three source references (SWEBOK, CS2023 and NIST);
- eight source locators, with different locator types and verification levels;
- seven evidence records;
- seven atomic claims;
- seven typed SUPPORTS relationships.

NIST is included as a source identity/abstract-only example but contributes no detailed practice claim. ISO sources remain outside this content fixture because their records are bibliographic-only under the recorded AI/ML-use restrictions.

## Admission rules

- PROPOSED claims are not accepted knowledge.
- Human review must confirm the source, locator, interpretation, scope and rights.
- The 18 SWEBOK KAs and nine CS2023 SE Knowledge Units are different classification structures, not equivalent node sets.
- This sample is deliberately small and is not a coverage claim.

## Authoring-format experiment

SEK-04 will encode the same logical claim/evidence records as:
A. Markdown with YAML front matter.
B. YAML instance plus linked Markdown explanation.
C. Standalone YAML.

The three representations are experimental fixtures, not three competing canonical copies.
