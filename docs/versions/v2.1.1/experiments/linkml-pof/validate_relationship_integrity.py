#!/usr/bin/env python3
"""Fail-closed integrity checker for the experimental LinkML proof-of-fit.

LinkML validates record shape. This companion validates resource identity,
cross-record closure, predicate-specific endpoints and generated RDF node kinds.
It is experiment tooling, not production runtime code.
"""
from __future__ import annotations

import argparse
import copy
import sys
from pathlib import Path

import yaml
from rdflib import Graph, Literal, Namespace, URIRef
from rdflib.namespace import RDF

SE_BASE = "https://github.com/kaicreator-mm/tally#"
SE = Namespace(SE_BASE)
PROV = Namespace("http://www.w3.org/ns/prov#")

SCHEMA_ID = "https://github.com/kaicreator-mm/tally"
EXPECTED_RULES = {
    "SUPPORTED_BY": ("KnowledgeClaim", "Evidence"),
    "INCLUDES": ("KnowledgeClaim", "DomainNode"),
}
REQUIRED_FAIL_CLOSED = frozenset({
    "unknown_predicate", "missing_subject", "missing_object",
    "dangling_subject_reference", "dangling_object_reference",
    "wrong_subject_class", "wrong_object_class_for_predicate",
    "duplicate_identifier",
})
EXPECTED_RDF_EXPECTATIONS = {
    "object_predicate": "rdf:object",
    "object_node_kind": "URIRef",
    "reject_object_literal": True,
    "identifier_prefix": "se",
    "expanded_identifier_base": SE_BASE,
}
# LinkML may also emit an explicit abstract superclass type for these classes.
CLASS_ANCESTORS = {"Evidence": {"GraphNode"}, "DomainNode": {"GraphNode"}}


class IntegrityError(Exception):
    pass


class UniqueLoader(yaml.SafeLoader):
    pass


def construct_mapping(loader, node, deep=False):
    result = {}
    for key_node, value_node in node.value:
        key = loader.construct_object(key_node, deep=deep)
        if key in result:
            mark = key_node.start_mark
            raise IntegrityError(
                f"duplicate YAML key {key!r} at {mark.line + 1}:{mark.column + 1}"
            )
        result[key] = loader.construct_object(value_node, deep=deep)
    return result


UniqueLoader.add_constructor(
    yaml.resolver.BaseResolver.DEFAULT_MAPPING_TAG, construct_mapping
)


def parse_yaml(text, label):
    try:
        value = yaml.load(text, Loader=UniqueLoader)
    except (yaml.YAMLError, IntegrityError) as exc:
        raise IntegrityError(f"{label}: {exc}") from exc
    if not isinstance(value, dict):
        raise IntegrityError(f"{label}: expected a YAML mapping")
    return value


def read_yaml(path):
    try:
        return parse_yaml(path.read_text(encoding="utf-8"), str(path))
    except OSError as exc:
        raise IntegrityError(f"{path}: {exc}") from exc


def read_frontmatter(path):
    try:
        lines = path.read_text(encoding="utf-8").splitlines()
    except OSError as exc:
        raise IntegrityError(f"{path}: {exc}") from exc
    if not lines or lines[0].strip() != "---":
        raise IntegrityError(f"{path}: missing opening front-matter delimiter")
    try:
        end = next(i for i in range(1, len(lines)) if lines[i].strip() == "---")
    except StopIteration as exc:
        raise IntegrityError(f"{path}: missing closing front-matter delimiter") from exc
    return parse_yaml("\n".join(lines[1:end]), str(path))


