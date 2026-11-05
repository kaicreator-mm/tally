#!/usr/bin/env python3
"""Deterministic experimental graph-integrity validator for SEK-05.

This is documentation tooling, not a Tally runtime dependency. LinkML validates
record shape; this program validates graph-wide identity, provenance, lifecycle
and predicate-specific endpoint constraints.
"""
from __future__ import annotations

import argparse
import copy
import hashlib
import re
import sys
from collections import defaultdict
from pathlib import Path

import yaml


ROOT = Path(__file__).resolve().parent
DEFAULT_CATALOG = ROOT / "ENTITY_CATALOG.yaml"
DEFAULT_RULES = ROOT / "RELATIONSHIP_CATALOG.yaml"
ID_PATTERN = re.compile(r"^se:[A-Z0-9][A-Z0-9._-]*$")
SHA256_PATTERN = re.compile(r"^[0-9a-f]{64}$")

COLLECTIONS = {
    "domains": "Domain",
    "subdomains": "Subdomain",
    "taxonomy_views": "TaxonomyView",
    "taxonomy_nodes": "TaxonomyNode",
    "sources": "Source",
    "source_locators": "SourceLocator",
    "evidence_records": "Evidence",
    "concepts": "Concept",
    "knowledge_claims": "KnowledgeClaim",
    "principles": "Principle",
    "practices": "Practice",
    "patterns": "Pattern",
    "anti_patterns": "AntiPattern",
    "decision_models": "DecisionModel",
    "cases": "Case",
    "counterexamples": "Counterexample",
    "review_decisions": "ReviewDecision",
    "coverage_gaps": "CoverageGap",
    "relationship_assertions": "RelationshipAssertion",
}

KNOWLEDGE_CLASSES = {
    "Domain", "Subdomain", "TaxonomyView", "TaxonomyNode", "Concept",
    "KnowledgeClaim", "Principle", "Practice", "Pattern", "AntiPattern",
    "DecisionModel", "Case", "Counterexample", "CoverageGap",
}
NON_KNOWLEDGE_CLASSES = {"Source", "SourceLocator", "Evidence", "ReviewDecision"}
PROVENANCE_EXEMPT = {"Source", "SourceLocator", "Evidence", "ReviewDecision"}

class ValidationError(Exception):
    pass


class UniqueLoader(yaml.SafeLoader):
    pass


def _construct_unique_mapping(loader, node, deep=False):
    result = {}
    for key_node, value_node in node.value:
        key = loader.construct_object(key_node, deep=deep)
        if key in result:
            mark = key_node.start_mark
            raise ValidationError(
                f"duplicate YAML key {key!r} at line {mark.line + 1}"
            )
        result[key] = loader.construct_object(value_node, deep=deep)
    return result


UniqueLoader.add_constructor(
    yaml.resolver.BaseResolver.DEFAULT_MAPPING_TAG,
    _construct_unique_mapping,
)


def load_yaml(path: Path):
    with path.open("r", encoding="utf-8") as handle:
        return yaml.load(handle, Loader=UniqueLoader)


def _ids(values):
    if values is None:
        return []
    if isinstance(values, str):
        return [values]
    if not isinstance(values, list):
        return []
    return values


def _error(errors, code, detail):
    errors.append(f"{code}: {detail}")


