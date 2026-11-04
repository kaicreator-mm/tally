# SEK-02 — Method Observation

**Task:** #302  
**Date:** 2026-11-04  
**Status:** OBSERVED / DRAFT

## Observations

1. Source authority cannot be reduced to one rank. SWEBOK is a professional consensus body-of-knowledge guide; CS2023 is a curriculum framework; ISO/IEC/IEEE standards are normative within their applicability; NIST SP 800-218 is a government-published recommendation framework. They support different claim types.
2. Source identity, content, locator, extracted claim, reviewed synthesis and accepted knowledge are distinct verification states. The registry therefore records them separately.
3. The official ISO publication pages state restrictions on using ISO content with AI/ML systems, including prompts. This is a material source-access constraint. ISO records are bibliographic-only in this draft; no normative clause or generated clause summary is included.
4. The SWEBOK PDF copyright notice permits educational/personal use subject to conditions and requires permission for commercial reprint/redistribution or new collective works. The corpus must prefer concise original synthesis and bibliographic references over reproduced text.
5. NIST SP 800-218 has a verified DOI and official CSRC publication record; the current registry verifies metadata and abstract only, not every practice or control.

## Negative / unresolved observations

- No licensed/authorized human access to the full ISO standards has been established.
- No full systematic review of all source references in SWEBOK has been completed.
- No books, ISBNs, paper DOIs or page locators have been added unless independently verified.
- The current registry is representative and incomplete by design.

## Method adjustment

Evaluate source authority per claim type; keep source class and evidence role separate; use bibliographic-only records when content access is restricted; never fabricate missing locators; preserve rights/access restrictions as first-class provenance fields.


## Source re-verification correction — 2026-11-04

Primary-source reinspection exposed a real bibliographic error: the SWEBOK v4.0a title page states **Released August 2026**, not September 2025, and the copyright page prints ISBN 978-0-7695-0000-3. The initial source ID embedded the incorrect year. The old ID is preserved as a non-canonical alias; the canonical ID now omits the year. This is a useful warning against encoding mutable or weakly verified publication metadata into stable identifiers.

A second locator error was found in the CS2023 registry entry: the nine SE Knowledge Units were initially assigned pages 166–168/170, but the official Body of Knowledge PDF places the Software Engineering section around PDF page indices 174–178. The registry now records PDF page indices plus named sections, avoiding the offset ambiguity between PDF indices and displayed page numbers.

The NIST source review discovered the final NIST SP 800-218A (2024-07-26), a companion profile that augments SP 800-218 v1.1 for generative AI and dual-use foundation models. The profile is now separately registered with metadata/abstract-only verification. Do not silently treat v1.1 as the whole AI-specific secure-development corpus.

## Method adjustment

Reopen the publisher's actual PDF/report, not just the landing-page snippet, before pinning edition dates, identifiers or page locators. Preserve superseded IDs as aliases where they have already been referenced. Record the distinction between PDF page index and printed page number. Check for companion/current profiles when a source is used as a security or AI boundary reference.


## Registry schema consistency check — 2026-11-04

The registry was parsed with PyYAML 6.0.3 using duplicate-key detection. It contains 9 unique canonical source IDs and one preserved SWEBOK legacy-ID alias whose canonical target resolves. All source references in the current taxonomy resolve against canonical registry IDs.

Source roles were normalized to the broad PRIMARY/SECONDARY vocabulary, with a separate `role_scope` field explaining claim-specific authority fit. This avoids encoding an overly specific claim type into the generic source-role field.