def validate_rules_document(doc, label):
    required_top = {"schema_id", "ruleset_id", "status", "description", "rules", "validation"}
    if not isinstance(doc, dict) or set(doc) != required_top:
        raise IntegrityError(f"{label}: malformed relationship-rules document fields")
    if doc["schema_id"] != SCHEMA_ID:
        raise IntegrityError(f"{label}: unexpected schema_id")
    if not isinstance(doc["ruleset_id"], str) or not doc["ruleset_id"]:
        raise IntegrityError(f"{label}: missing ruleset_id")
    if doc["status"] != "EXPERIMENTAL / NOT_ADOPTED":
        raise IntegrityError(f"{label}: unexpected ruleset status")
    if not isinstance(doc["description"], str) or not doc["description"].strip():
        raise IntegrityError(f"{label}: missing ruleset description")
    entries = doc["rules"]
    if not isinstance(entries, list) or not entries:
        raise IntegrityError(f"{label}: rules must be a non-empty list")
    rules = {}
    required_rule_fields = {"predicate", "subject_class", "object_class", "object_slot", "evidence_required"}
    for item in entries:
        if not isinstance(item, dict) or set(item) != required_rule_fields:
            raise IntegrityError(f"{label}: malformed rule fields")
        predicate = item["predicate"]
        if not isinstance(predicate, str) or not predicate:
            raise IntegrityError(f"{label}: rule missing predicate")
        if predicate not in EXPECTED_RULES:
            raise IntegrityError(f"{label}: unknown predicate {predicate!r}")
        if predicate in rules:
            raise IntegrityError(f"{label}: duplicate predicate {predicate}")
        expected_subject, expected_object = EXPECTED_RULES[predicate]
        if item["subject_class"] != expected_subject or item["object_class"] != expected_object:
            raise IntegrityError(f"{label}: {predicate} has incorrect subject/object classes")
        if item["object_slot"] != "object_ref":
            raise IntegrityError(f"{label}: {predicate} must use object_slot object_ref")
        if type(item["evidence_required"]) is not bool or item["evidence_required"] is not True:
            raise IntegrityError(f"{label}: {predicate} must require evidence (boolean true)")
        rules[predicate] = item
    if set(rules) != set(EXPECTED_RULES):
        missing = sorted(set(EXPECTED_RULES) - set(rules))
        raise IntegrityError(f"{label}: missing required predicates {missing}")
    validation = doc["validation"]
    if not isinstance(validation, dict) or set(validation) != {"fail_closed", "rdf_expectations"}:
        raise IntegrityError(f"{label}: malformed validation contract")
    fail_closed = validation["fail_closed"]
    if not isinstance(fail_closed, list) or any(not isinstance(x, str) for x in fail_closed):
        raise IntegrityError(f"{label}: fail_closed must be a string list")
    if len(fail_closed) != len(set(fail_closed)) or set(fail_closed) != REQUIRED_FAIL_CLOSED:
        raise IntegrityError(f"{label}: fail_closed contract is incomplete or contains unsupported checks")
    rdf_expectations = validation["rdf_expectations"]
    if not isinstance(rdf_expectations, dict) or set(rdf_expectations) != set(EXPECTED_RDF_EXPECTATIONS):
        raise IntegrityError(f"{label}: RDF expectations do not match the executable contract")
    for key, expected_value in EXPECTED_RDF_EXPECTATIONS.items():
        actual_value = rdf_expectations[key]
        if type(actual_value) is not type(expected_value) or actual_value != expected_value:
            raise IntegrityError(f"{label}: RDF expectations do not match the executable contract")
    return rules


def read_rules(path):
    return validate_rules_document(read_yaml(path), str(path))


def as_list(value, where):
    if value is None:
        return []
    if not isinstance(value, list):
        raise IntegrityError(f"{where}: expected a list")
    return value


def collect_corpus(data):
    records = [("KnowledgeCorpus", data)]
    slots = (
        ("sources", "Source"),
        ("source_locators", "SourceLocator"),
        ("evidence_records", "Evidence"),
        ("domain_nodes", "DomainNode"),
        ("knowledge_claims", "KnowledgeClaim"),
        ("relationship_assertions", "RelationshipAssertion"),
    )
    for slot, cls in slots:
        for item in as_list(data.get(slot, []), slot):
            if not isinstance(item, dict):
                raise IntegrityError(f"{slot}: each record must be a mapping")
            records.append((cls, item))
    return records


def collect_bundle(bundle, source, locator, evidence):
    records = [
        ("Source", source), ("SourceLocator", locator),
        ("Evidence", evidence), ("ClaimBundle", bundle),
    ]
    claim = bundle.get("claim")
    if not isinstance(claim, dict):
        raise IntegrityError("ClaimBundle.claim must be a mapping")
    records.append(("KnowledgeClaim", claim))
    for slot, cls in (("domain_nodes", "DomainNode"),
                      ("relationship_assertions", "RelationshipAssertion")):
        for item in as_list(bundle.get(slot, []), "ClaimBundle." + slot):
            if not isinstance(item, dict):
                raise IntegrityError("ClaimBundle." + slot + ": expected mappings")
            records.append((cls, item))
    return records