def validate_graph(graph, entity_catalog, relationship_catalog):
    errors = []
    if not isinstance(graph, dict):
        return ["ROOT_NOT_MAPPING: graph root must be a mapping"]

    entities = {}
    classes = {}
    records = []
    for collection, class_name in COLLECTIONS.items():
        values = graph.get(collection, [])
        if not isinstance(values, list):
            _error(errors, "COLLECTION_NOT_LIST", collection)
            continue
        for index, record in enumerate(values):
            if not isinstance(record, dict):
                _error(errors, "RECORD_NOT_MAPPING", f"{collection}[{index}]")
                continue
            record_id = record.get("id")
            if not isinstance(record_id, str) or not ID_PATTERN.fullmatch(record_id):
                _error(errors, "INVALID_CANONICAL_ID", f"{collection}[{index}] id={record_id!r}")
                continue
            if record_id in entities:
                _error(errors, "DUPLICATE_IDENTIFIER", record_id)
                continue
            entities[record_id] = record
            classes[record_id] = class_name
            records.append((class_name, record_id, record))

    aliases = {}
    for class_name, record_id, record in records:
        for field in ("record_version", "lifecycle_status", "created_on"):
            if not record.get(field):
                _error(errors, "MISSING_COMMON_FIELD", f"{record_id}.{field}")
        if not re.fullmatch(r"\d+\.\d+\.\d+", str(record.get("record_version", ""))):
            _error(errors, "INVALID_RECORD_VERSION", record_id)
        for alias in _ids(record.get("aliases")):
            if not isinstance(alias, str) or not alias.strip():
                _error(errors, "INVALID_ALIAS", record_id)
            elif alias in aliases or alias in entities:
                _error(errors, "DUPLICATE_ALIAS", alias)
            else:
                aliases[alias] = record_id
        status = record.get("lifecycle_status")
        if status == "ACCEPTED" and class_name in KNOWLEDGE_CLASSES:
            if record.get("synthetic") is True:
                _error(errors, "ACCEPTED_SYNTHETIC_RECORD", record_id)
            if not _ids(record.get("evidence_refs")):
                _error(errors, "ACCEPTED_WITHOUT_EVIDENCE", record_id)

    def ref(record_id, field, expected, required=False):
        record = entities.get(record_id, {})
        value = record.get(field)
        if required and not value:
            _error(errors, "MISSING_REQUIRED_REFERENCE", f"{record_id}.{field}")
            return
        if not value:
            return
        values = _ids(value)
        for target in values:
            if target not in entities:
                _error(errors, "DANGLING_REFERENCE", f"{record_id}.{field}->{target}")
            elif expected and classes[target] not in expected:
                _error(errors, "WRONG_REFERENCE_CLASS", f"{record_id}.{field}->{target} is {classes[target]}, expected {sorted(expected)}")

    # The root's declared domain is the canonical graph scope.
    root_domain = graph.get("domain_ref")
    if root_domain not in entities or classes.get(root_domain) != "Domain":
        _error(errors, "INVALID_GRAPH_DOMAIN", str(root_domain))

    allowed_lifecycle = set(
        entity_catalog.get("identity_contract", {}).get("lifecycle", {}).get("values", [])
    )
    for class_name, record_id, record in records:
        if allowed_lifecycle and record.get("lifecycle_status") not in allowed_lifecycle:
            _error(errors, "INVALID_LIFECYCLE_STATUS", f"{record_id}: {record.get('lifecycle_status')!r}")
        if "evidence_refs" in record:
            ref(record_id, "evidence_refs", {"Evidence"})
        if class_name == "KnowledgeClaim" and record.get("lifecycle_status") == "ACCEPTED":
            if record.get("review_status") != "ACCEPTED":
                _error(errors, "CLAIM_REVIEW_STATUS_MISMATCH", record_id)
        if class_name == "Subdomain":
            ref(record_id, "domain_ref", {"Domain"}, True)
        elif class_name == "TaxonomyNode":
            ref(record_id, "view_ref", {"TaxonomyView"}, True)
            ref(record_id, "source_locator_refs", {"SourceLocator"})
        elif class_name == "SourceLocator":
            ref(record_id, "source_ref", {"Source"}, True)
        elif class_name == "Evidence":
            ref(record_id, "source_locator_ref", {"SourceLocator"}, True)
        elif class_name == "Concept":
            ref(record_id, "evidence_refs", {"Evidence"})
            ref(record_id, "claim_refs", {"KnowledgeClaim"})
        elif class_name == "KnowledgeClaim":
            ref(record_id, "evidence_refs", {"Evidence"})
        elif class_name in {"Principle", "Practice", "DecisionModel"}:
            ref(record_id, "evidence_refs", {"Evidence"})
            ref(record_id, "claim_refs", {"KnowledgeClaim"})
        elif class_name == "Pattern":
            ref(record_id, "evidence_refs", {"Evidence"})
            ref(record_id, "case_refs", {"Case", "Counterexample"})
        elif class_name == "AntiPattern":
            ref(record_id, "evidence_refs", {"Evidence"})
            ref(record_id, "case_refs", {"Case", "Counterexample"})
        elif class_name == "Case":
            ref(record_id, "evidence_refs", {"Evidence"})
        elif class_name == "Counterexample":
            ref(record_id, "evidence_refs", {"Evidence"})
            ref(record_id, "counterexample_to_refs", KNOWLEDGE_CLASSES, True)
        elif class_name == "ReviewDecision":
            ref(record_id, "target_ref", set(COLLECTIONS.values()) - {"ReviewDecision"}, True)
            target = entities.get(record.get("target_ref"), {})
            if record.get("target_record_version") != target.get("record_version"):
                _error(errors, "REVIEW_VERSION_MISMATCH", record_id)
            if not SHA256_PATTERN.fullmatch(str(record.get("target_digest_sha256", ""))):
                _error(errors, "INVALID_REVIEW_DIGEST", record_id)
            if record.get("decision") == "ACCEPT" and target.get("lifecycle_status") != "ACCEPTED":
                _error(errors, "REVIEW_ACCEPT_TARGET_NOT_ACCEPTED", record_id)
        elif class_name == "CoverageGap":
            ref(record_id, "target_scope_ref", {"Domain", "Subdomain", "TaxonomyView", "TaxonomyNode"}, True)
            ref(record_id, "evidence_refs", {"Evidence"})
        elif class_name == "RelationshipAssertion":
            ref(record_id, "subject_ref", set(COLLECTIONS.values()) - {"ReviewDecision"}, True)
            ref(record_id, "object_ref", set(COLLECTIONS.values()) - {"ReviewDecision"}, True)
            ref(record_id, "evidence_refs", {"Evidence"})

    locator_keys = {}
    for class_name, record_id, record in records:
        if class_name == "SourceLocator":
            key = (record.get("source_ref"), record.get("locator"))
            if key in locator_keys:
                _error(errors, "DUPLICATE_SOURCE_LOCATOR", f"{record_id} duplicates {locator_keys[key]}")
            locator_keys[key] = record_id
        if class_name == "Evidence":
            locator = entities.get(record.get("source_locator_ref"), {})
            if locator and locator.get("source_ref") not in entities:
                _error(errors, "BROKEN_EVIDENCE_SOURCE_CHAIN", record_id)
        if class_name in KNOWLEDGE_CLASSES and record.get("lifecycle_status") == "ACCEPTED":
            if class_name not in PROVENANCE_EXEMPT and not _ids(record.get("evidence_refs")):
                _error(errors, "ACCEPTED_WITHOUT_EVIDENCE", record_id)
        if class_name in {"Case", "Counterexample"}:
            if record.get("synthetic") is True and record.get("lifecycle_status") == "ACCEPTED":
                _error(errors, "ACCEPTED_SYNTHETIC_RECORD", record_id)
            if record.get("synthetic") is False and record.get("lifecycle_status") == "ACCEPTED" and not _ids(record.get("evidence_refs")):
                _error(errors, "REAL_CASE_WITHOUT_EVIDENCE", record_id)

    predicate_rules = {
        item["predicate"]: item
        for item in relationship_catalog.get("predicates", [])
    }
    seen_edges = set()
    graph_edges = defaultdict(list)
    for class_name, record_id, record in records:
        if class_name != "RelationshipAssertion":
            continue
        predicate = record.get("predicate")
        rule = predicate_rules.get(predicate)
        if not rule:
            _error(errors, "UNKNOWN_PREDICATE", f"{record_id}: {predicate!r}")
            continue
        subject_id = record.get("subject_ref")
        object_id = record.get("object_ref")
        subject_class = classes.get(subject_id)
        object_class = classes.get(object_id)
        if subject_class not in set(rule.get("subject_classes", [])):
            _error(errors, "WRONG_SUBJECT_CLASS", f"{record_id}: {subject_class} not allowed for {predicate}")
        if object_class not in set(rule.get("object_classes", [])):
            _error(errors, "WRONG_OBJECT_CLASS_FOR_PREDICATE", f"{record_id}: {object_class} not allowed for {predicate}")
        if rule.get("evidence_required") and not _ids(record.get("evidence_refs")):
            _error(errors, "MISSING_REQUIRED_EVIDENCE", record_id)
        if rule.get("qualifiers_required") and not _ids(record.get("qualifiers")):
            _error(errors, "MISSING_REQUIRED_QUALIFIER", record_id)
        if record.get("lifecycle_status") == "ACCEPTED":
            if not _ids(record.get("evidence_refs")) and not rule.get("object_is_evidence") and not rule.get("object_carries_provenance"):
                _error(errors, "ACCEPTED_RELATIONSHIP_WITHOUT_EVIDENCE", record_id)
        if not record.get("context") or not record.get("rationale"):
            _error(errors, "MISSING_RELATION_CONTEXT", record_id)
        edge_key = (subject_id, predicate, object_id)
        if edge_key in seen_edges and not _ids(record.get("qualifiers")):
            _error(errors, "DUPLICATE_RELATIONSHIP", record_id)
        seen_edges.add(edge_key)
        if predicate in {"PART_OF", "SUPERSEDES"}:
            graph_edges[predicate].append((subject_id, object_id))
        if predicate == "SUPERSEDES" and subject_class != object_class:
            _error(errors, "SUPERSEDES_CLASS_MISMATCH", record_id)

    for predicate, edges in graph_edges.items():
        adjacency = defaultdict(list)
        for source, target in edges:
            adjacency[source].append(target)
        visiting, visited = set(), set()

        def visit(node):
            if node in visiting:
                return True
            if node in visited:
                return False
            visiting.add(node)
            for child in adjacency.get(node, []):
                if visit(child):
                    return True
            visiting.remove(node)
            visited.add(node)
            return False

        if any(visit(node) for node in list(adjacency)):
            _error(errors, "PART_OF_CYCLE" if predicate == "PART_OF" else "SUPERSEDES_CYCLE", predicate)

    # Accepted records require a human ACCEPT decision bound to the same revision.
    accepted_ids = {
        record_id for class_name, record_id, record in records
        if class_name in KNOWLEDGE_CLASSES and record.get("lifecycle_status") == "ACCEPTED"
    }
    accepted_targets = {
        record.get("target_ref")
        for class_name, _, record in records
        if class_name == "ReviewDecision" and record.get("decision") == "ACCEPT"
    }
    for record_id in sorted(accepted_ids - accepted_targets):
        _error(errors, "ACCEPTED_WITHOUT_HUMAN_REVIEW", record_id)

    return errors


