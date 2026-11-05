#!/usr/bin/env python3
"""Deterministic experimental graph-integrity validator for SEK-05.

This is documentation tooling, not a Tally runtime dependency. LinkML validates
record shape; this program validates graph-wide identity, provenance, lifecycle,
typed relationship endpoints, and exact human-review bindings.
"""
from __future__ import annotations

import argparse
import copy
import hashlib
import json
import re
import sys
from collections import defaultdict
from pathlib import Path

import yaml


ROOT = Path(__file__).resolve().parent
DEFAULT_CATALOG = ROOT / "ENTITY_CATALOG.yaml"
DEFAULT_RULES = ROOT / "RELATIONSHIP_CATALOG.yaml"
ID_PATTERN = re.compile(r"^se:[A-Z0-9][A-Z0-9._-]*$")
VERSION_PATTERN = re.compile(r"^\d+\.\d+\.\d+$")
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
    "RelationshipAssertion",
}


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


def _ids(value):
    if value is None:
        return []
    if isinstance(value, str):
        return [value]
    if isinstance(value, list):
        return value
    return []


def canonical_record_digest(record):
    """SHA-256 over canonical UTF-8 JSON for one exact record revision."""
    payload = json.dumps(
        record,
        sort_keys=True,
        ensure_ascii=False,
        separators=(",", ":"),
    )
    return hashlib.sha256(payload.encode("utf-8")).hexdigest()


def _error(errors, code, detail):
    errors.append(f"{code}: {detail}")