def require_curie(value, where):
    if (not isinstance(value, str) or not value.startswith("se:")
            or value == "se:" or value.count(":") != 1
            or any(ch.isspace() for ch in value)):
        raise IntegrityError(f"{where}: expected canonical se: CURIE, got {value!r}")
    return value


def validate_records(records, rules):
    index = {}
    counts = {}
    for cls, record in records:
        if not isinstance(record, dict):
            raise IntegrityError(f"{cls}: record must be a mapping")
        rid = require_curie(record.get("id"), cls + ".id")
        if rid in index:
            raise IntegrityError(f"duplicate identifier {rid}: {index[rid][0]} and {cls}")
        index[rid] = (cls, record)
        counts[cls] = counts.get(cls, 0) + 1

    def resolve(value, expected, where):
        ref = require_curie(value, where)
        target = index.get(ref)
        if target is None:
            raise IntegrityError(f"{where}: dangling reference {ref}")
        if target[0] != expected:
            raise IntegrityError(f"{where}: {ref} resolves to {target[0]}, expected {expected}")
        return target[1]

    locator_keys = {}
    for cls, record in records:
        rid = record["id"]
        if cls == "SourceLocator":
            source = require_curie(record.get("source_ref"), rid + ".source_ref")
            resolve(source, "Source", rid + ".source_ref")
            locator = record.get("locator")
            if not isinstance(locator, str) or not locator:
                raise IntegrityError(f"{rid}.locator: expected non-empty string")
            key = (source, locator)
            if key in locator_keys:
                raise IntegrityError(
                    f"duplicate source/locator compound key: {locator_keys[key]} and {rid}"
                )
            locator_keys[key] = rid
        elif cls == "Evidence":
            resolve(record.get("source_locator_ref"), "SourceLocator",
                    rid + ".source_locator_ref")
        elif cls == "DomainNode":
            resolve(record.get("source_ref"), "Source", rid + ".source_ref")
            resolve(record.get("source_locator_ref"), "SourceLocator",
                    rid + ".source_locator_ref")
        elif cls == "KnowledgeClaim":
            for ref in as_list(record.get("source_refs", []), rid + ".source_refs"):
                resolve(ref, "Source", rid + ".source_refs")
            for ref in as_list(record.get("domain_node_refs", []), rid + ".domain_node_refs"):
                resolve(ref, "DomainNode", rid + ".domain_node_refs")
        elif cls == "RelationshipAssertion":
            predicate = record.get("predicate")
            rule = rules.get(predicate) if isinstance(predicate, str) else None
            if rule is None:
                raise IntegrityError(f"{rid}.predicate: unsupported predicate {predicate!r}")
            resolve(record.get("subject_ref"), rule["subject_class"], rid + ".subject_ref")
            resolve(record.get("object_ref"), rule["object_class"],
                    rid + ".object_ref (" + predicate + ")")
            evidence_ref = record.get("evidence_ref")
            if rule.get("evidence_required", False) and evidence_ref is None:
                raise IntegrityError(f"{rid}.evidence_ref: required by {predicate}")
            if evidence_ref is not None:
                resolve(evidence_ref, "Evidence", rid + ".evidence_ref")
    return counts


def require_exact_se_class(graph, node, expected, label, role):
    type_values = set(graph.objects(node, RDF.type))
    if any(not isinstance(value, URIRef) for value in type_values):
        raise IntegrityError(f"{label}: rdf:type assertion for {role} must have a URIRef class")
    expected_uri = URIRef(SE_BASE + expected)
    allowed = {expected_uri}
    allowed.update(URIRef(SE_BASE + ancestor) for ancestor in CLASS_ANCESTORS.get(expected, set()))
    if expected_uri not in type_values or not type_values.issubset(allowed):
        found = sorted(str(value) for value in type_values)
        raise IntegrityError(
            f"{label}: rdf:{role} must have exactly class {expected} "
            f"(allowing declared ancestors), found {found}"
        )