def positive_fixture():
    common = {
        "record_version": "0.1.0",
        "created_on": "2026-11-05",
    }

    def rec(record_id, status="PROPOSED", **fields):
        return {"id": record_id, **common, "lifecycle_status": status, **fields}

    graph = {
        "id": "se:GRAPH-TEST-001",
        "graph_version": "0.1.0",
        "generated_on": "2026-11-05",
        "domain_ref": "se:DOMAIN-SE",
        "domains": [rec("se:DOMAIN-SE", label="Software Engineering", definition="A bounded engineering discipline.", scope_inclusions=["software lifecycle and engineering practice"], scope_exclusions=["all of computer science"], boundary_notes=["Coverage is relative to a reviewed profile."], evidence_refs=["se:EVD-001"])],
        "subdomains": [rec("se:SUB-REQ", label="Requirements Engineering", definition="Elicitation, analysis and specification.", domain_ref="se:DOMAIN-SE", boundary_notes=["Interfaces with business analysis."], evidence_refs=["se:EVD-001"])],
        "taxonomy_views": [
            rec("se:VIEW-DISCIPLINE", label="Discipline view", view_kind="DISCIPLINE", purpose="Classify professional knowledge areas.", coverage_state="PARTIAL", limitations=["SWEBOK is not exhaustive."], evidence_refs=["se:EVD-001"]),
            rec("se:VIEW-SECURITY", label="Security view", view_kind="SECURITY", purpose="Classify secure development practices.", coverage_state="PROPOSED", limitations=["NIST content is scoped to inspected sections."], evidence_refs=["se:EVD-001"]),
        ],
        "taxonomy_nodes": [
            rec("se:NODE-REQ", label="Requirements Engineering", definition="Requirements knowledge area.", node_kind="KNOWLEDGE_AREA", view_ref="se:VIEW-DISCIPLINE", source_locator_refs=["se:LOC-001"], evidence_refs=["se:EVD-001"]),
            rec("se:NODE-SSDF", label="Secure Software Development", definition="Secure development practice view.", node_kind="PRACTICE_GROUP", view_ref="se:VIEW-SECURITY", source_locator_refs=["se:LOC-001"], evidence_refs=["se:EVD-001"]),
        ],
        "sources": [rec("se:SRC-001", "ACTIVE", title="SWEBOK Guide v4.0a", source_class="PROFESSIONAL_BOK", source_roles=["PRIMARY"], organization="IEEE Computer Society", authors=["Hironori Washizaki"], edition="4.0a", publication_date="2026-08", official_url="https://example.invalid/swebok.pdf", stable_identifier="ISBN 978-0-7695-0000-3", authority_fit=["Professional BOK taxonomy"], authority_limitations=["Not exhaustive."], rights_handling="Concise paraphrase only.", verification_state="CONTENT_PARTIAL_VERIFIED")],
        "source_locators": [rec("se:LOC-001", "ACTIVE", source_ref="se:SRC-001", locator_type="TABLE", locator="Table I.1", verified_content_summary="The Guide lists knowledge areas.", verification_state="LOCATOR_VERIFIED", accessed_on="2026-11-05")],
        "evidence_records": [rec("se:EVD-001", "PENDING_REVIEW", source_locator_ref="se:LOC-001", evidence_type="DIRECT_SOURCE_PARAPHRASE", statement="The source lists defined knowledge areas.", context="Taxonomy description.", method="Inspected official PDF.", result="A taxonomy list is present.", limitations=["Not a completeness proof."], uncertainty=[], extraction_status="EXTRACTED_PENDING_REVIEW", synthetic=False)],
        "concepts": [rec("se:CON-REQ", label="Requirements", definition="A statement of a need or capability.", concept_kind="ABSTRACTION", synonyms=["requirement"], scope_description="Software product and system context.", applicability="Use only with context.", evidence_refs=["se:EVD-001"], claim_refs=["se:CLM-001"])],
        "knowledge_claims": [
            rec("se:CLM-001", "PENDING_REVIEW", statement="The taxonomy identifies requirements as a knowledge area.", claim_kind="TAXONOMY_STATEMENT", scope_description="The inspected source taxonomy.", applicability="Use as a reference mapping.", assumptions=[], exceptions=[], limitations=["Not universal."], uncertainty=[], evidence_refs=["se:EVD-001"], review_status="PENDING"),
            rec("se:CLM-002", "PENDING_REVIEW", statement="A later edition may replace an earlier taxonomy statement.", claim_kind="LIMITATION", scope_description="Versioned source comparison.", applicability="When editions differ.", assumptions=[], exceptions=[], limitations=[], uncertainty=[], evidence_refs=["se:EVD-001"], review_status="PENDING"),
        ],
        "principles": [rec("se:PRI-001", label="Context matters", statement="Engineering guidance is applied in context.", rationale="Different systems have different constraints.", scope_description="Engineering practice.", applicability="Evaluate project risk.", exceptions=["Mandatory contractual constraints remain binding."], limitations=["Illustrative fixture."], evidence_refs=["se:EVD-001"], claim_refs=["se:CLM-001"])],
        "practices": [rec("se:PRA-001", label="Review requirements", intent="Detect ambiguity and omissions.", description="Conduct structured review.", preconditions=["A candidate baseline exists."], steps=["Review each requirement."], expected_outcomes=["Recorded findings."], risks=["Review bias."], scope_description="Requirements work.", applicability="Adapt to project size.", evidence_refs=["se:EVD-001"], claim_refs=["se:CLM-001"])],
        "patterns": [rec("se:PAT-001", label="Incremental refinement", problem="Large requirements sets contain uncertainty.", context="Evolving products.", forces=["Feedback speed", "Traceability"], solution="Refine in increments.", consequences=["Earlier feedback."], tradeoffs=["Coordination overhead."], scope_description="Requirements and delivery.", evidence_refs=["se:EVD-001"], case_refs=["se:CASE-001"])],
        "anti_patterns": [rec("se:ANT-001", label="Big-bang specification", symptoms=["Late discovery of ambiguity."], causes=["No incremental feedback."], consequences=["Expensive rework."], remediation="Use staged review and validation.", scope_description="Requirements work.", evidence_refs=["se:EVD-001"], case_refs=["se:CASE-001"])],
        "decision_models": [rec("se:DM-001", decision_question="How much requirements detail is appropriate?", options=["Lightweight", "Formal"], criteria=["Risk", "Change rate", "Regulatory context"], constraints=["Time", "Stakeholder availability"], procedure="Score options against context-specific criteria.", tradeoffs=["Cost versus assurance."], scope_description="Project planning.", evidence_refs=["se:EVD-001"], claim_refs=["se:CLM-001"])],
        "cases": [rec("se:CASE-001", label="Illustrative review case", case_kind="CASE_STUDY", context="Synthetic scenario for validator coverage.", problem="Ambiguous requirement.", intervention="Peer review.", outcome="Ambiguity recorded.", lessons=["Review is context-sensitive."], synthetic=False, evidence_refs=["se:EVD-001"])],
        "counterexamples": [rec("se:COUNTER-001", label="Synthetic counterexample", case_kind="SYNTHETIC_TEST", context="Validator-only negative example.", problem="A counterexample challenges an overbroad rule.", intervention="Construct a boundary condition.", outcome="Rule requires qualification.", lessons=["Do not generalize."], synthetic=True, evidence_refs=[], counterexample_to_refs=["se:CLM-001"], contrary_observation="A constrained project may intentionally omit this practice.")],
        "review_decisions": [rec("se:REV-001", "ACTIVE", target_ref="se:CLM-001", target_record_version="0.1.0", target_digest_sha256="0"*64, reviewer_role="HUMAN_DOMAIN_REVIEWER", reviewer_ref="reviewer:test-fixture", decision="DEFER", rationale="Fixture only; no human acceptance is asserted.", reviewed_on="2026-11-05", review_policy_ref="SE-REVIEW-POLICY-0.1")],
        "coverage_gaps": [rec("se:GAP-001", label="Detailed practice coverage gap", target_scope_ref="se:NODE-SSDF", gap_kind="SOURCE_GAP", description="Detailed practices are not yet extracted.", severity="MEDIUM", coverage_status="OPEN", impact="Do not claim full secure-development coverage.", resolution_notes=["Inspect permitted source material."], evidence_refs=[])],
        "relationship_assertions": [],
    }
    def rel(rid, subject, predicate, obj, evidence=None, qualifiers=None):
        return rec(rid, "PENDING_REVIEW", subject_ref=subject, predicate=predicate, object_ref=obj, qualifiers=qualifiers or [], context="Synthetic structural test context.", rationale="Synthetic fixture relationship; not accepted knowledge.", evidence_refs=evidence or [])
    graph["relationship_assertions"] = [
        rel("se:REL-001", "se:CLM-001", "SUPPORTED_BY", "se:EVD-001"),
        rel("se:REL-002", "se:CLM-001", "APPLIES_TO", "se:NODE-REQ", ["se:EVD-001"], ["taxonomy scope"]),
        rel("se:REL-003", "se:SUB-REQ", "PART_OF", "se:DOMAIN-SE"),
        rel("se:REL-004", "se:PRA-001", "DEPENDS_ON", "se:CON-REQ", ["se:EVD-001"]),
        rel("se:REL-005", "se:PRA-001", "IMPLEMENTS", "se:PRI-001", ["se:EVD-001"], ["operationalization"]),
        rel("se:REL-006", "se:PAT-001", "ILLUSTRATED_BY", "se:CASE-001"),
        rel("se:REL-007", "se:CLM-001", "CONTRADICTED_BY", "se:COUNTER-001", [], ["synthetic counterexample only"]),
        rel("se:REL-008", "se:CLM-001", "QUALIFIED_BY", "se:EVD-001", [], ["applicability boundary"]),
        rel("se:REL-009", "se:CLM-001", "VALIDATED_BY", "se:REV-001", [], ["review deferred"]),
        rel("se:REL-010", "se:CON-REQ", "DERIVED_FROM", "se:EVD-001", ["se:EVD-001"]),
        rel("se:REL-011", "se:CLM-002", "SUPERSEDES", "se:CLM-001", ["se:EVD-001"], ["later revision"]),
        rel("se:REL-012", "se:NODE-SSDF", "CROSS_CUTS", "se:NODE-REQ", ["se:EVD-001"], ["security across requirements"]),
        rel("se:REL-013", "se:NODE-SSDF", "OVERLAPS", "se:NODE-REQ", ["se:EVD-001"], ["partial scope overlap"]),
        rel("se:REL-014", "se:SUB-REQ", "SPECIALIZES", "se:NODE-REQ", ["se:EVD-001"], ["subdomain mapping"]),
        rel("se:REL-015", "se:DOMAIN-SE", "BOUNDED_BY", "se:NODE-REQ", [], ["illustrative boundary"]),
        rel("se:REL-016", "se:CON-REQ", "RELATED_TO", "se:PAT-001", ["se:EVD-001"], ["illustrative conceptual relation"]),
    ]
    return graph


