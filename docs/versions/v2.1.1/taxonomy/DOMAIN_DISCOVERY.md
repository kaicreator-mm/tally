# SEK-01 — Software Engineering Domain Discovery

**Status:** PROPOSED / DISCOVERY IN PROGRESS / NOT_FROZEN  
**Task:** [#301](https://github.com/kaicreator-mm/tally/issues/301)  
**Parent:** [#299](https://github.com/kaicreator-mm/tally/issues/299)  
**Charter:** [#300](https://github.com/kaicreator-mm/tally/issues/300)  
**Research date:** 2026-11-04

## 1. Executive finding

The initial ten-category list is a useful project-oriented hypothesis, but it is not a sufficient discipline taxonomy. The strongest starting point found is the IEEE Computer Society's SWEBOK Guide v4.0a, released August 2026. It presents 18 Knowledge Areas (KAs), recognizes related disciplines, and explicitly distinguishes a guide to generally accepted knowledge from the body of knowledge itself.

The first-pass recommendation is therefore:

1. Use SWEBOK's 18 KAs as the **primary discipline-oriented reference spine**, not as a final immutable ontology.
2. Keep lifecycle/process, product quality, architecture description, application context and evidence strength as **orthogonal views/dimensions**.
3. Represent relationships across those views in a graph; do not force them into a single tree.
4. Treat the taxonomy and coverage profile as PROPOSED until source comparison, independent review and vertical-slice production challenge them.

## 2. Authority and purpose of the main references

### 2.1 SWEBOK Guide v4.0a — primary discipline taxonomy reference

**Identity:** Guide to the Software Engineering Body of Knowledge, Version 4.0a; editor Hironori Washizaki; IEEE Computer Society; released August 2026.

Official source: https://ieeecs-media.computer.org/media/education/swebok/swebok-v4.pdf  
Guide landing page: https://www.computer.org/education/bodies-of-knowledge/software-engineering

Useful verified locators:
- Introduction to the Guide, “What Is Software Engineering?”: printed pages xxxvii–xxxviii.
- Introduction, scope/objectives and related disciplines: printed pages xxxviii–xxxix.
- Table I.1, “The 18 SWEBOK KAs”, and Table I.2, “Related Disciplines”: printed page xxxix.
- “Hierarchical Organization” and “Depth of Treatment”: printed pages xl–xli.
- Official PDF opening/title page identifies v4.0a and September 2025 release.

The Guide says it organizes 18 KAs and uses a two- or three-level hierarchy of topics. It also says the Guide is not the body of knowledge itself, does not attempt to be comprehensive in its citations, and describes generally accepted knowledge rather than requiring uniform application to every project. Those limits matter: it is an excellent taxonomy anchor, not an exhaustive truth oracle.

The Guide recognizes eleven related disciplines: Business Analysis, Computer Engineering, Computer Science, Cybersecurity, Data Science, General Management, Information Systems and Technology, Mathematics, Project Management, Quality Management and Systems Engineering. It says characterizing the knowledge of those related disciplines is not the Guide's objective.

**18 Knowledge Areas (Table I.1):**

| SWEBOK KA | Candidate internal ID | Initial ten-category mapping / observation |
|---|---|---|
| Software Requirements | SE-KA-01 | Requirements Engineering |
| Software Architecture | SE-KA-02 | Architecture & Design |
| Software Design | SE-KA-03 | Architecture & Design |
| Software Construction | SE-KA-04 | Code Engineering & Maintainability |
| Software Testing | SE-KA-05 | Testing, Verification & Validation |
| Software Engineering Operations | SE-KA-06 | Integration/Delivery and Reliability/Operations; broader than CI/CD |
| Software Maintenance | SE-KA-07 | Maintenance, Evolution & Technical Debt |
| Software Configuration Management | SE-KA-08 | Missing as an explicit initial category |
| Software Engineering Management | SE-KA-09 | Engineering Management |
| Software Engineering Process | SE-KA-10 | Missing as a distinct knowledge area |
| Software Engineering Models and Methods | SE-KA-11 | Missing as a distinct knowledge area |
| Software Quality | SE-KA-12 | Quality is not merely a subtopic of management |
| Software Security | SE-KA-13 | Security Engineering |
| Software Engineering Professional Practice | SE-KA-14 | Missing as an explicit category |
| Software Engineering Economics | SE-KA-15 | Missing as an explicit category |
| Computing Foundations | SE-KA-16 | Foundations |
| Mathematical Foundations | SE-KA-17 | Foundations |
| Engineering Foundations | SE-KA-18 | Foundations |

The 18 names above are taken from SWEBOK Table I.1. The internal IDs are experimental Tally identifiers, not IEEE identifiers.

The table also shows why the original ten areas conflate different kinds of knowledge:
- Configuration management is not just build automation; it covers controlled change and configuration identity.
- Engineering process and models/methods are not equivalent to project management.
- Professional practice and engineering economics are first-class knowledge areas.
- Software quality is its own KA and also cross-cuts the others.
- Software Engineering Operations is wider than CI/CD: its name and chapter cover operational engineering, not merely pipeline tooling.
- Reliability, resilience and performance are distributed across several KAs rather than one clean SWEBOK KA.

### 2.2 ACM/IEEE-CS/AAAI CS2023 — curriculum and boundary comparator

Official site: https://csed.acm.org/  
Knowledge Areas: https://csed.acm.org/knowledge-areas/  
Final report page states endorsements by ACM (2024-01-18), IEEE-CS (2024-01-22) and AAAI (2024-02-22).

CS2023 defines 17 Computer Science curriculum Knowledge Areas. Its own knowledge-model document defines a curriculum KA as a set of Knowledge Units and states that KAs are not insular: cross-KA topic references may reflect different perspectives. Software Engineering is one of the 17 areas and its Body of Knowledge table lists 9 Knowledge Units:

1. Teamwork
2. Tools and Environments
3. Product Requirements
4. Software Design
5. Software Construction
6. Software Verification and Validation
7. Refactoring and Code Evolution
8. Software Reliability
9. Formal Methods

The official Body of Knowledge report places the SE KU descriptions around printed pages 166–167 and the core-hours table on pages 167–168. The SE-Tools section explicitly includes software configuration management/version control, release management, testing tools and process automation/CI. Other CS2023 areas include AI, Data Management, HCI, Security, Systems Fundamentals, Networking and Communication, and Society/Ethics/Profession.

**Interpretation:** CS2023 is valuable for distinguishing Software Engineering from the wider Computer Science curriculum and identifying shared/adjacent knowledge. It is a curriculum framework, not a complete taxonomy of the Software Engineering discipline. Its 9 SE knowledge units must not be mistaken for the full scope of the SWEBOK Guide. The KU selection also demonstrates that curriculum structures can place tools/SCM/CI, reliability and formal methods inside the Software Engineering area without reproducing SWEBOK's 18-KA hierarchy.

### 2.3 Standards and frameworks — orthogonal views, not one taxonomy

The following official publication identities were verified. ISO source pages include restrictions on using ISO content with AI/ML systems. Accordingly, this discovery draft uses bibliographic metadata only for ISO references; it does not claim clause-level content verification or reproduce normative text.

| Reference | Verified identity | Intended discovery use | Content status |
|---|---|---|---|
| ISO/IEC/IEEE 12207:2026 | Edition 2, published 2026-04-29; Software life cycle processes; https://www.iso.org/standard/90219.html | Candidate lifecycle/process view | Bibliographic metadata only; no normative text ingested |
| ISO/IEC/IEEE 15288:2023 | Edition 2, published 2023-05-16; System life cycle processes; https://www.iso.org/standard/81702.html | Systems-engineering boundary and software-in-system context | Bibliographic metadata only |
| ISO/IEC 25010:2023 | Edition 2, published 2023-11-15; Product quality model; https://www.iso.org/standard/78176.html | Candidate product-quality view | Bibliographic metadata only |
| ISO/IEC/IEEE 42010:2022 | Edition 2, published 2022-11-07; Architecture description; https://www.iso.org/standard/74393.html | Candidate architecture-description view | Bibliographic metadata only |
| ISO/IEC/IEEE 29119-2:2021 | Edition 2, published 2021-10-28; Software testing — Test processes; https://www.iso.org/standard/79428.html | Candidate testing-process reference | Bibliographic metadata only |
| NIST SP 800-218 v1.1 | February 2022; Secure Software Development Framework; https://csrc.nist.gov/pubs/sp/800/218/final | Security-practice framework and lifecycle cross-cutting | Official publication metadata and abstract verified |

These references are not interchangeable:
- a lifecycle/process standard is not a knowledge-area taxonomy;
- a quality model is not a development process;
- an architecture-description standard is not the whole of software architecture knowledge;
- a testing-process standard is not a complete testing knowledge area;
- a security-practice framework does not replace the broader Software Security KA.

No ISO clause, control, process name or normative requirement is asserted in this draft. Exact standard clauses require an authorized human-access and evidence-recording path consistent with the applicable terms.

## 3. Why one tree is insufficient

A single tree creates category errors because the candidate structures answer different questions:

| View | Question answered | Example authority/reference |
|---|---|---|
| Discipline / knowledge area | What kinds of knowledge belong to Software Engineering? | SWEBOK |
| Lifecycle / process | What process activities occur across acquisition, development, operation, maintenance and retirement? | ISO/IEC/IEEE 12207 candidate |
| System lifecycle context | How does software engineering sit within a larger system lifecycle? | ISO/IEC/IEEE 15288 candidate |
| Product quality | Which quality properties are being specified, evaluated or traded off? | ISO/IEC 25010 candidate |
| Architecture description | How are architecture concerns, viewpoints and descriptions structured? | ISO/IEC/IEEE 42010 candidate |
| Testing process | How is testing governed and performed? | ISO/IEC/IEEE 29119-2 candidate |
| Security practices | Which secure-development practices cross the lifecycle? | NIST SSDF |
| Curriculum / foundations | Which computing, mathematical, systems and professional areas intersect with software engineering education? | CS2023 |

A requirements concept can be associated with multiple lifecycle processes, quality attributes, artifacts, risks and evidence sources. A security practice can apply during requirements, design, construction, testing, delivery and operation. A quality attribute can affect architecture, implementation, testing and maintenance. A tree cannot express these many-to-many relationships without either duplication or misleading parentage.

**Proposed model:** one primary discipline hierarchy plus multiple typed, cross-linked views. A graph can represent cross-view relations; the authoring model must preserve the view/type of each assertion.

## 4. Review of the original ten-category hypothesis

The original list remains useful as an engineering-oriented navigation view, but should not be frozen as the canonical domain taxonomy.

| Original candidate | Discovery assessment |
|---|---|
| Foundations | Retain, but distinguish computing, mathematical and engineering foundations; identify external foundational knowledge vs core practice. |
| Requirements Engineering | Retain; use SWEBOK Software Requirements as the direct anchor. |
| Architecture & Design | Split into Software Architecture and Software Design; they are related but distinct KAs. |
| Code Engineering & Maintainability | Construction and Maintenance should be distinct; maintainability also cross-cuts quality/design/operations. |
| Testing, Verification & Validation | Retain as a major area; distinguish testing from broader verification/validation and quality assurance. |
| Security Engineering | Retain; software security is a dedicated KA, but cybersecurity remains a broader related discipline. |
| Reliability, Resilience & Performance | Retain as a cross-cutting quality/reliability view, not necessarily a single top-level discipline node. |
| Integration, Build, CI/CD & Delivery | Broaden to Software Engineering Operations and Configuration Management; process and methods also intersect. |
| Maintenance, Evolution & Technical Debt | Retain; explicitly connect to configuration management, quality and architecture. |
| Engineering Management, Risk & Quality | Split management, process, quality, professional practice and economics; risk remains cross-cutting. |

Missing explicit areas from the ten-category hypothesis:
1. Software Configuration Management.
2. Software Engineering Process.
3. Software Engineering Models and Methods.
4. Software Engineering Professional Practice.
5. Software Engineering Economics.

Conflated areas:
- Architecture and Design.
- Construction and Maintenance.
- Management and Quality.
- CI/CD and the broader Operations/Configuration Management disciplines.

## 5. Proposed multi-view taxonomy (draft)

### View A — Discipline knowledge

Use the 18 SWEBOK KAs as the initial reference spine. The taxonomy may regroup or split topics only when an evidence-backed reason is recorded. Maintain explicit mapping to SWEBOK rather than silently replacing its concepts.

### View B — Lifecycle/process

Map knowledge records to lifecycle/process contexts as many-to-many relationships. Do not encode the lifecycle as a mandatory waterfall sequence. The taxonomy must support iterative, concurrent and recursive engineering activities.

### View C — Quality attributes

Represent reliability, availability, resilience, performance, security, safety, maintainability, usability, accessibility, interoperability, compatibility and other validated quality concerns as cross-cutting dimensions. Exact ISO 25010 characteristics are not transcribed in this draft due source restrictions; they require permitted review.

### View D — Architecture and engineering concerns

Represent architecture views, stakeholders, concerns, decisions, patterns, constraints, tactics and trade-offs as typed concepts and relations. Architecture description is not synonymous with all architecture knowledge.

### View E — Foundations and related disciplines

Classify Computing, Mathematics and Engineering Foundations as explicit foundational knowledge areas. Represent Computer Science, Systems Engineering, Cybersecurity, Data Science, HCI, Business Analysis and other SWEBOK-related disciplines as external/shared knowledge references with explicit boundary relations.

### View F — Application and system context

Record contexts such as enterprise, cloud, embedded, IoT, AI/ML-based, safety-critical and regulated systems as applicability/context tags or specialized subdomain views. Do not duplicate the whole discipline taxonomy for each application domain.

## 6. Boundary hypotheses

These are provisional and require deeper evidence:

- **Computer Science:** provides scientific and technical foundations; Software Engineering is a distinct engineering discipline concerned with applying knowledge to real-world software problems under constraints. Do not absorb the whole CS curriculum.
- **Systems Engineering:** broader whole-system engineering, including hardware, software, people, processes and system context. Software Engineering is both a distinct discipline and a major contributor within systems engineering.
- **Cybersecurity:** broader than software security; software-specific security engineering belongs in the corpus, while network, organizational and infrastructure security knowledge is linked where necessary.
- **Data Engineering:** adjacent and intersecting; data-intensive software engineering practices are in scope, but the entire data science/analytics domain is not.
- **AI/ML Engineering:** emerging software-engineering specialization and application context; include engineering of AI-enabled software, model integration, evaluation, reliability, security and lifecycle issues when supported. Do not absorb all AI theory or claim a complete AI Engineering taxonomy.
- **HCI / accessibility:** shared interdisciplinary concerns; include software-engineering practices and interfaces to HCI/accessibility sources without claiming ownership of the entire disciplines.
- **Embedded / safety-critical engineering:** software-specific practices and constraints are in scope when they change engineering decisions; the full hardware/safety/regulatory domain is an adjacent body of knowledge.

## 7. Counterarguments and unresolved questions

1. SWEBOK is a consensus guide and may lag emerging practice; using it as the sole spine could reproduce its omissions.
2. SWEBOK KAs are not necessarily mutually exclusive or ontologically disjoint. The corpus must not assume each concept has exactly one parent.
3. Standards are scoped normative references, not empirical proof that a practice is effective in all contexts.
4. CS2023 reflects curriculum design, not the complete professional discipline.
5. A multi-view graph is more expressive but can be harder to author, validate and review than a simpler tree. SEK-04/05 must test this cost.
6. The right granularity of subdomains and cross-cutting dimensions remains open until the vertical slice.
7. The boundary of AI Engineering and Data Engineering needs further dedicated research before deciding whether they are specialized software-engineering areas, adjacent disciplines or separate domain families.

## 8. Next work

- SEK-02 establishes the authority/provenance rules and source registry.
- SEK-03 verifies a representative source set and exact permitted locators.
- SEK-04 uses the real taxonomy and sample records to challenge LinkML and authoring-format fit.
- SEK-06 should include contrasting slices: requirements/architecture, testing/security, and operations/maintenance (final selection after SEK-03).
- Do not freeze the taxonomy before independent review and real-record coverage testing.

## 9. Source and method notes

- SWEBOK Guide v4.0a was opened and inspected at the exact printed sections listed above; the 18-KA list and related-discipline table are verified.
- CS2023 official site confirms the final report endorsement dates; the official Knowledge Areas page and report summary identify 17 KAs and 9 SE Knowledge Units.
- ISO source records in this discovery are bibliographic-only because of the source-page AI/ML use restrictions. This draft does not assert normative clauses or use standard text as LLM prompt material.
- NIST SP 800-218 metadata and abstract were inspected; detailed practice-level extraction is deferred to source review.
- Search results and snippets were used for discovery only; they are not treated as full-text verification.


## 10. SEK-03 source verification corrections and extension (2026-11-04)

A full re-open of the official source artifacts found two bibliographic/locator defects in the initial discovery draft and registry:

1. The official SWEBOK v4.0a PDF title page states **Released August 2026**, not September 2025. The official copyright page prints ISBN **978-0-7695-0000-3**. The provisional ID suffix that encoded 2025 was therefore wrong; it is preserved only as a registry alias while the canonical source ID omits the year.
2. The CS2023 Software Engineering Knowledge Unit locators initially recorded as printed pages 166–168/170 were wrong. The official Body of Knowledge PDF's Software Engineering section begins at PDF page index 174; its nine-KU overview and core-hours table are on PDF page indices 175–177, and the SE-Tools unit begins at PDF page index 177/178 boundary. The registry now uses PDF page indices and named section headings to avoid printed-page offset ambiguity.

The source review also identified NIST SP 800-218A (final, 2024-07-26; DOI 10.6028/NIST.SP.800-218A) as a companion SSDF profile that adds practices for generative AI and dual-use foundation models and is intended to be used with SP 800-218. It is now a separately identified source record; only metadata and abstract are verified so far. This prevents the corpus from treating SP 800-218 v1.1 as the complete secure-development guidance for AI-model development.

These are material corrections from primary-source reinspection, not editorial date formatting.