def validate_rdf(graph, rules, label, records=None):
    for node, rdf_type in graph.subject_objects(RDF.type):
        if not isinstance(rdf_type, URIRef):
            raise IntegrityError(f"{label}: rdf:type object for {node} must be a URIRef class")
    statements = list(graph.subjects(RDF.type, RDF.Statement))
    if not statements:
        raise IntegrityError(f"{label}: no rdf:Statement nodes found")
    reified_edge_nodes = (
        set(graph.subjects(RDF.subject, None))
        | set(graph.subjects(RDF.predicate, None))
        | set(graph.subjects(RDF.object, None))
    )
    untyped_edge_nodes = reified_edge_nodes - set(statements)
    if untyped_edge_nodes:
        found = sorted(str(value) for value in untyped_edge_nodes)
        raise IntegrityError(
            f"{label}: untyped reification edge nodes are not allowed: {found}"
        )
    expected = {}
    for cls, record in records or []:
        if cls != "RelationshipAssertion":
            continue
        def expand(value):
            return URIRef(SE_BASE + require_curie(value, "RDF expected reference").split(":", 1)[1])
        expected[expand(record["id"])] = {
            "subject": expand(record["subject_ref"]),
            "predicate": URIRef(SE_BASE + record["predicate"]),
            "object": expand(record["object_ref"]),
            "evidence": expand(record["evidence_ref"]) if record.get("evidence_ref") else None,
        }
    if records is not None and set(statements) != set(expected):
        raise IntegrityError(f"{label}: RDF statement identifiers do not exactly match YAML relationship IDs")
    for node in statements:
        if not isinstance(node, URIRef) or not str(node).startswith(SE_BASE):
            raise IntegrityError(f"{label}: rdf:Statement identifier is not a canonical se: URIRef")
        statement_types = set(graph.objects(node, RDF.type))
        if statement_types != {RDF.Statement}:
            found = sorted(str(value) for value in statement_types)
            raise IntegrityError(
                f"{label}: rdf:Statement node must have exactly rdf:Statement type, found {found}"
            )
        def one(prop, name):
            values = list(graph.objects(node, prop))
            if len(values) != 1:
                raise IntegrityError(
                    f"{label}: {node} requires exactly one {name}, found {len(values)}"
                )
            return values[0]

        subject = one(RDF.subject, "rdf:subject")
        predicate = one(RDF.predicate, "rdf:predicate")
        obj = one(RDF.object, "rdf:object")
        if not isinstance(subject, URIRef):
            raise IntegrityError(f"{label}: rdf:subject is not a URIRef")
        if not isinstance(predicate, URIRef):
            raise IntegrityError(f"{label}: rdf:predicate is not a URIRef")
        if not isinstance(obj, URIRef):
            raise IntegrityError(f"{label}: rdf:object is not a URIRef resource (Literal rejected)")
        if not str(subject).startswith(SE_BASE) or not str(obj).startswith(SE_BASE):
            raise IntegrityError(f"{label}: subject/object outside canonical se: namespace")
        if records is not None:
            expected_edge = expected[node]
            if subject != expected_edge["subject"] or predicate != expected_edge["predicate"] or obj != expected_edge["object"]:
                raise IntegrityError(f"{label}: generated RDF edge does not preserve the exact YAML subject/predicate/object CURIE identity for {node}")
        local = str(predicate).removeprefix(SE_BASE)
        rule = rules.get(local)
        if rule is None:
            raise IntegrityError(f"{label}: unsupported RDF predicate {predicate}")
        require_exact_se_class(graph, subject, rule["subject_class"], label, "subject")
        require_exact_se_class(graph, obj, rule["object_class"], label, "object")
        evidence = list(graph.objects(node, PROV.wasDerivedFrom))
        if rule.get("evidence_required", False) and not evidence:
            raise IntegrityError(f"{label}: missing prov:wasDerivedFrom evidence")
        for evidence_node in evidence:
            if not isinstance(evidence_node, URIRef):
                raise IntegrityError(f"{label}: evidence target must be an Evidence URIRef")
            require_exact_se_class(graph, evidence_node, "Evidence", label, "evidence")
        if records is not None:
            expected_evidence = expected[node]["evidence"]
            if (expected_evidence is not None and evidence != [expected_evidence]) or (expected_evidence is None and evidence):
                raise IntegrityError(f"{label}: generated RDF evidence reference does not match YAML evidence_ref for {node}")
    return len(statements)