def validate_graph(graph, entity_catalog, relationship_catalog):
    errors = []
    if not isinstance(graph, dict):
        return ["ROOT_NOT_MAPPING: graph root must be a mapping"]

    for field in ("id", "graph_version", "generated_on", "domain_ref"):
        if not graph.get(field):
            _error(errors, "MISSING_GRAPH_ROOT_FIELD", field)
    if graph.get("id") and not ID_PATTERN.fullmatch(str(graph["id"])):
        _error(errors, "INVALID_GRAPH_ID", str(graph["id"]))
    if graph.get("graph_version") and not VERSION_PATTERN.fullmatch(str(graph["graph_version"])):
        _error(errors, "INVALID_GRAPH_VERSION", str(graph["graph_version"]))

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

    allowed_lifecycle = set(
        entity_catalog.get("identity_contract", {}).get("lifecycle", {}).get("values", [])
    )
    aliases = {}
    for class_name, record_id, record in records:
        for field in ("record_version", "lifecycle_status", "created_on"):
            if not record.get(field):
                _error(errors, "MISSING_COMMON_FIELD", f"{record_id}.{field}")
        if not VERSION_PATTERN.fullmatch(str(record.get("record_version", ""))):
            _error(errors, "INVALID_RECORD_VERSION", record_id)
        if allowed_lifecycle and record.get("lifecycle_status") not in allowed_lifecycle:
            _error(errors, "INVALID_LIFECYCLE_STATUS", f"{record_id}: {record.get('lifecycle_status')!r}")
        if record.get("created_on") and not re.fullmatch(r"\d{4}-\d{2}-\d{2}", str(record["created_on"])):
            _error(errors, "INVALID_CREATED_ON", record_id)
        for alias in _ids(record.get("aliases")):
            if not isinstance(alias, str) or not alias.strip():
                _error(errors, "INVALID_ALIAS", record_id)
            elif alias in aliases or alias in entities:
                _error(errors, "DUPLICATE_ALIAS", alias)
            else:
                aliases[alias] = record_id
        if class_name in KNOWLEDGE_CLASSES and record.get("lifecycle_status") == "ACCEPTED":
            if record.get("synthetic") is True:
                _error(errors, "ACCEPTED_SYNTHETIC_RECORD", record_id)

    def ref(record_id, field, expected, required=False):
        record = entities.get(record_id, {})
        value = record.get(field)
        if required and not value:
            _error(errors, "MISSING_REQUIRED_REFERENCE", f"{record_id}.{field}")
            return
        if value is None:
            return
        values = _ids(value)
        if not values:
            _error(errors, "INVALID_REFERENCE_VALUE", f"{record_id}.{field}")
            return
        for target in values:
            if not isinstance(target, str) or target not in entities:
                _error(errors, "DANGLING_REFERENCE", f"{record_id}.{field}->{target}")
            elif expected and classes[target] not in expected:
                _error(errors, "WRONG_REFERENCE_CLASS", f"{record_id}.{field}->{target} is {classes[target]}, expected {sorted(expected)}")

    # Any explicit evidence_refs field is type-checked, regardless of entity family.
    for class_name, record_id, record in records:
        if "evidence_refs" in record:
            ref(record_id, "evidence_refs", {"Evidence"})
        if class_name == "Subdomain":
            ref(record_id, "domain_ref", {"Domain"}, True)
        elif class_name == "TaxonomyNode":
            ref(record_id, "view_ref", {"TaxonomyView"}, True)
        elif class_name == "SourceLocator":
            ref(record_id, "source_ref", {"Source"}, True)
        elif class_name == "Evidence":
            ref(record_id, "source_locator_ref", {"SourceLocator"}, True)
            evidence_type = record.get("evidence_type")
            if record.get("synthetic") is True and evidence_type != "SYNTHETIC_FIXTURE":
                _error(errors, "SYNTHETIC_EVIDENCE_TYPE_MISMATCH", record_id)
            if evidence_type == "SYNTHETIC_FIXTURE" and record.get("synthetic") is not True:
                _error(errors, "SYNTHETIC_EVIDENCE_FLAG_MISSING", record_id)
        elif class_name == "Concept":
            ref(record_id, "claim_refs", {"KnowledgeClaim"})
        elif class_name in {"Principle", "Practice", "DecisionModel"}:
            ref(record_id, "claim_refs", {"KnowledgeClaim"})
        elif class_name in {"Pattern", "AntiPattern"}:
            ref(record_id, "case_refs", {"Case", "Counterexample"})
        elif class_name == "Counterexample":
            ref(record_id, "counterexample_to_refs", KNOWLEDGE_CLASSES, True)
        elif class_name == "ReviewDecision":
            ref(record_id, "target_ref", set(COLLECTIONS.values()) - {"ReviewDecision"}, True)
        elif class_name == "CoverageGap":
            ref(record_id, "target_scope_ref", {"Domain", "Subdomain", "TaxonomyView", "TaxonomyNode"}, True)
        elif class_name == "RelationshipAssertion":
            ref(record_id, "subject_ref", set(COLLECTIONS.values()), True)
            ref(record_id, "object_ref", set(COLLECTIONS.values()), True)

    root_domain = graph.get("domain_ref")
    if root_domain not in entities or classes.get(root_domain) != "Domain":
        _error(errors, "INVALID_GRAPH_DOMAIN", str(root_domain))

    # A locator is unique within its source; evidence has exactly one source locator.
    locator_keys = {}
    for class_name, record_id, record in records:
        if class_name == "SourceLocator":
            key = (record.get("source_ref"), record.get("locator"))
            if key in locator_keys:
                _error(errors, "DUPLICATE_SOURCE_LOCATOR", f"{record_id} duplicates {locator_keys[key]}")
            locator_keys[key] = record_id
        elif class_name == "Evidence":
            locator = entities.get(record.get("source_locator_ref"), {})
            source_id = locator.get("source_ref")
            if not source_id or classes.get(source_id) != "Source":
                _error(errors, "BROKEN_EVIDENCE_SOURCE_CHAIN", record_id)

    predicate_rules = {
        item["predicate"]: item
        for item in relationship_catalog.get("predicates", [])
    }

    def evidence_is_usable(evidence_id, owner_id):
        evidence = entities.get(evidence_id)
        if not evidence or classes.get(evidence_id) != "Evidence":
            _error(errors, "DANGLING_EVIDENCE", f"{owner_id}->{evidence_id}")
            return False
        if evidence.get("synthetic") is True or evidence.get("evidence_type") == "SYNTHETIC_FIXTURE":
            _error(errors, "ACCEPTED_SYNTHETIC_EVIDENCE", f"{owner_id}->{evidence_id}")
            return False
        if evidence.get("extraction_status") != "REVIEWED":
            _error(errors, "EVIDENCE_NOT_REVIEWED", f"{owner_id}->{evidence_id}")
            return False
        locator_id = evidence.get("source_locator_ref")
        locator = entities.get(locator_id, {})
        source_id = locator.get("source_ref")
        if classes.get(locator_id) != "SourceLocator" or classes.get(source_id) != "Source":
            _error(errors, "BROKEN_EVIDENCE_SOURCE_CHAIN", f"{owner_id}->{evidence_id}")
            return False
        return True

    def usable_evidence_refs(owner_id, evidence_ids):
        values = _ids(evidence_ids)
        if not values:
            _error(errors, "ACCEPTED_WITHOUT_EVIDENCE", owner_id)
            return False
        valid = True
        for evidence_id in values:
            valid = evidence_is_usable(evidence_id, owner_id) and valid
        return valid

    # Validate the exact review binding for every ReviewDecision.
    accepted_review_bindings = set()
    for class_name, record_id, record in records:
        if class_name != "ReviewDecision":
            continue
        target_id = record.get("target_ref")
        target = entities.get(target_id)
        if not target:
            continue
        if record.get("target_record_version") != target.get("record_version"):
            _error(errors, "REVIEW_VERSION_MISMATCH", record_id)
        digest = record.get("target_digest_sha256", "")
        if not SHA256_PATTERN.fullmatch(str(digest)):
            _error(errors, "INVALID_REVIEW_DIGEST", record_id)
        elif digest != canonical_record_digest(target):
            _error(errors, "REVIEW_DIGEST_MISMATCH", record_id)
        if record.get("decision") == "ACCEPT":
            if target.get("lifecycle_status") != "ACCEPTED":
                _error(errors, "REVIEW_ACCEPT_TARGET_NOT_ACCEPTED", record_id)
            if classes.get(target_id) == "KnowledgeClaim" and target.get("review_status") != "ACCEPTED":
                _error(errors, "CLAIM_REVIEW_STATUS_MISMATCH", target_id)
            if (
                record.get("target_record_version") == target.get("record_version")
                and digest == canonical_record_digest(target)
                and target.get("lifecycle_status") == "ACCEPTED"
            ):
                accepted_review_bindings.add((target_id, target.get("record_version"), digest))

    # Accepted records must have reviewed, non-synthetic evidence and an exact human decision.
    accepted_ids = set()
    for class_name, record_id, record in records:
        if class_name not in KNOWLEDGE_CLASSES or record.get("lifecycle_status") != "ACCEPTED":
            continue
        accepted_ids.add(record_id)
        if class_name == "CoverageGap":
            if record.get("coverage_status") != "ACCEPTED_LIMITATION" or not _ids(record.get("resolution_notes")):
                _error(errors, "ACCEPTED_GAP_WITHOUT_DISPOSITION", record_id)
            if _ids(record.get("evidence_refs")):
                usable_evidence_refs(record_id, record.get("evidence_refs"))
        elif class_name in {"Pattern", "AntiPattern"} and not _ids(record.get("evidence_refs")):
            case_ids = _ids(record.get("case_refs"))
            if not case_ids:
                _error(errors, "ACCEPTED_WITHOUT_EVIDENCE_OR_CASE", record_id)
            for case_id in case_ids:
                case = entities.get(case_id, {})
                if case.get("synthetic") is True:
                    _error(errors, "ACCEPTED_SYNTHETIC_CASE", f"{record_id}->{case_id}")
                else:
                    usable_evidence_refs(record_id, case.get("evidence_refs"))
        else:
            usable_evidence_refs(record_id, record.get("evidence_refs"))
        if class_name == "KnowledgeClaim" and record.get("review_status") != "ACCEPTED":
            _error(errors, "CLAIM_REVIEW_STATUS_MISMATCH", record_id)
        expected_binding = (
            record_id,
            record.get("record_version"),
            canonical_record_digest(record),
        )
        if expected_binding not in accepted_review_bindings:
            _error(errors, "ACCEPTED_WITHOUT_HUMAN_REVIEW", record_id)

    # Validate every relationship against the controlled endpoint/predicate catalog.
    edge_signatures = set()
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
        if not record.get("context") or not record.get("rationale"):
            _error(errors, "MISSING_RELATION_CONTEXT", record_id)
        if subject_id == object_id:
            _error(errors, "SELF_REFERENTIAL_RELATIONSHIP", record_id)
        if predicate == "OVERLAPS" and isinstance(subject_id, str) and isinstance(object_id, str) and subject_id >= object_id:
            _error(errors, "OVERLAPS_NONCANONICAL_PAIR", record_id)
        signature = (
            subject_id,
            predicate,
            object_id,
            tuple(sorted(_ids(record.get("qualifiers")))),
            record.get("context"),
        )
        if signature in edge_signatures:
            _error(errors, "DUPLICATE_RELATIONSHIP_SIGNATURE", record_id)
        edge_signatures.add(signature)
        if predicate in {"PART_OF", "SUPERSEDES"}:
            graph_edges[predicate].append((subject_id, object_id))
        if rule.get("same_class_required") and subject_class != object_class:
            _error(errors, "SUPERSEDES_CLASS_MISMATCH", record_id)

        if record.get("lifecycle_status") == "ACCEPTED":
            evidence_ids = _ids(record.get("evidence_refs"))
            if not evidence_ids and rule.get("object_is_evidence") and object_class == "Evidence":
                evidence_ids = [object_id]
            elif not evidence_ids and rule.get("object_carries_provenance") and object_class in {"Case", "Counterexample"}:
                object_record = entities.get(object_id, {})
                if object_record.get("synthetic") is True:
                    _error(errors, "ACCEPTED_SYNTHETIC_CASE", f"{record_id}->{object_id}")
                evidence_ids = _ids(object_record.get("evidence_refs"))
            if not evidence_ids:
                _error(errors, "ACCEPTED_RELATIONSHIP_WITHOUT_EVIDENCE", record_id)
            else:
                usable_evidence_refs(record_id, evidence_ids)

    # Reject cycles in containment and supersession relationships.
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
        "domain_ref": "se:DOMAIN-SOFTWARE-ENGINEERING",
        "domains": [
            rec("se:DOMAIN-SOFTWARE-ENGINEERING", label="Software Engineering", definition="A bounded engineering discipline whose coverage is represented through multiple source and classification views.", scope_inclusions=["professional knowledge", "curriculum comparison", "secure-development guidance"], scope_exclusions=["the entirety of computer science"], boundary_notes=["No single source is treated as a complete ontology."], evidence_refs=["se:EVD-SWEBOK-KA", "se:EVD-CS2023-SE", "se:EVD-NIST-SSDF"])
        ],
        "subdomains": [
            rec("se:SUB-REQUIREMENTS-ENGINEERING", label="Requirements Engineering", definition="Engineering work concerned with identifying, analyzing and specifying software requirements.", domain_ref="se:DOMAIN-SOFTWARE-ENGINEERING", boundary_notes=["Its classification depends on the selected source view."], evidence_refs=["se:EVD-SWEBOK-KA"])
        ],
        "taxonomy_views": [
            rec("se:VIEW-DISCIPLINE", label="Professional discipline view", view_kind="DISCIPLINE", purpose="Represent the SWEBOK professional knowledge-area taxonomy.", coverage_state="PARTIAL", limitations=["SWEBOK states it does not present the entire body of knowledge."], evidence_refs=["se:EVD-SWEBOK-KA"]),
            rec("se:VIEW-CURRICULUM", label="Computer science curriculum view", view_kind="CURRICULUM", purpose="Represent the CS2023 curriculum knowledge model.", coverage_state="PARTIAL", limitations=["Curriculum classification is not professional-domain completeness."], evidence_refs=["se:EVD-CS2023-SE"]),
            rec("se:VIEW-SECURITY", label="Secure development view", view_kind="SECURITY", purpose="Represent the NIST SSDF v1.1 practice groups.", coverage_state="PARTIAL", limitations=["The SSDF is a recommendation framework, not a universal taxonomy."], evidence_refs=["se:EVD-NIST-SSDF"])
        ],
        "taxonomy_nodes": [
            rec("se:NODE-SWEBOK-ENG", label="SWEBOK Software Engineering Knowledge Areas", definition="The first fifteen SWEBOK v4.0a knowledge areas, distinguished from three foundation knowledge areas.", node_kind="KNOWLEDGE_AREA", view_ref="se:VIEW-DISCIPLINE", evidence_refs=["se:EVD-SWEBOK-KA"]),
            rec("se:NODE-SWEBOK-REQ", label="Software Requirements", definition="A Software Engineering knowledge area in SWEBOK v4.0a.", node_kind="KNOWLEDGE_AREA", view_ref="se:VIEW-DISCIPLINE", evidence_refs=["se:EVD-SWEBOK-KA"]),
            rec("se:NODE-CS2023-SE", label="Software Engineering (CS2023)", definition="A curriculum knowledge area represented by nine knowledge units.", node_kind="KNOWLEDGE_AREA", view_ref="se:VIEW-CURRICULUM", evidence_refs=["se:EVD-CS2023-SE"]),
            rec("se:NODE-SSDF-GROUPS", label="SSDF Practice Groups", definition="The four practice groups in NIST SP 800-218 v1.1.", node_kind="PRACTICE_GROUP", view_ref="se:VIEW-SECURITY", evidence_refs=["se:EVD-NIST-SSDF"])
        ],
        "sources": [
            rec("se:SRC-SWEBOK-GUIDE-4-0A", "ACTIVE", title="Guide to the Software Engineering Body of Knowledge (SWEBOK Guide), Version 4.0a", source_class="PROFESSIONAL_BOK", source_roles=["PRIMARY", "SECONDARY"], organization="IEEE Computer Society", authors=["Hironori Washizaki"], edition="Version 4.0a", publication_date="2026-08", official_url="https://ieeecs-media.computer.org/media/education/swebok/swebok-v4.pdf", stable_identifier="ISBN 978-0-7695-0000-3", authority_fit=["Professional knowledge-area taxonomy"], authority_limitations=["Not the entire body of knowledge", "Citations are not comprehensive"], rights_handling="Bibliographic facts and concise original paraphrases only; no substantial reproduction.", verification_state="CONTENT_PARTIAL_VERIFIED"),
            rec("se:SRC-CS2023", "ACTIVE", title="Computer Science Curricula 2023", source_class="CURRICULUM_FRAMEWORK", source_roles=["PRIMARY"], organization="ACM; IEEE Computer Society; AAAI", authors=["Amruth N. Kumar", "Rajendra K. Raj", "Sherif G. Aly"], edition="Final report", publication_date="2023", official_url="https://csed.acm.org/", stable_identifier="DOI 10.1145/3664191", authority_fit=["CS curriculum knowledge-area and knowledge-unit classification"], authority_limitations=["Curriculum comparator, not professional domain completeness"], rights_handling="Concise original paraphrases; respect report permissions.", verification_state="CONTENT_PARTIAL_VERIFIED"),
            rec("se:SRC-NIST-SP-800-218-V1-1", "ACTIVE", title="Secure Software Development Framework (SSDF) Version 1.1: Recommendations for Mitigating the Risk of Software Vulnerabilities", source_class="GOVERNMENT_FRAMEWORK", source_roles=["PRIMARY"], organization="National Institute of Standards and Technology", authors=["Murugiah Souppaya", "Karen Scarfone", "Donna Dodson"], edition="Version 1.1", publication_date="2022-02-03", official_url="https://csrc.nist.gov/pubs/sp/800/218/final", stable_identifier="NIST SP 800-218; DOI 10.6028/NIST.SP.800-218", authority_fit=["Government-published secure-development recommendation framework"], authority_limitations=["Not an international standard", "Applicability must be checked for each claim"], rights_handling="Original paraphrase and official citation; no substantial reproduction.", verification_state="CONTENT_PARTIAL_VERIFIED"),
            rec("se:SRC-SEK05-TEST-FIXTURE", "ACTIVE", title="SEK-05 synthetic validator fixture", source_class="OTHER", source_roles=["CONTEXT"], organization="Tally v2.1.1 SEK-05 experiment", authors=[], edition="0.1.0", publication_date="2026-11-05", stable_identifier="SEK-05-FIXTURE-0.1.0", authority_fit=["Structural negative/positive testing only"], authority_limitations=["Not a domain source and cannot support accepted knowledge"], rights_handling="Original synthetic test content.", verification_state="NOT_VERIFIED")
        ],
        "source_locators": [
            rec("se:LOC-SWEBOK-KA", "ACTIVE", source_ref="se:SRC-SWEBOK-GUIDE-4-0A", locator_type="TABLE", locator="PDF page index 39; printed page xxxix; Introduction to the Guide, Table I.1", verified_content_summary="18 knowledge areas; the first 15 are Software Engineering KAs and the last three are foundation KAs.", verification_state="LOCATOR_VERIFIED", accessed_on="2026-11-04"),
            rec("se:LOC-CS2023-SE", "ACTIVE", source_ref="se:SRC-CS2023", locator_type="TABLE", locator="Body of Knowledge PDF page index 0; front-matter Knowledge Areas table", verified_content_summary="17 knowledge areas; Software Engineering is one area with nine knowledge units.", verification_state="LOCATOR_VERIFIED", accessed_on="2026-11-04"),
            rec("se:LOC-NIST-SSDF", "ACTIVE", source_ref="se:SRC-NIST-SP-800-218-V1-1", locator_type="SECTION", locator="NIST SP 800-218 PDF page index 12; printed page 4; Section 2", verified_content_summary="Four SSDF practice groups: PO, PS, PW and RV.", verification_state="LOCATOR_VERIFIED", accessed_on="2026-11-04"),
            rec("se:LOC-NIST-RISK", "ACTIVE", source_ref="se:SRC-NIST-SP-800-218-V1-1", locator_type="SECTION", locator="NIST SP 800-218 PDF page index 11; printed page 3; Section 1", verified_content_summary="Risk, cost, feasibility and applicability guide practice selection; SSDF is not a checklist.", verification_state="LOCATOR_VERIFIED", accessed_on="2026-11-04"),
            rec("se:LOC-SEK05-FIXTURE", "ACTIVE", source_ref="se:SRC-SEK05-TEST-FIXTURE", locator_type="OTHER", locator="Synthetic fixture assertions authored for SEK-05 validator self-tests", verified_content_summary="Synthetic relation and case examples used only to exercise graph constraints.", verification_state="NOT_VERIFIED", accessed_on="2026-11-05")
        ],
        "evidence_records": [
            rec("se:EVD-SWEBOK-KA", "PENDING_REVIEW", source_locator_ref="se:LOC-SWEBOK-KA", evidence_type="DIRECT_SOURCE_PARAPHRASE", statement="SWEBOK v4.0a lists 18 Knowledge Areas and classifies the first 15 as Software Engineering KAs and the final three as foundation KAs.", context="The Guide's own taxonomy, not an independently exhaustive classification.", method="Inspected official PDF Table I.1.", result="The source lists 18 KAs with a 15/3 distinction.", limitations=["SWEBOK is not the entire body of knowledge."], uncertainty=[], extraction_status="EXTRACTED_PENDING_REVIEW", synthetic=False),
            rec("se:EVD-CS2023-SE", "PENDING_REVIEW", source_locator_ref="se:LOC-CS2023-SE", evidence_type="DIRECT_SOURCE_PARAPHRASE", statement="CS2023 lists 17 Knowledge Areas; Software Engineering is one area and is represented by nine Knowledge Units.", context="Curriculum classification, not professional-domain completeness.", method="Inspected official Body of Knowledge table.", result="The source lists the SE area and nine KUs.", limitations=["Specific to the CS2023 curriculum model."], uncertainty=[], extraction_status="EXTRACTED_PENDING_REVIEW", synthetic=False),
            rec("se:EVD-NIST-SSDF", "PENDING_REVIEW", source_locator_ref="se:LOC-NIST-SSDF", evidence_type="DIRECT_SOURCE_PARAPHRASE", statement="NIST SP 800-218 v1.1 organizes SSDF practices into four groups: PO, PS, PW and RV.", context="Section 2 practice-group structure.", method="Inspected official NIST PDF Section 2.", result="Four practice groups are identified.", limitations=["Not a universal security taxonomy."], uncertainty=[], extraction_status="EXTRACTED_PENDING_REVIEW", synthetic=False),
            rec("se:EVD-NIST-RISK", "PENDING_REVIEW", source_locator_ref="se:LOC-NIST-RISK", evidence_type="DIRECT_SOURCE_PARAPHRASE", statement="NIST says practice selection should consider risk, cost, feasibility and applicability and that SSDF is not intended as a checklist.", context="Section 1 risk-based adoption guidance.", method="Inspected official NIST PDF Section 1.", result="The framework calls for context-specific practice selection.", limitations=["Requires organization- and use-case-specific judgment."], uncertainty=[], extraction_status="EXTRACTED_PENDING_REVIEW", synthetic=False),
            rec("se:EVD-SEK05-TEST", "PENDING_REVIEW", source_locator_ref="se:LOC-SEK05-FIXTURE", evidence_type="SYNTHETIC_FIXTURE", statement="Synthetic relation fixture used only to test typed graph validation.", context="No claim about the Software Engineering domain is made.", method="Authored structural fixture.", result="Provides a test-only evidence resource.", limitations=["Must never support accepted corpus knowledge."], uncertainty=[], extraction_status="EXTRACTED_PENDING_REVIEW", synthetic=True)
        ],
        "concepts": [
            rec("se:CON-SOFTWARE-REQUIREMENTS", label="Software Requirements", definition="A defined knowledge topic in software engineering.", concept_kind="TERM", synonyms=["requirements"], scope_description="Software engineering knowledge-area context.", applicability="Interpret within the selected source view.", evidence_refs=["se:EVD-SWEBOK-KA"], claim_refs=["se:CLM-SWEBOK-KA"])
        ],
        "knowledge_claims": [
            rec("se:CLM-SWEBOK-KA", "PENDING_REVIEW", statement="SWEBOK v4.0a lists 18 Knowledge Areas; the first 15 are Software Engineering KAs and the final three are foundation KAs.", claim_kind="TAXONOMY_STATEMENT", scope_description="SWEBOK Guide v4.0a taxonomy.", applicability="Use as a reference mapping, not the only valid taxonomy.", assumptions=[], exceptions=[], limitations=["The Guide does not present the entire body of knowledge."], uncertainty=[], evidence_refs=["se:EVD-SWEBOK-KA"], review_status="PENDING"),
            rec("se:CLM-CS2023-SE", "PENDING_REVIEW", statement="CS2023 represents Software Engineering as one of 17 Knowledge Areas with nine Knowledge Units.", claim_kind="TAXONOMY_STATEMENT", scope_description="ACM/IEEE-CS/AAAI CS2023 curriculum model.", applicability="Use as a curriculum comparator.", assumptions=[], exceptions=[], limitations=["Not a professional domain-completeness claim."], uncertainty=[], evidence_refs=["se:EVD-CS2023-SE"], review_status="PENDING"),
            rec("se:CLM-TEST-SUPERSESSION", "PENDING_REVIEW", statement="Synthetic fixture: a later claim revision supersedes an earlier claim in a test graph.", claim_kind="SYNTHESIS", scope_description="SEK-05 validator fixture only.", applicability="Test the SUPERSEDES predicate and cycle detector.", assumptions=[], exceptions=[], limitations=["Synthetic; never accepted knowledge."], uncertainty=[], evidence_refs=["se:EVD-SEK05-TEST"], review_status="PENDING")
        ],
        "principles": [
            rec("se:PRI-CONTEXTUAL-APPLICATION", label="Contextual application", statement="Engineering guidance should be applied in context rather than as a universal checklist.", rationale="NIST's risk-based adoption guidance considers risk, cost, feasibility and applicability.", scope_description="Secure software development guidance.", applicability="Evaluate the organization and use case.", exceptions=["Binding contractual or regulatory obligations remain applicable."], limitations=["This is a contextual principle, not a universal exemption."], evidence_refs=["se:EVD-NIST-RISK"], claim_refs=["se:CLM-CS2023-SE"])
        ],
        "practices": [
            rec("se:PRA-RISK-BASED-SELECTION", label="Risk-based practice selection", intent="Select secure-development practices proportionate to context.", description="Evaluate risk, cost, feasibility and applicability before selecting practices.", preconditions=["A system and risk context are known."], steps=["Assess risk.", "Compare cost and feasibility.", "Record applicability."], expected_outcomes=["A context-specific practice selection."], risks=["Incomplete risk information."], scope_description="Secure software development.", applicability="NIST SP 800-218 v1.1 recommendation context.", evidence_refs=["se:EVD-NIST-RISK"], claim_refs=["se:CLM-CS2023-SE"])
        ],
        "patterns": [
            rec("se:PAT-INCREMENTAL-REFINEMENT", label="Incremental refinement", problem="Large requirements sets contain uncertainty.", context="Evolving products.", forces=["Feedback speed", "Traceability"], solution="Refine in increments.", consequences=["Earlier feedback."], tradeoffs=["Coordination overhead."], scope_description="Requirements and delivery.", evidence_refs=[], case_refs=["se:CASE-TEST-001"])
        ],
        "anti_patterns": [
            rec("se:ANT-BIG-BANG-SPECIFICATION", label="Big-bang specification", symptoms=["Late discovery of ambiguity."], causes=["No incremental feedback."], consequences=["Expensive rework."], remediation="Use staged review and validation.", scope_description="Requirements work.", evidence_refs=[], case_refs=["se:CASE-TEST-001"])
        ],
        "decision_models": [
            rec("se:DM-PRACTICE-SELECTION", decision_question="How much secure-development practice effort is appropriate?", options=["Baseline", "Enhanced"], criteria=["Risk", "Cost", "Feasibility", "Applicability"], constraints=["Time", "Organizational capability"], procedure="Evaluate options against the context-specific risk and feasibility profile.", tradeoffs=["Cost versus assurance."], scope_description="Secure software development.", evidence_refs=["se:EVD-NIST-RISK"], claim_refs=["se:CLM-CS2023-SE"])
        ],
        "cases": [
            rec("se:CASE-TEST-001", label="Synthetic requirements review scenario", case_kind="SYNTHETIC_TEST", context="Synthetic scenario used only to test case references.", problem="Ambiguous requirement.", intervention="Peer review.", outcome="Ambiguity recorded.", lessons=["A fixture is not domain evidence."], synthetic=True, evidence_refs=[])
        ],
        "counterexamples": [
            rec("se:COUNTEREXAMPLE-TEST-001", label="Synthetic boundary counterexample", case_kind="SYNTHETIC_TEST", context="Validator-only negative example.", problem="An overbroad rule is challenged by a boundary condition.", intervention="Construct a constrained project context.", outcome="Rule requires qualification.", lessons=["Do not generalize from a synthetic case."], synthetic=True, evidence_refs=[], counterexample_to_refs=["se:CLM-SWEBOK-KA"], contrary_observation="A constrained project may intentionally omit an optional practice.")
        ],
        "review_decisions": [
            rec("se:REV-CLM-SWEBOK-DEFER", "ACTIVE", target_ref="se:CLM-SWEBOK-KA", target_record_version="0.1.0", target_digest_sha256="0"*64, reviewer_role="HUMAN_DOMAIN_REVIEWER", reviewer_ref="reviewer:test-fixture", decision="DEFER", rationale="Fixture only; no human acceptance is asserted.", reviewed_on="2026-11-05", review_policy_ref="SE-REVIEW-POLICY-0.1")
        ],
        "coverage_gaps": [
            rec("se:GAP-NIST-DETAILED-PRACTICES", label="Detailed practice coverage gap", target_scope_ref="se:NODE-SSDF-GROUPS", gap_kind="SOURCE_GAP", description="Detailed practices have not been modeled in this fixture.", severity="MEDIUM", coverage_status="OPEN", impact="Do not claim full secure-development coverage.", resolution_notes=["Inspect permitted source material."], evidence_refs=[])
        ],
        "relationship_assertions": []
    }

    def rel(rid, subject, predicate, obj, evidence=None, qualifiers=None):
        return rec(
            rid,
            "PENDING_REVIEW",
            subject_ref=subject,
            predicate=predicate,
            object_ref=obj,
            qualifiers=qualifiers or [],
            context="SEK-05 structural fixture context; not an accepted domain assertion.",
            rationale="Exercises the typed relationship contract and is not corpus knowledge.",
            evidence_refs=evidence or [],
        )

    graph["relationship_assertions"] = [
        rel("se:REL-SUPPORTED", "se:CLM-SWEBOK-KA", "SUPPORTED_BY", "se:EVD-SWEBOK-KA"),
        rel("se:REL-DERIVED", "se:CON-SOFTWARE-REQUIREMENTS", "DERIVED_FROM", "se:EVD-SWEBOK-KA", ["se:EVD-SEK05-TEST"]),
        rel("se:REL-CONTRADICTED", "se:CLM-SWEBOK-KA", "CONTRADICTED_BY", "se:COUNTEREXAMPLE-TEST-001", [], ["synthetic boundary fixture"]),
        rel("se:REL-QUALIFIED", "se:CLM-SWEBOK-KA", "QUALIFIED_BY", "se:EVD-SWEBOK-KA", [], ["Guide scope limitation"]),
        rel("se:REL-APPLIES", "se:CLM-SWEBOK-KA", "APPLIES_TO", "se:NODE-SWEBOK-REQ", ["se:EVD-SWEBOK-KA"], ["SWEBOK v4.0a taxonomy scope"]),
        rel("se:REL-DEPENDS", "se:PRA-RISK-BASED-SELECTION", "DEPENDS_ON", "se:CON-SOFTWARE-REQUIREMENTS", ["se:EVD-SEK05-TEST"]),
        rel("se:REL-IMPLEMENTS", "se:PRA-RISK-BASED-SELECTION", "IMPLEMENTS", "se:PRI-CONTEXTUAL-APPLICATION", ["se:EVD-SEK05-TEST"], ["operationalization fixture"]),
        rel("se:REL-ILLUSTRATED", "se:PAT-INCREMENTAL-REFINEMENT", "ILLUSTRATED_BY", "se:CASE-TEST-001"),
        rel("se:REL-VALIDATED", "se:CLM-SWEBOK-KA", "VALIDATED_BY", "se:EVD-SWEBOK-KA", [], ["evidence inspection, not acceptance"]),
        rel("se:REL-SUPERSEDES", "se:CLM-TEST-SUPERSESSION", "SUPERSEDES", "se:CLM-SWEBOK-KA", ["se:EVD-SEK05-TEST"], ["synthetic supersession fixture"]),
        rel("se:REL-RELATED", "se:CON-SOFTWARE-REQUIREMENTS", "RELATED_TO", "se:PAT-INCREMENTAL-REFINEMENT", ["se:EVD-SEK05-TEST"], ["concept-to-pattern fixture"]),
        rel("se:REL-PART-OF", "se:NODE-SWEBOK-REQ", "PART_OF", "se:NODE-SWEBOK-ENG"),
        rel("se:REL-SPECIALIZES", "se:SUB-REQUIREMENTS-ENGINEERING", "SPECIALIZES", "se:NODE-SWEBOK-REQ", ["se:EVD-SEK05-TEST"], ["subdomain-to-taxonomy mapping fixture"]),
        rel("se:REL-CROSS-CUTS", "se:NODE-SSDF-GROUPS", "CROSS_CUTS", "se:NODE-SWEBOK-REQ", ["se:EVD-SEK05-TEST"], ["cross-cutting security mapping fixture"]),
        rel("se:REL-OVERLAPS", "se:NODE-CS2023-SE", "OVERLAPS", "se:NODE-SWEBOK-ENG", ["se:EVD-SEK05-TEST"], ["partial scope overlap fixture"]),
        rel("se:REL-BOUNDED-BY", "se:DOMAIN-SOFTWARE-ENGINEERING", "BOUNDED_BY", "se:EVD-SWEBOK-KA", [], ["source-defined boundary evidence"]),
    ]

    target = next(record for record in graph["knowledge_claims"] if record["id"] == "se:CLM-SWEBOK-KA")
    graph["review_decisions"][0]["target_digest_sha256"] = canonical_record_digest(target)
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

    def add_exact_acceptance(graph, target):
        target["lifecycle_status"] = "ACCEPTED"
        if "review_status" in target:
            target["review_status"] = "ACCEPTED"
        graph["review_decisions"].append({
            "id": "se:REV-TEST-ACCEPT",
            "record_version": "0.1.0",
            "created_on": "2026-11-05",
            "lifecycle_status": "ACTIVE",
            "target_ref": target["id"],
            "target_record_version": target["record_version"],
            "target_digest_sha256": canonical_record_digest(target),
            "reviewer_role": "HUMAN_DOMAIN_REVIEWER",
            "reviewer_ref": "reviewer:test-fixture",
            "decision": "ACCEPT",
            "rationale": "Negative fixture proving evidence gate enforcement.",
            "reviewed_on": "2026-11-05",
            "review_policy_ref": "SE-REVIEW-POLICY-0.1",
        })

    baseline_errors = validate_graph(base, catalog, rules)
    cases.append(("positive-full-entity-and-predicate-coverage", not baseline_errors, baseline_errors))
    expect("duplicate-id", lambda g: g["concepts"].append(copy.deepcopy(g["concepts"][0])), "DUPLICATE_IDENTIFIER")
    expect("dangling-reference", lambda g: g["evidence_records"][0].update(source_locator_ref="se:LOC-MISSING"), "DANGLING_REFERENCE")
    expect("wrong-endpoint-class", lambda g: g["relationship_assertions"][0].update(object_ref="se:CON-SOFTWARE-REQUIREMENTS"), "WRONG_OBJECT_CLASS_FOR_PREDICATE")
    expect("unknown-predicate", lambda g: g["relationship_assertions"][0].update(predicate="UNDECLARED_EDGE"), "UNKNOWN_PREDICATE")
    expect("missing-evidence", lambda g: g["relationship_assertions"][5].update(evidence_refs=[]), "MISSING_REQUIRED_EVIDENCE")
    expect("missing-qualifier", lambda g: g["relationship_assertions"][4].update(qualifiers=[]), "MISSING_REQUIRED_QUALIFIER")
    expect("accepted-synthetic", lambda g: g["counterexamples"][0].update(lifecycle_status="ACCEPTED"), "ACCEPTED_SYNTHETIC_RECORD")
    expect("accepted-without-review", lambda g: g["concepts"][0].update(lifecycle_status="ACCEPTED"), "ACCEPTED_WITHOUT_HUMAN_REVIEW")
    expect("duplicate-alias", lambda g: (g["concepts"][0].update(aliases=["se:LEGACY-1"]), g["knowledge_claims"][0].update(aliases=["se:LEGACY-1"])), "DUPLICATE_ALIAS")
    expect("supersedes-cycle", lambda g: g["relationship_assertions"].append({
        **g["relationship_assertions"][9],
        "id": "se:REL-SUPERSEDES-CYCLE",
        "subject_ref": "se:CLM-SWEBOK-KA",
        "object_ref": "se:CLM-TEST-SUPERSESSION",
    }), "SUPERSEDES_CYCLE")

    def part_of_cycle(g):
        g["relationship_assertions"].extend([
            {**g["relationship_assertions"][11], "id": "se:REL-PART-A", "subject_ref": "se:NODE-SWEBOK-REQ", "object_ref": "se:NODE-SWEBOK-ENG"},
            {**g["relationship_assertions"][11], "id": "se:REL-PART-B", "subject_ref": "se:NODE-SWEBOK-ENG", "object_ref": "se:NODE-SWEBOK-REQ"},
        ])
    expect("part-of-cycle", part_of_cycle, "PART_OF_CYCLE")
    expect("duplicate-source-locator", lambda g: g["source_locators"].append({
        **g["source_locators"][0], "id": "se:LOC-DUP",
    }), "DUPLICATE_SOURCE_LOCATOR")
    expect("review-version-mismatch", lambda g: g["review_decisions"][0].update(target_record_version="9.9.9"), "REVIEW_VERSION_MISMATCH")
    expect("review-digest-mismatch", lambda g: g["review_decisions"][0].update(target_digest_sha256="1"*64), "REVIEW_DIGEST_MISMATCH")
    expect("overlaps-reverse-pair", lambda g: g["relationship_assertions"].append({
        **g["relationship_assertions"][14],
        "id": "se:REL-OVERLAPS-REVERSE",
        "subject_ref": "se:NODE-SWEBOK-ENG",
        "object_ref": "se:NODE-CS2023-SE",
    }), "OVERLAPS_NONCANONICAL_PAIR")

    def accepted_with_pending_evidence(g):
        add_exact_acceptance(g, g["concepts"][0])
    expect("accepted-evidence-not-reviewed", accepted_with_pending_evidence, "EVIDENCE_NOT_REVIEWED")

    def accepted_with_synthetic_evidence(g):
        g["evidence_records"][-1]["synthetic"] = True
        add_exact_acceptance(g, g["concepts"][0])
    expect("accepted-synthetic-evidence", accepted_with_synthetic_evidence, "ACCEPTED_SYNTHETIC_EVIDENCE")

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