def run_self_tests(catalog, rules):
    base = positive_fixture()
    cases = []

    def expect(name, mutate, expected_code):
        candidate = copy.deepcopy(base)
        mutate(candidate)
        errors = validate_graph(candidate, catalog, rules)
        passed = any(error.startswith(expected_code + ":") for error in errors)
        cases.append((name, passed, errors))

    baseline_errors = validate_graph(base, catalog, rules)
    cases.append(("positive-full-entity-coverage", not baseline_errors, baseline_errors))
    expect("duplicate-id", lambda g: g["concepts"].append(copy.deepcopy(g["concepts"][0])), "DUPLICATE_IDENTIFIER")
    expect("dangling-reference", lambda g: g["evidence_records"][0].update(source_locator_ref="se:LOC-MISSING"), "DANGLING_REFERENCE")
    expect("wrong-endpoint-class", lambda g: g["relationship_assertions"][0].update(object_ref="se:CON-REQ"), "WRONG_OBJECT_CLASS_FOR_PREDICATE")
    expect("unknown-predicate", lambda g: g["relationship_assertions"][0].update(predicate="UNDECLARED_EDGE"), "UNKNOWN_PREDICATE")
    expect("missing-evidence", lambda g: g["relationship_assertions"][3].update(evidence_refs=[]), "MISSING_REQUIRED_EVIDENCE")
    expect("missing-qualifier", lambda g: g["relationship_assertions"][1].update(qualifiers=[]), "MISSING_REQUIRED_QUALIFIER")
    expect("accepted-synthetic", lambda g: g["counterexamples"][0].update(lifecycle_status="ACCEPTED"), "ACCEPTED_SYNTHETIC_RECORD")
    expect("accepted-without-review", lambda g: g["concepts"][0].update(lifecycle_status="ACCEPTED"), "ACCEPTED_WITHOUT_HUMAN_REVIEW")
    expect("duplicate-alias", lambda g: g["concepts"][0].update(aliases=["se:LEGACY-1"]) or g["knowledge_claims"][0].update(aliases=["se:LEGACY-1"]), "DUPLICATE_ALIAS")
    expect("supersedes-cycle", lambda g: g["relationship_assertions"].append({
        **g["relationship_assertions"][10],
        "id": "se:REL-CYCLE",
        "subject_ref": "se:CLM-001",
        "object_ref": "se:CLM-002",
    }), "SUPERSEDES_CYCLE")
    def part_of_cycle(g):
        g["subdomains"].append({
            "id": "se:SUB-CHILD",
            "record_version": "0.1.0",
            "created_on": "2026-11-05",
            "lifecycle_status": "PROPOSED",
            "label": "Child",
            "definition": "A test child subdomain.",
            "domain_ref": "se:DOMAIN-SE",
        })
        g["relationship_assertions"].extend([
            {**g["relationship_assertions"][2], "id": "se:REL-PART-A", "subject_ref": "se:SUB-REQ", "object_ref": "se:SUB-CHILD"},
            {**g["relationship_assertions"][2], "id": "se:REL-PART-B", "subject_ref": "se:SUB-CHILD", "object_ref": "se:SUB-REQ"},
        ])
    expect("part-of-cycle", part_of_cycle, "PART_OF_CYCLE")
    expect("duplicate-source-locator", lambda g: g["source_locators"].append({
        **g["source_locators"][0], "id": "se:LOC-DUP",
    }), "DUPLICATE_SOURCE_LOCATOR")
    expect("review-version-mismatch", lambda g: g["review_decisions"][0].update(target_record_version="9.9.9"), "REVIEW_VERSION_MISMATCH")
    passed = sum(1 for _, ok, _ in cases if ok)
    for name, ok, errors in cases:
        print(f"{'PASS' if ok else 'FAIL'} {name}")
        if not ok:
            for error in errors:
                print(f"  {error}")
    print(f"SELF_TESTS={passed}/{len(cases)}")
    return 0 if passed == len(cases) else 1


