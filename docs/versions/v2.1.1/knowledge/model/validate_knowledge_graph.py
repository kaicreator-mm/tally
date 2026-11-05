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
from datetime import date, datetime
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


def _canonical_json_value(value):
    """Normalize YAML-native dates before canonical JSON hashing."""
    if isinstance(value, datetime):
        return value.isoformat()
    if isinstance(value, date):
        return value.isoformat()
    if isinstance(value, dict):
        return {str(key): _canonical_json_value(item) for key, item in value.items()}
    if isinstance(value, (list, tuple)):
        return [_canonical_json_value(item) for item in value]
    return value


def canonical_record_digest(record):
    """SHA-256 over canonical UTF-8 JSON for one exact record revision."""
    payload = json.dumps(
        _canonical_json_value(record),
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
    if graph.get("generated_on"):
        try:
            date.fromisoformat(str(graph["generated_on"]))
        except ValueError:
            _error(errors, "INVALID_GRAPH_GENERATED_ON", str(graph["generated_on"]))

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

    if isinstance(graph.get("id"), str) and graph["id"] in entities:
        _error(errors, "GRAPH_ID_COLLIDES_WITH_ENTITY", str(graph.get("id")))

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
        if record.get("created_on"):
            try:
                date.fromisoformat(str(record["created_on"]))
            except ValueError:
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
    if not isinstance(root_domain, str) or root_domain not in entities or classes.get(root_domain) != "Domain":
        _error(errors, "INVALID_GRAPH_DOMAIN", str(root_domain))

    # A locator is unique within its source; evidence has exactly one source locator.
    locator_keys = {}
    for class_name, record_id, record in records:
        if class_name == "SourceLocator":
            source_ref = record.get("source_ref")
            locator_value = record.get("locator")
            if not isinstance(source_ref, str) or not isinstance(locator_value, str):
                _error(errors, "INVALID_SOURCE_LOCATOR_IDENTITY", record_id)
                continue
            key = (source_ref, locator_value)
            if key in locator_keys:
                _error(errors, "DUPLICATE_SOURCE_LOCATOR", f"{record_id} duplicates {locator_keys[key]}")
            locator_keys[key] = record_id
        elif class_name == "Evidence":
            locator_id = record.get("source_locator_ref")
            if not isinstance(locator_id, str):
                _error(errors, "BROKEN_EVIDENCE_SOURCE_CHAIN", record_id)
                continue
            locator = entities.get(locator_id, {})
            source_id = locator.get("source_ref")
            if not isinstance(source_id, str) or classes.get(source_id) != "Source":
                _error(errors, "BROKEN_EVIDENCE_SOURCE_CHAIN", record_id)

    predicate_rules = {
        item["predicate"]: item
        for item in relationship_catalog.get("predicates", [])
    }

    def evidence_is_usable(evidence_id, owner_id):
        if not isinstance(evidence_id, str):
            _error(errors, "INVALID_EVIDENCE_REFERENCE", f"{owner_id}->{evidence_id!r}")
            return False
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
        if evidence.get("lifecycle_status") != "ACCEPTED":
            _error(errors, "EVIDENCE_NOT_ACCEPTED", f"{owner_id}->{evidence_id}")
            return False
        locator_id = evidence.get("source_locator_ref")
        if not isinstance(locator_id, str):
            _error(errors, "BROKEN_EVIDENCE_SOURCE_CHAIN", f"{owner_id}->{evidence_id}")
            return False
        locator = entities.get(locator_id, {})
        source_id = locator.get("source_ref")
        if not isinstance(source_id, str) or classes.get(locator_id) != "SourceLocator" or classes.get(source_id) != "Source":
            _error(errors, "BROKEN_EVIDENCE_SOURCE_CHAIN", f"{owner_id}->{evidence_id}")
            return False
        if locator.get("verification_state") != "LOCATOR_VERIFIED":
            _error(errors, "SOURCE_LOCATOR_NOT_VERIFIED", f"{owner_id}->{evidence_id}")
            return False
        binding = (evidence_id, evidence.get("record_version"), canonical_record_digest(evidence))
        if binding not in accepted_review_bindings:
            _error(errors, "EVIDENCE_WITHOUT_HUMAN_REVIEW", f"{owner_id}->{evidence_id}")
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

    # Bind review outcomes to the exact target revision and content digest.
    accepted_review_bindings = set()
    review_outcomes = defaultdict(set)
    for class_name, record_id, record in records:
        if class_name != "ReviewDecision":
            continue
        reviewer_ref = record.get("reviewer_ref")
        review_policy_ref = record.get("review_policy_ref")
        review_rationale = record.get("rationale")
        if reviewer_ref is None or not str(reviewer_ref).strip():
            _error(errors, "MISSING_REVIEWER_IDENTITY", record_id)
        if review_policy_ref is None or not str(review_policy_ref).strip():
            _error(errors, "MISSING_REVIEW_POLICY", record_id)
        if review_rationale is None or not str(review_rationale).strip():
            _error(errors, "MISSING_REVIEW_RATIONALE", record_id)
        target_id = record.get("target_ref")
        target = entities.get(target_id) if isinstance(target_id, str) else None
        if not target:
            continue
        target_version = record.get("target_record_version")
        if not VERSION_PATTERN.fullmatch(str(target_version or "")):
            _error(errors, "INVALID_REVIEW_TARGET_VERSION", record_id)
        digest = record.get("target_digest_sha256", "")
        digest_valid = bool(SHA256_PATTERN.fullmatch(str(digest)))
        if not digest_valid:
            _error(errors, "INVALID_REVIEW_DIGEST", record_id)

        # Historical review events remain auditable but cannot authorize or
        # conflict with the current record. Only ACTIVE decisions bind the
        # current target revision and participate in effective outcomes.
        if record.get("lifecycle_status") != "ACTIVE":
            continue

        version_matches = target_version == target.get("record_version")
        if not version_matches:
            _error(errors, "REVIEW_VERSION_MISMATCH", record_id)
        digest_matches = digest_valid and digest == canonical_record_digest(target)
        if digest_valid and not digest_matches:
            _error(errors, "REVIEW_DIGEST_MISMATCH", record_id)
        exact_target = version_matches and digest_matches
        decision = record.get("decision")
        if exact_target and isinstance(decision, str):
            review_outcomes[(target_id, target.get("record_version"), digest)].add(decision)
        if decision == "ACCEPT":
            if target.get("lifecycle_status") != "ACCEPTED":
                _error(errors, "REVIEW_ACCEPT_TARGET_NOT_ACCEPTED", record_id)
            if classes.get(target_id) == "KnowledgeClaim" and target.get("review_status") != "ACCEPTED":
                _error(errors, "CLAIM_REVIEW_STATUS_MISMATCH", target_id)
            if classes.get(target_id) == "Evidence" and target.get("extraction_status") != "REVIEWED":
                _error(errors, "ACCEPTED_EVIDENCE_NOT_REVIEWED", target_id)
            if exact_target and target.get("lifecycle_status") == "ACCEPTED":
                accepted_review_bindings.add((target_id, target.get("record_version"), digest))
        elif isinstance(decision, str) and decision in {"REJECT", "REQUEST_CHANGES"} and target.get("lifecycle_status") == "ACCEPTED":
            _error(errors, "REVIEW_NEGATIVE_DECISION_TARGET_ACCEPTED", record_id)

    for binding, outcomes in review_outcomes.items():
        if "ACCEPT" in outcomes and outcomes.intersection({"REJECT", "REQUEST_CHANGES"}):
            _error(errors, "CONFLICTING_REVIEW_DECISIONS", f"{binding[0]} has conflicting outcomes {sorted(outcomes)}")

    # An accepted Evidence record must have its own exact human acceptance.
    for class_name, record_id, record in records:
        if class_name != "Evidence" or record.get("lifecycle_status") != "ACCEPTED":
            continue
        if record.get("extraction_status") != "REVIEWED":
            _error(errors, "ACCEPTED_EVIDENCE_NOT_REVIEWED", record_id)
        binding = (record_id, record.get("record_version"), canonical_record_digest(record))
        if binding not in accepted_review_bindings:
            _error(errors, "EVIDENCE_WITHOUT_HUMAN_REVIEW", record_id)

    # Accepted records must have reviewed, non-synthetic evidence and an exact human decision.
    accepted_ids = set()
    for class_name, record_id, record in records:
        if class_name not in KNOWLEDGE_CLASSES or record.get("lifecycle_status") != "ACCEPTED":
            continue
        accepted_ids.add(record_id)
        if class_name != "RelationshipAssertion":
            if class_name == "CoverageGap":
                if record.get("coverage_status") != "ACCEPTED_LIMITATION" or not _ids(record.get("resolution_notes")):
                    _error(errors, "ACCEPTED_GAP_WITHOUT_DISPOSITION", record_id)
                if _ids(record.get("evidence_refs")):
                    usable_evidence_refs(record_id, record.get("evidence_refs"))
            elif class_name in {"Pattern", "AntiPattern"}:
                evidence_ids = _ids(record.get("evidence_refs"))
                case_ids = _ids(record.get("case_refs"))
                if not evidence_ids and not case_ids:
                    _error(errors, "ACCEPTED_WITHOUT_EVIDENCE_OR_CASE", record_id)
                if evidence_ids:
                    usable_evidence_refs(record_id, evidence_ids)
                for case_id in case_ids:
                    if not isinstance(case_id, str):
                        _error(errors, "INVALID_CASE_REFERENCE", f"{record_id}->{case_id!r}")
                        continue
                    case = entities.get(case_id, {})
                    if case.get("synthetic") is True:
                        _error(errors, "ACCEPTED_SYNTHETIC_CASE", f"{record_id}->{case_id}")
                    elif case.get("lifecycle_status") != "ACCEPTED":
                        _error(errors, "ACCEPTED_PATTERN_REFERENCES_UNACCEPTED_CASE", f"{record_id}->{case_id}")
                    else:
                        case_binding = (case_id, case.get("record_version"), canonical_record_digest(case))
                        if case_binding not in accepted_review_bindings:
                            _error(errors, "ACCEPTED_PATTERN_CASE_WITHOUT_HUMAN_REVIEW", f"{record_id}->{case_id}")
                        usable_evidence_refs(case_id, case.get("evidence_refs"))
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
        if not isinstance(predicate, str):
            _error(errors, "UNKNOWN_PREDICATE", f"{record_id}: {predicate!r}")
            continue
        rule = predicate_rules.get(predicate)
        if not rule:
            _error(errors, "UNKNOWN_PREDICATE", f"{record_id}: {predicate!r}")
            continue
        subject_id = record.get("subject_ref")
        object_id = record.get("object_ref")
        if not isinstance(subject_id, str) or not isinstance(object_id, str):
            _error(errors, "INVALID_RELATIONSHIP_ENDPOINT", f"{record_id}: subject={subject_id!r}, object={object_id!r}")
            continue
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
        context = record.get("context")
        rationale = record.get("rationale")
        if not isinstance(context, str) or not context.strip() or not isinstance(rationale, str) or not rationale.strip():
            _error(errors, "MISSING_RELATION_CONTEXT", record_id)
        if subject_id == object_id:
            _error(errors, "SELF_REFERENTIAL_RELATIONSHIP", record_id)
        if predicate == "OVERLAPS" and subject_id >= object_id:
            _error(errors, "OVERLAPS_NONCANONICAL_PAIR", record_id)
        qualifiers = _ids(record.get("qualifiers"))
        if any(not isinstance(value, str) or not value.strip() for value in qualifiers):
            _error(errors, "INVALID_RELATIONSHIP_QUALIFIER", record_id)
            qualifier_signature = ()
        else:
            qualifier_signature = tuple(sorted(qualifiers))
        signature = (
            subject_id,
            predicate,
            object_id,
            qualifier_signature,
            context if isinstance(context, str) else "",
        )
        if signature in edge_signatures:
            _error(errors, "DUPLICATE_RELATIONSHIP_SIGNATURE", record_id)
        edge_signatures.add(signature)
        terminal_edge_states = {"REJECTED", "WITHDRAWN", "RETRACTED", "SUPERSEDED", "DEPRECATED"}
        if predicate in {"PART_OF", "SUPERSEDES"} and record.get("lifecycle_status") not in terminal_edge_states:
            graph_edges[predicate].append((subject_id, object_id))
        if rule.get("same_class_required") and subject_class != object_class:
            _error(errors, "SUPERSEDES_CLASS_MISMATCH", record_id)

        if record.get("lifecycle_status") == "ACCEPTED":
            evidence_ids = _ids(record.get("evidence_refs"))
            object_record = entities.get(object_id, {})
            if object_class in {"Case", "Counterexample"} and object_record.get("synthetic") is True:
                _error(errors, "ACCEPTED_SYNTHETIC_CASE", f"{record_id}->{object_id}")
            if not evidence_ids and rule.get("object_is_evidence") and object_class == "Evidence":
                evidence_ids = [object_id]
            elif not evidence_ids and rule.get("object_carries_provenance"):
                if object_class == "Evidence":
                    evidence_ids = [object_id]
                elif object_class == "ReviewDecision":
                    review_target_id = object_record.get("target_ref")
                    if not isinstance(review_target_id, str):
                        _error(errors, "VALIDATION_REVIEW_DOES_NOT_BIND_SUBJECT", f"{record_id}->{object_id}")
                    else:
                        review_binding = (review_target_id, object_record.get("target_record_version"), object_record.get("target_digest_sha256"))
                        if object_record.get("decision") != "ACCEPT" or review_target_id != subject_id or review_binding not in accepted_review_bindings:
                            _error(errors, "VALIDATION_REVIEW_DOES_NOT_BIND_SUBJECT", f"{record_id}->{object_id}")
                        else:
                            evidence_ids = _ids(entities.get(review_target_id, {}).get("evidence_refs"))
                else:
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


def accepted_relationship_with_exact_review_graph(base, add_exact_acceptance):
    graph = copy.deepcopy(base)
    evidence = graph["evidence_records"][0]
    evidence["extraction_status"] = "REVIEWED"
    add_exact_acceptance(graph, evidence)
    add_exact_acceptance(graph, graph["relationship_assertions"][0])
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

    def add_exact_acceptance(graph, target, review_id=None):
        target["lifecycle_status"] = "ACCEPTED"
        if "review_status" in target:
            target["review_status"] = "ACCEPTED"
        graph["review_decisions"].append({
            "id": review_id or f"se:REV-TEST-ACCEPT-{target['id'].split(':', 1)[1]}",
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
        g["concepts"][0]["evidence_refs"] = ["se:EVD-SEK05-TEST"]
        add_exact_acceptance(g, g["concepts"][0])
    expect("accepted-synthetic-evidence", accepted_with_synthetic_evidence, "ACCEPTED_SYNTHETIC_EVIDENCE")

    def accept_fixture_evidence(g):
        evidence = g["evidence_records"][0]
        evidence["extraction_status"] = "REVIEWED"
        add_exact_acceptance(g, evidence)

    def accepted_relationship_without_review(g):
        accept_fixture_evidence(g)
        g["relationship_assertions"][0]["lifecycle_status"] = "ACCEPTED"
    expect("accepted-relationship-without-review", accepted_relationship_without_review, "ACCEPTED_WITHOUT_HUMAN_REVIEW")

    def accepted_relationship_with_exact_review(g):
        accept_fixture_evidence(g)
        add_exact_acceptance(g, g["relationship_assertions"][0])
    valid_relation_errors = validate_graph(
        accepted_relationship_with_exact_review_graph(base, add_exact_acceptance),
        catalog,
        rules,
    )
    cases.append(("accepted-relationship-with-exact-review-and-evidence", not valid_relation_errors, valid_relation_errors))

    def duplicate_relationship_signature(g):
        duplicate = copy.deepcopy(g["relationship_assertions"][4])
        duplicate["id"] = "se:REL-DUPLICATE-SIGNATURE"
        g["relationship_assertions"].append(duplicate)
    expect("duplicate-relationship-signature", duplicate_relationship_signature, "DUPLICATE_RELATIONSHIP_SIGNATURE")

    expect("graph-id-collides-with-entity", lambda g: g.update(id=g["domains"][0]["id"]), "GRAPH_ID_COLLIDES_WITH_ENTITY")
    expect("invalid-root-domain-type", lambda g: g.update(domain_ref=["se:DOMAIN-SOFTWARE-ENGINEERING"]), "INVALID_GRAPH_DOMAIN")
    expect("invalid-relationship-endpoint-type", lambda g: g["relationship_assertions"][0].update(subject_ref=["se:CLM-SWEBOK-KA"]), "INVALID_RELATIONSHIP_ENDPOINT")
    def invalid_evidence_reference_type(g):
        target = g["concepts"][0]
        target["evidence_refs"] = [{"not": "an-id"}]
        add_exact_acceptance(g, target)
    expect("invalid-evidence-reference-type", invalid_evidence_reference_type, "INVALID_EVIDENCE_REFERENCE")

    def accepted_evidence_without_own_review(g):
        evidence = g["evidence_records"][0]
        evidence["extraction_status"] = "REVIEWED"
        evidence["lifecycle_status"] = "ACCEPTED"
        add_exact_acceptance(g, g["concepts"][0])
    expect("accepted-evidence-without-own-review", accepted_evidence_without_own_review, "EVIDENCE_WITHOUT_HUMAN_REVIEW")

    def conflicting_review_outcomes(g):
        target = g["concepts"][0]
        target["lifecycle_status"] = "ACCEPTED"
        add_exact_acceptance(g, target)
        g["review_decisions"].append({
            "id": "se:REV-TEST-REJECT-CONFLICT",
            "record_version": "0.1.0",
            "created_on": "2026-11-05",
            "lifecycle_status": "ACTIVE",
            "target_ref": target["id"],
            "target_record_version": target["record_version"],
            "target_digest_sha256": canonical_record_digest(target),
            "reviewer_role": "HUMAN_DOMAIN_REVIEWER",
            "reviewer_ref": "reviewer:test-fixture-2",
            "decision": "REJECT",
            "rationale": "Conflicting exact-revision decision fixture.",
            "reviewed_on": "2026-11-05",
            "review_policy_ref": "SE-REVIEW-POLICY-0.1",
        })
    expect("conflicting-review-outcomes", conflicting_review_outcomes, "CONFLICTING_REVIEW_DECISIONS")

    def accepted_pattern_with_synthetic_case(g):
        pattern = g["patterns"][0]
        pattern["lifecycle_status"] = "ACCEPTED"
        add_exact_acceptance(g, pattern)
    expect("accepted-pattern-synthetic-case", accepted_pattern_with_synthetic_case, "ACCEPTED_SYNTHETIC_CASE")

    def rejected_part_of_cycle(g):
        first = {**g["relationship_assertions"][11], "id": "se:REL-REJECTED-PART-A", "subject_ref": "se:NODE-SWEBOK-REQ", "object_ref": "se:NODE-SWEBOK-ENG", "lifecycle_status": "REJECTED", "context": "rejected cycle fixture A"}
        second = {**g["relationship_assertions"][11], "id": "se:REL-REJECTED-PART-B", "subject_ref": "se:NODE-SWEBOK-ENG", "object_ref": "se:NODE-SWEBOK-REQ", "lifecycle_status": "REJECTED", "context": "rejected cycle fixture B"}
        g["relationship_assertions"].extend([first, second])
    rejected_cycle_graph = copy.deepcopy(base)
    rejected_part_of_cycle(rejected_cycle_graph)
    rejected_cycle_errors = validate_graph(rejected_cycle_graph, catalog, rules)
    cases.append(("rejected-edges-do-not-form-active-cycle", not any(error.startswith("PART_OF_CYCLE:") for error in rejected_cycle_errors), rejected_cycle_errors))

    try:
        canonical_record_digest({"created_on": date(2026, 11, 5)})
        date_digest_ok = True
    except (TypeError, ValueError):
        date_digest_ok = False
    cases.append(("canonical-digest-normalizes-yaml-date", date_digest_ok, []))

    def missing_reviewer_identity(g):
        target = g["concepts"][0]
        add_exact_acceptance(g, target)
        g["review_decisions"][-1]["reviewer_ref"] = " "
    expect("missing-reviewer-identity", missing_reviewer_identity, "MISSING_REVIEWER_IDENTITY")

    def inactive_accept_review_does_not_bind(g):
        accept_fixture_evidence(g)
        target = g["concepts"][0]
        add_exact_acceptance(g, target)
        g["review_decisions"][-1]["lifecycle_status"] = "RETRACTED"
    expect("inactive-accept-review-does-not-bind", inactive_accept_review_does_not_bind, "ACCEPTED_WITHOUT_HUMAN_REVIEW")

    def inactive_negative_review_is_historical(g):
        accept_fixture_evidence(g)
        target = g["concepts"][0]
        add_exact_acceptance(g, target)
        g["review_decisions"].append({
            "id": "se:REV-TEST-INACTIVE-REJECT",
            "record_version": "0.1.0",
            "created_on": "2026-11-05",
            "lifecycle_status": "RETRACTED",
            "target_ref": target["id"],
            "target_record_version": target["record_version"],
            "target_digest_sha256": canonical_record_digest(target),
            "reviewer_role": "HUMAN_DOMAIN_REVIEWER",
            "reviewer_ref": "reviewer:test-fixture-inactive",
            "decision": "REJECT",
            "rationale": "Historical negative decision retained for audit.",
            "reviewed_on": "2026-11-05",
            "review_policy_ref": "SE-REVIEW-POLICY-0.1",
        })
    inactive_negative_graph = copy.deepcopy(base)
    inactive_negative_review_is_historical(inactive_negative_graph)
    inactive_negative_errors = validate_graph(inactive_negative_graph, catalog, rules)
    cases.append(("inactive-negative-review-does-not-conflict", not inactive_negative_errors, inactive_negative_errors))

    def inactive_stale_review_history(g):
        accept_fixture_evidence(g)
        target = g["concepts"][0]
        add_exact_acceptance(g, target)
        g["review_decisions"].append({
            "id": "se:REV-TEST-SUPERSEDED-HISTORY",
            "record_version": "0.1.0",
            "created_on": "2026-11-05",
            "lifecycle_status": "SUPERSEDED",
            "target_ref": target["id"],
            "target_record_version": "0.0.9",
            "target_digest_sha256": "a" * 64,
            "reviewer_role": "HUMAN_DOMAIN_REVIEWER",
            "reviewer_ref": "reviewer:test-fixture-history",
            "decision": "ACCEPT",
            "rationale": "Historical review of an older record revision.",
            "reviewed_on": "2026-11-05",
            "review_policy_ref": "SE-REVIEW-POLICY-0.1",
        })
    inactive_history_graph = copy.deepcopy(base)
    inactive_stale_review_history(inactive_history_graph)
    inactive_history_errors = validate_graph(inactive_history_graph, catalog, rules)
    cases.append(("inactive-stale-review-remains-historical", not inactive_history_errors, inactive_history_errors))

    def validated_by_review_decision_graph(g):
        evidence = g["evidence_records"][0]
        evidence["extraction_status"] = "REVIEWED"
        add_exact_acceptance(g, evidence)
        target = g["knowledge_claims"][0]
        add_exact_acceptance(g, target)
        g["review_decisions"][0]["target_digest_sha256"] = canonical_record_digest(target)
        review = g["review_decisions"][-1]
        edge = {
            "id": "se:REL-VALIDATED-BY-REVIEW",
            "record_version": "0.1.0",
            "created_on": "2026-11-05",
            "lifecycle_status": "ACCEPTED",
            "subject_ref": target["id"],
            "predicate": "VALIDATED_BY",
            "object_ref": review["id"],
            "qualifiers": ["exact human review fixture"],
            "context": "SEK-05 test-only validation relationship.",
            "rationale": "Exercises ReviewDecision object provenance.",
            "evidence_refs": [],
        }
        g["relationship_assertions"].append(edge)
        add_exact_acceptance(g, edge)
        return g
    review_object_graph = validated_by_review_decision_graph(copy.deepcopy(base))
    review_object_errors = validate_graph(review_object_graph, catalog, rules)
    cases.append(("accepted-validated-by-review-object-provenance", not review_object_errors, review_object_errors))

    def expect_format_failure(name, mutate, expected_code):
        format_files = _serialize_format_b(base)
        mutate(format_files)
        try:
            _parse_format_b(format_files)
            format_errors = []
        except ValidationError as exc:
            format_errors = [str(exc)]
        passed = any(error.startswith(expected_code + ":") for error in format_errors)
        cases.append((name, passed, format_errors))

    concept_slug = base["concepts"][0]["id"].split(":", 1)[1].lower()
    concept_explanation = f"explanations/{concept_slug}.md"
    expect_format_failure(
        "format-b-missing-explanation",
        lambda files: files.pop(concept_explanation, None),
        "FORMAT_B_MISSING_EXPLANATION",
    )
    expect_format_failure(
        "format-b-explanation-id-mismatch",
        lambda files: files.__setitem__(concept_explanation, "# se:WRONG-RECORD-ID\\n\\nWrong link.\\n"),
        "FORMAT_B_EXPLANATION_ID_MISMATCH",
    )
    expect_format_failure(
        "format-b-orphan-explanation",
        lambda files: files.__setitem__("explanations/orphan.md", "# se:ORPHAN\\n"),
        "FORMAT_B_ORPHAN_EXPLANATION",
    )

    passed = sum(1 for _, ok, _ in cases if ok)
    for name, ok, errors in cases:
        print(f"{'PASS' if ok else 'FAIL'} {name}")
        if not ok:
            for error in errors:
                print(f"  {error}")
    print(f"SELF_TESTS={passed}/{len(cases)}")
    return 0 if passed == len(cases) else 1




def _yaml_dump(value):
    return yaml.safe_dump(
        value,
        allow_unicode=True,
        sort_keys=False,
        default_flow_style=False,
    )


def _serialize_format_a(graph):
    """One Markdown file per entity with YAML front matter and inline narrative."""
    files = {}
    root = {
        key: graph[key]
        for key in ("id", "graph_version", "generated_on", "domain_ref")
    }
    root["collection_order"] = list(COLLECTIONS)
    root_text = _yaml_dump({"graph_root": root})
    files["graph-root.md"] = (
        f"---\n{root_text}---\n\n"
        "# Software Engineering knowledge graph\n\n"
        "Graph-level narrative is stored beside its YAML front matter.\n"
    )
    for collection in COLLECTIONS:
        for index, record in enumerate(graph.get(collection, [])):
            slug = record["id"].split(":", 1)[1].lower()
            record_text = _yaml_dump({"record": record})
            files[f"records/{collection}/{index:04d}-{slug}.md"] = (
                f"---\n{record_text}---\n\n"
                f"# {record['id']}\n\n"
                "Entity explanation is stored in the same Markdown file as its front matter.\n"
            )
    return files


def _parse_frontmatter(document, key):
    parts = document.split("---", 2)
    if len(parts) != 3 or parts[0].strip():
        raise ValidationError("Markdown front matter delimiters are invalid")
    payload = yaml.safe_load(parts[1])
    if not isinstance(payload, dict) or not isinstance(payload.get(key), dict):
        raise ValidationError(f"Markdown front matter does not contain {key} mapping")
    return payload[key]


def _parse_format_a(files):
    root = _parse_frontmatter(files["graph-root.md"], "graph_root")
    graph = {
        key: root[key]
        for key in ("id", "graph_version", "generated_on", "domain_ref")
    }
    for collection in root["collection_order"]:
        prefix = f"records/{collection}/"
        record_paths = sorted(path for path in files if path.startswith(prefix) and path.endswith(".md"))
        graph[collection] = [
            _parse_frontmatter(files[path], "record")
            for path in record_paths
        ]
    return graph


def _serialize_format_b(graph):
    """One YAML data file per entity plus a separate linked Markdown explanation."""
    files = {}
    root = {
        key: graph[key]
        for key in ("id", "graph_version", "generated_on", "domain_ref")
    }
    root["collection_order"] = list(COLLECTIONS)
    files["graph-root.yaml"] = _yaml_dump(root)
    for collection in COLLECTIONS:
        for index, record in enumerate(graph.get(collection, [])):
            slug = record["id"].split(":", 1)[1].lower()
            files[f"records/{collection}/{index:04d}-{slug}.yaml"] = _yaml_dump(record)
            files[f"explanations/{slug}.md"] = (
                f"# {record['id']}\n\n"
                "Entity explanation is separate from the authoritative structured YAML record.\n"
            )
    return files


def _parse_format_b(files):
    if "graph-root.yaml" not in files:
        raise ValidationError("FORMAT_B_MISSING_GRAPH_ROOT: graph-root.yaml is absent")
    root = yaml.safe_load(files["graph-root.yaml"])
    if not isinstance(root, dict):
        raise ValidationError("FORMAT_B_INVALID_GRAPH_ROOT: graph root must be a mapping")
    collection_order = root.get("collection_order")
    if (
        not isinstance(collection_order, list)
        or len(collection_order) != len(COLLECTIONS)
        or set(collection_order) != set(COLLECTIONS)
    ):
        raise ValidationError("FORMAT_B_INVALID_COLLECTION_ORDER: every entity collection must appear exactly once")
    graph = {
        key: root[key]
        for key in ("id", "graph_version", "generated_on", "domain_ref")
    }
    expected_explanations = set()
    consumed_record_paths = set()
    for collection in collection_order:
        prefix = f"records/{collection}/"
        record_paths = sorted(path for path in files if path.startswith(prefix) and path.endswith(".yaml"))
        graph[collection] = []
        for path in record_paths:
            record = yaml.safe_load(files[path])
            if not isinstance(record, dict) or not isinstance(record.get("id"), str):
                raise ValidationError(f"FORMAT_B_INVALID_RECORD: {path} must contain a record with a canonical id")
            slug = record["id"].split(":", 1)[-1].lower()
            explanation_path = f"explanations/{slug}.md"
            if explanation_path in expected_explanations:
                raise ValidationError(f"FORMAT_B_DUPLICATE_EXPLANATION: {explanation_path} maps to multiple records")
            expected_explanations.add(explanation_path)
            explanation = files.get(explanation_path)
            if explanation is None:
                raise ValidationError(f"FORMAT_B_MISSING_EXPLANATION: {record['id']} requires {explanation_path}")
            lines = explanation.splitlines()
            if not lines or lines[0].strip() != f"# {record['id']}":
                raise ValidationError(f"FORMAT_B_EXPLANATION_ID_MISMATCH: {explanation_path} must identify {record['id']}")
            graph[collection].append(record)
            consumed_record_paths.add(path)

    actual_record_paths = {
        path for path in files
        if path.startswith("records/") and path.endswith(".yaml")
    }
    orphan_records = actual_record_paths - consumed_record_paths
    if orphan_records:
        raise ValidationError(f"FORMAT_B_ORPHAN_RECORD: {sorted(orphan_records)}")

    actual_explanations = {
        path for path in files
        if path.startswith("explanations/") and path.endswith(".md")
    }
    orphan_explanations = actual_explanations - expected_explanations
    if orphan_explanations:
        raise ValidationError(f"FORMAT_B_ORPHAN_EXPLANATION: {sorted(orphan_explanations)}")
    return graph


def _serialize_format_c(graph):
    """One standalone YAML graph document."""
    return _yaml_dump(graph)


def _parse_format_c(document):
    graph = yaml.safe_load(document)
    if not isinstance(graph, dict):
        raise ValidationError("Format C root must be a graph mapping")
    return graph


def _changed_metrics(before, after):
    import difflib

    changed_paths = sorted(
        path for path in set(before) | set(after)
        if before.get(path) != after.get(path)
    )
    changed_lines = 0
    for path in changed_paths:
        old_lines = before.get(path, "").splitlines()
        new_lines = after.get(path, "").splitlines()
        diff = difflib.unified_diff(old_lines, new_lines, lineterm="")
        changed_lines += sum(
            1 for line in diff
            if (line.startswith("+") or line.startswith("-"))
            and not line.startswith("+++") and not line.startswith("---")
        )
    return changed_paths, changed_lines


def _format_snapshots(graph):
    return {
        "A": _serialize_format_a(graph),
        "B": _serialize_format_b(graph),
        "C": {"knowledge.yaml": _serialize_format_c(graph)},
    }


def run_format_comparison():
    graph = positive_fixture()
    original = canonical_record_digest(graph)
    snapshots = _format_snapshots(graph)
    roundtrips = {
        "A": _parse_format_a(snapshots["A"]),
        "B": _parse_format_b(snapshots["B"]),
        "C": _parse_format_c(snapshots["C"]["knowledge.yaml"]),
    }
    roundtrip_digests = {
        name: canonical_record_digest(value)
        for name, value in roundtrips.items()
    }
    equivalent = all(value == original for value in roundtrip_digests.values())
    deterministic = all(
        _format_snapshots(graph)[name] == snapshots[name]
        for name in ("A", "B", "C")
    )

    one_record = copy.deepcopy(graph)
    one_record["concepts"][0]["definition"] += " Revised once."
    one_record_snapshots = _format_snapshots(one_record)
    one_record_metrics = {
        name: _changed_metrics(snapshots[name], one_record_snapshots[name])
        for name in ("A", "B", "C")
    }

    two_records = copy.deepcopy(graph)
    two_records["concepts"][0]["definition"] += " Revised once."
    two_records["knowledge_claims"][0]["statement"] += " Revised once."
    two_record_snapshots = _format_snapshots(two_records)
    two_record_metrics = {
        name: _changed_metrics(snapshots[name], two_record_snapshots[name])
        for name in ("A", "B", "C")
    }

    writer_a = copy.deepcopy(graph)
    writer_b = copy.deepcopy(graph)
    writer_a["concepts"][0]["definition"] += " Writer A."
    writer_b["knowledge_claims"][0]["statement"] += " Writer B."
    writer_a_snapshots = _format_snapshots(writer_a)
    writer_b_snapshots = _format_snapshots(writer_b)
    conflict_paths = {
        name: sorted(
            set(_changed_metrics(snapshots[name], writer_a_snapshots[name])[0])
            & set(_changed_metrics(snapshots[name], writer_b_snapshots[name])[0])
        )
        for name in ("A", "B", "C")
    }

    # Simulate concurrent structured-data and narrative edits to the same entity.
    data_writer = copy.deepcopy(graph)
    data_writer["concepts"][0]["definition"] += " Structured edit."
    data_snapshots = _format_snapshots(data_writer)
    narrative_writer = {
        name: dict(files)
        for name, files in snapshots.items()
    }
    concept_slug = graph["concepts"][0]["id"].split(":", 1)[1].lower()
    for path in narrative_writer["A"]:
        if path.endswith(f"-{concept_slug}.md"):
            narrative_writer["A"][path] += "\nConcurrent narrative edit.\n"
            break
    narrative_writer["B"][f"explanations/{concept_slug}.md"] += "\nConcurrent narrative edit.\n"
    same_entity_conflicts = {
        "A": sorted(
            set(_changed_metrics(snapshots["A"], data_snapshots["A"])[0])
            & set(_changed_metrics(snapshots["A"], narrative_writer["A"])[0])
        ),
        "B": sorted(
            set(_changed_metrics(snapshots["B"], data_snapshots["B"])[0])
            & set(_changed_metrics(snapshots["B"], narrative_writer["B"])[0])
        ),
        "C": None,
    }

    print("FORMAT_B_EXPLANATION_LINKS=PASS")
    print(f"FORMAT_EQUIVALENCE={'PASS' if equivalent else 'FAIL'}")
    print(f"CANONICAL_GRAPH_SHA256={original}")
    for name in ("A", "B", "C"):
        print(f"FORMAT_{name}_ROUNDTRIP_SHA256={roundtrip_digests[name]}")
        print(f"FORMAT_{name}_FILE_COUNT={len(snapshots[name])}")
        paths, lines = one_record_metrics[name]
        print(f"FORMAT_{name}_ONE_RECORD_CHANGED_FILES={len(paths)}")
        print(f"FORMAT_{name}_ONE_RECORD_CHANGED_LINES={lines}")
        paths, lines = two_record_metrics[name]
        print(f"FORMAT_{name}_TWO_RECORD_CHANGED_FILES={len(paths)}")
        print(f"FORMAT_{name}_TWO_RECORD_CHANGED_LINES={lines}")
        print(f"FORMAT_{name}_TWO_WRITER_CONFLICT_FILES={len(conflict_paths[name])}")
        if same_entity_conflicts[name] is None:
            print(f"FORMAT_{name}_SAME_ENTITY_DATA_NARRATIVE_CONFLICT=NOT_APPLICABLE_NO_SEPARATE_NARRATIVE_CHANNEL")
        else:
            print(f"FORMAT_{name}_SAME_ENTITY_DATA_NARRATIVE_CONFLICT_FILES={len(same_entity_conflicts[name])}")
    print(f"DETERMINISTIC_REBUILD={'PASS' if deterministic else 'FAIL'}")
    print("HUMAN_REVIEW_BURDEN=NOT_AUTOMATED_REQUIRES_REVIEWER_ASSESSMENT")
    return 0 if equivalent and deterministic else 1


def main():
    parser = argparse.ArgumentParser(description=__doc__)
    parser.add_argument("--catalog", type=Path, default=DEFAULT_CATALOG)
    parser.add_argument("--rules", type=Path, default=DEFAULT_RULES)
    parser.add_argument("--input", type=Path)
    parser.add_argument("--self-test", action="store_true")
    parser.add_argument("--emit-fixture", type=Path)
    parser.add_argument("--format-compare", action="store_true")
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
        if args.format_compare:
            return run_format_comparison()
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