def sample_records():
    return [
        ("Source", {"id": "se:S1"}),
        ("SourceLocator", {"id": "se:L1", "source_ref": "se:S1", "locator": "p.1"}),
        ("Evidence", {"id": "se:E1", "source_locator_ref": "se:L1"}),
        ("DomainNode", {"id": "se:D1", "source_ref": "se:S1", "source_locator_ref": "se:L1"}),
        ("KnowledgeClaim", {"id": "se:C1", "source_refs": ["se:S1"], "domain_node_refs": ["se:D1"]}),
        ("RelationshipAssertion", {"id": "se:R1", "subject_ref": "se:C1", "predicate": "SUPPORTED_BY", "object_ref": "se:E1", "evidence_ref": "se:E1"}),
        ("RelationshipAssertion", {"id": "se:R2", "subject_ref": "se:C1", "predicate": "INCLUDES", "object_ref": "se:D1", "evidence_ref": "se:E1"}),
    ]


def self_test(rules):
    valid_records = sample_records()
    validate_records(valid_records, rules)
    print("SELF-TEST PASS: valid YAML reference closure")
    passed = 1
    rule_doc = {
        "schema_id": SCHEMA_ID,
        "ruleset_id": "se-linkml-pof-relationship-rules-0.1.0",
        "status": "EXPERIMENTAL / NOT_ADOPTED",
        "description": "Self-test ruleset.",
        "rules": [copy.deepcopy(rules[name]) for name in ("SUPPORTED_BY", "INCLUDES")],
        "validation": {
            "fail_closed": sorted(REQUIRED_FAIL_CLOSED),
            "rdf_expectations": dict(EXPECTED_RDF_EXPECTATIONS),
        },
    }
    rule_cases = (
        ("duplicate rule predicate", lambda d: d["rules"].append(copy.deepcopy(d["rules"][0])), "duplicate predicate"),
        ("unknown rule predicate", lambda d: d["rules"][0].update(predicate="CONTRADICTS"), "unknown predicate"),
        ("malformed rule", lambda d: d["rules"][0].pop("object_slot"), "malformed rule fields"),
        ("duplicate fail_closed entry", lambda d: d["validation"]["fail_closed"].append(d["validation"]["fail_closed"][0]), "fail_closed contract is incomplete"),
        ("incomplete fail_closed contract", lambda d: d["validation"]["fail_closed"].remove("unknown_predicate"), "fail_closed contract is incomplete"),
        ("non-boolean RDF expectation", lambda d: d["validation"]["rdf_expectations"].update(reject_object_literal=1), "RDF expectations do not match"),
        ("missing required predicate", lambda d: d["rules"].pop(), "missing required predicates"),
    )
    for name, mutate, expected in rule_cases:
        candidate = copy.deepcopy(rule_doc)
        mutate(candidate)
        try:
            validate_rules_document(candidate, "self-test-" + name)
        except IntegrityError as exc:
            if expected not in str(exc):
                raise IntegrityError(f"self-test {name}: unexpected diagnostic: {exc}") from exc
            print("SELF-TEST PASS: " + name)
            passed += 1
        else:
            raise IntegrityError("self-test " + name + ": invalid rules accepted")

    cases = (
        ("duplicate id", lambda r: r.append(("Evidence", {"id": "se:E1"})), "duplicate identifier"),
        ("dangling object", lambda r: r[-2][1].update(object_ref="se:MISSING"), "dangling reference"),
        ("dangling source", lambda r: r[1][1].update(source_ref="se:MISSING"), "dangling reference"),
        ("wrong endpoint class", lambda r: r[-2][1].update(object_ref="se:D1"), "expected Evidence"),
        ("unknown predicate", lambda r: r[-2][1].update(predicate="CONTRADICTS"), "unsupported predicate"),
        ("duplicate compound key", lambda r: r.insert(2, ("SourceLocator", {"id": "se:L2", "source_ref": "se:S1", "locator": "p.1"})), "compound key"),
        ("bare record id", lambda r: r[0][1].update(id="S1"), "canonical se: CURIE"),
        ("bare typed reference", lambda r: r[1][1].update(source_ref="S1"), "canonical se: CURIE"),
        ("missing required evidence", lambda r: r[-2][1].pop("evidence_ref"), "required by SUPPORTED_BY"),
    )
    for name, mutate, expected in cases:
        records = copy.deepcopy(sample_records())
        mutate(records)
        try:
            validate_records(records, rules)
        except IntegrityError as exc:
            if expected not in str(exc):
                raise IntegrityError(f"self-test {name}: unexpected diagnostic: {exc}") from exc
            print("SELF-TEST PASS: " + name)
            passed += 1
        else:
            raise IntegrityError("self-test " + name + ": invalid data accepted")

    graph = Graph()
    stmt, claim, evidence = (URIRef(SE_BASE + x) for x in ("R1", "C1", "E1"))
    graph.add((stmt, RDF.type, RDF.Statement))
    graph.add((stmt, RDF.subject, claim))
    graph.add((stmt, RDF.predicate, SE.SUPPORTED_BY))
    graph.add((stmt, RDF.object, evidence))
    graph.add((stmt, PROV.wasDerivedFrom, evidence))
    graph.add((claim, RDF.type, SE.KnowledgeClaim))
    graph.add((evidence, RDF.type, SE.Evidence))
    validate_rdf(graph, rules, "self-test-valid-rdf", [r for r in sample_records() if r[0] != "RelationshipAssertion" or r[1]["id"] == "se:R1"])
    print("SELF-TEST PASS: RDF URIRef positive")
    passed += 1

    try:
        parse_yaml("id: se:ONE\nid: se:TWO\n", "self-test-duplicate-yaml")
    except IntegrityError as exc:
        if "duplicate YAML key" not in str(exc):
            raise
        print("SELF-TEST PASS: duplicate YAML key rejected")
        passed += 1
    else:
        raise IntegrityError("self-test: duplicate YAML key accepted")

    bad = Graph()
    for triple in graph:
        bad.add(triple)
    bad.remove((stmt, RDF.object, evidence))
    bad.add((stmt, RDF.object, Literal("se:E1")))
    try:
        validate_rdf(bad, rules, "self-test-literal", [r for r in sample_records() if r[0] != "RelationshipAssertion" or r[1]["id"] == "se:R1"])
    except IntegrityError as exc:
        if "not a URIRef" not in str(exc):
            raise
        print("SELF-TEST PASS: RDF Literal rejected")
        passed += 1
    else:
        raise IntegrityError("self-test: RDF Literal accepted")

    bad = Graph()
    for triple in graph:
        bad.add(triple)
    bad.add((evidence, RDF.type, Literal("Evidence")))
    try:
        validate_rdf(bad, rules, "self-test-literal-rdf-type", [r for r in sample_records() if r[0] != "RelationshipAssertion" or r[1]["id"] == "se:R1"])
    except IntegrityError as exc:
        if "rdf:type object for" not in str(exc):
            raise
        print("SELF-TEST PASS: non-URIRef rdf:type rejected")
        passed += 1
    else:
        raise IntegrityError("self-test: non-URIRef rdf:type accepted")

    bad = Graph()
    for triple in graph:
        bad.add(triple)
    bad.remove((evidence, RDF.type, SE.Evidence))
    bad.add((evidence, RDF.type, SE.DomainNode))
    try:
        validate_rdf(bad, rules, "self-test-wrong-class", [r for r in sample_records() if r[0] != "RelationshipAssertion" or r[1]["id"] == "se:R1"])
    except IntegrityError as exc:
        if "rdf:object must have exactly class Evidence" not in str(exc):
            raise
        print("SELF-TEST PASS: RDF wrong endpoint class rejected")
        passed += 1
    else:
        raise IntegrityError("self-test: wrong RDF endpoint class accepted")
    bad = Graph()
    for triple in graph:
        bad.add(triple)
    bad.remove((claim, RDF.type, SE.KnowledgeClaim))
    bad.add((claim, RDF.type, SE.Evidence))
    try:
        validate_rdf(bad, rules, "self-test-wrong-subject-class", [r for r in sample_records() if r[0] != "RelationshipAssertion" or r[1]["id"] == "se:R1"])
    except IntegrityError as exc:
        if "rdf:subject must have exactly class KnowledgeClaim" not in str(exc):
            raise
        print("SELF-TEST PASS: RDF wrong subject class rejected")
        passed += 1
    else:
        raise IntegrityError("self-test: wrong RDF subject class accepted")

    bad = Graph()
    for triple in graph:
        bad.add(triple)
    bad.add((evidence, RDF.type, SE.KnowledgeClaim))
    try:
        validate_rdf(bad, rules, "self-test-extra-object-class", [r for r in sample_records() if r[0] != "RelationshipAssertion" or r[1]["id"] == "se:R1"])
    except IntegrityError as exc:
        if "rdf:object must have exactly class Evidence" not in str(exc):
            raise
        print("SELF-TEST PASS: RDF incompatible additional object class rejected")
        passed += 1
    else:
        raise IntegrityError("self-test: incompatible additional object class accepted")

    bad = Graph()
    for triple in graph:
        bad.add(triple)
    bad.add((claim, RDF.type, SE.Evidence))
    try:
        validate_rdf(bad, rules, "self-test-extra-subject-class", [r for r in sample_records() if r[0] != "RelationshipAssertion" or r[1]["id"] == "se:R1"])
    except IntegrityError as exc:
        if "rdf:subject must have exactly class KnowledgeClaim" not in str(exc):
            raise
        print("SELF-TEST PASS: RDF incompatible additional subject class rejected")
        passed += 1
    else:
        raise IntegrityError("self-test: incompatible additional subject class accepted")

    bad = Graph()
    for triple in graph:
        bad.add(triple)
    bad.add((evidence, RDF.type, URIRef("https://example.org/ExternalClass")))
    try:
        validate_rdf(bad, rules, "self-test-foreign-object-class", [r for r in sample_records() if r[0] != "RelationshipAssertion" or r[1]["id"] == "se:R1"])
    except IntegrityError as exc:
        if "rdf:object must have exactly class Evidence" not in str(exc):
            raise
        print("SELF-TEST PASS: foreign-namespace additional object class rejected")
        passed += 1
    else:
        raise IntegrityError("self-test: foreign-namespace additional object class accepted")

    bad = Graph()
    for triple in graph:
        bad.add(triple)
    bad.add((stmt, RDF.type, Literal("Statement")))
    try:
        validate_rdf(bad, rules, "self-test-literal-statement-type", [r for r in sample_records() if r[0] != "RelationshipAssertion" or r[1]["id"] == "se:R1"])
    except IntegrityError as exc:
        if "rdf:type object for" not in str(exc):
            raise
        print("SELF-TEST PASS: non-URIRef rdf:type on statement node rejected")
        passed += 1
    else:
        raise IntegrityError("self-test: non-URIRef rdf:type on statement node accepted")

    bad = Graph()
    for triple in graph:
        bad.add(triple)
    bad.add((stmt, RDF.type, URIRef("https://example.org/ExternalStatement")))
    try:
        validate_rdf(bad, rules, "self-test-extra-statement-class", [r for r in sample_records() if r[0] != "RelationshipAssertion" or r[1]["id"] == "se:R1"])
    except IntegrityError as exc:
        if "rdf:Statement node must have exactly rdf:Statement type" not in str(exc):
            raise
        print("SELF-TEST PASS: incompatible additional statement class rejected")
        passed += 1
    else:
        raise IntegrityError("self-test: incompatible additional statement class accepted")

    bad = Graph()
    for triple in graph:
        bad.add(triple)
    untyped = URIRef(SE_BASE + "R_UNTYPED")
    bad.add((untyped, RDF.subject, claim))
    bad.add((untyped, RDF.predicate, SE.SUPPORTED_BY))
    bad.add((untyped, RDF.object, evidence))
    try:
        validate_rdf(bad, rules, "self-test-untyped-reification", [r for r in sample_records() if r[0] != "RelationshipAssertion" or r[1]["id"] == "se:R1"])
    except IntegrityError as exc:
        if "untyped reification edge nodes are not allowed" not in str(exc):
            raise
        print("SELF-TEST PASS: untyped reification edge node rejected")
        passed += 1
    else:
        raise IntegrityError("self-test: untyped reification edge node accepted")

    bad = Graph()
    for triple in graph:
        bad.add(triple)
    bad.remove((stmt, RDF.object, evidence))
    other = URIRef(SE_BASE + "D1")
    bad.add((stmt, RDF.object, other))
    bad.add((other, RDF.type, SE.DomainNode))
    try:
        validate_rdf(bad, rules, "self-test-edge-identity", [r for r in sample_records() if r[0] != "RelationshipAssertion" or r[1]["id"] == "se:R1"])
    except IntegrityError as exc:
        if "exact YAML subject/predicate/object CURIE identity" not in str(exc):
            raise
        print("SELF-TEST PASS: RDF edge identity mismatch rejected")
        passed += 1
    else:
        raise IntegrityError("self-test: RDF edge identity mismatch accepted")

    bad = Graph()
    for s, p, o in graph:
        bad.add((URIRef(SE_BASE + "WRONG") if s == stmt else s, p,
                 URIRef(SE_BASE + "WRONG") if o == stmt else o))
    try:
        validate_rdf(bad, rules, "self-test-statement-id", [r for r in sample_records() if r[0] != "RelationshipAssertion" or r[1]["id"] == "se:R1"])
    except IntegrityError as exc:
        if "statement identifiers do not exactly match" not in str(exc):
            raise
        print("SELF-TEST PASS: RDF statement ID mismatch rejected")
        passed += 1
    else:
        raise IntegrityError("self-test: RDF statement ID mismatch accepted")

    if passed != 31:
        raise IntegrityError(f"self-test coverage mismatch: expected 31 checks, observed {passed}")
    print(f"SELF-TEST SUMMARY: {passed}/31 PASS")
    return 0