def main():
    parser = argparse.ArgumentParser(description=__doc__)
    parser.add_argument("--catalog", type=Path, default=DEFAULT_CATALOG)
    parser.add_argument("--rules", type=Path, default=DEFAULT_RULES)
    parser.add_argument("--input", type=Path)
    parser.add_argument("--self-test", action="store_true")
    parser.add_argument("--emit-fixture", type=Path)
    args = parser.parse_args()
    try:
        catalog = load_yaml(args.catalog)
        rules = load_yaml(args.rules)
        if args.emit_fixture:
            args.emit_fixture.parent.mkdir(parents=True, exist_ok=True)
            with args.emit_fixture.open("w", encoding="utf-8", newline="\n") as handle:
                yaml.safe_dump(positive_fixture(), handle, allow_unicode=True, sort_keys=False)
            print(f"FIXTURE_WRITTEN={args.emit_fixture}")
        if args.self_test:
            return run_self_tests(catalog, rules)
        if args.input:
            graph = load_yaml(args.input)
            errors = validate_graph(graph, catalog, rules)
            if errors:
                for error in errors:
                    print(error, file=sys.stderr)
                print(f"GRAPH_VALIDATION=FAIL errors={len(errors)}")
                return 1
            print("GRAPH_VALIDATION=PASS")
            return 0
        if not args.emit_fixture:
            parser.error("choose --self-test, --input or --emit-fixture")
        return 0
    except (OSError, yaml.YAMLError, ValidationError) as exc:
        print(f"VALIDATOR_ERROR: {exc}", file=sys.stderr)
        return 2


if __name__ == "__main__":
    raise SystemExit(main())