def main():
    parser = argparse.ArgumentParser(description=__doc__)
    parser.add_argument("--rules", type=Path, default=Path(__file__).with_name("RELATIONSHIP_RULES.yaml"))
    parser.add_argument("--corpus", type=Path)
    parser.add_argument("--frontmatter", type=Path)
    parser.add_argument("--bundle", type=Path)
    parser.add_argument("--source", type=Path)
    parser.add_argument("--locator", type=Path)
    parser.add_argument("--evidence", type=Path)
    parser.add_argument("--ttl", action="append", type=Path, default=[])
    parser.add_argument("--self-test", action="store_true")
    args = parser.parse_args()
    try:
        rules = read_rules(args.rules)
        if args.self_test:
            return self_test(rules)
        modes = sum(bool(x) for x in (args.corpus, args.frontmatter, args.bundle))
        if modes != 1:
            raise IntegrityError("choose exactly one of --corpus, --frontmatter or --bundle")
        if args.corpus:
            records = collect_corpus(read_yaml(args.corpus))
            label = str(args.corpus)
        elif args.frontmatter:
            records = collect_corpus(read_frontmatter(args.frontmatter))
            label = str(args.frontmatter)
        else:
            if not all((args.source, args.locator, args.evidence)):
                raise IntegrityError("--bundle requires --source, --locator and --evidence")
            records = collect_bundle(read_yaml(args.bundle), read_yaml(args.source),
                                     read_yaml(args.locator), read_yaml(args.evidence))
            label = str(args.bundle)
        counts = validate_records(records, rules)
        print("YAML INTEGRITY PASS: " + label)
        print("RECORD COUNTS: " + ", ".join(f"{k}={v}" for k, v in sorted(counts.items())))
        for path in args.ttl:
            graph = Graph()
            graph.parse(path, format="turtle")
            checked = validate_rdf(graph, rules, str(path), records)
            print(f"RDF INTEGRITY PASS: {path} ({checked} relationship statements)")
        return 0
    except IntegrityError as exc:
        print("INTEGRITY FAIL: " + str(exc), file=sys.stderr)
        return 1
    except Exception as exc:
        print(f"INTEGRITY ERROR: {type(exc).__name__}: {exc}", file=sys.stderr)
        return 2


if __name__ == "__main__":
    raise SystemExit(main())
