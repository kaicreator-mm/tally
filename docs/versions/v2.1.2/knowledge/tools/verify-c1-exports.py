#!/usr/bin/env python3
# verify-c1-exports.py — independent rdflib parse-back verification of the
# Tally v2.1.2 C1 canonical exports (the authoritative equivalence evidence,
# complementing the native Node gates).
#
# Checks, in order:
#   1. graph.ttl parses with rdflib; graph.jsonld parses with rdflib
#   2. TTL triple set == JSON-LD triple set == builder-recorded canonical set
#      (compared as sorted canonical N-Triples, byte-for-byte)
#   3. the sorted-N-Triples SHA-256 matches manifest.sha256's digest line
#   4. sha256(graph.ttl) and sha256(graph.jsonld) match manifest.sha256
#   5. reference-valued slots expand to URIRefs; se:predicate values URIRefs;
#      no dangling record references within the node-id space
#   6. counts: 809 subjects with se: KnowledgeGraph root
#      se:GRAPH-V2.1.2-C1-OWNER-R13, 106 PENDING claims, 0 ReviewDecision
#   7. optional --also-v211: the frozen v2.1.1 exports still verify against
#      their own manifest (frozen-history regression anchor)
#
# Exit code 0 = all checks pass; any failure exits 1 with a FAIL line.
#
# Pinned environment: Python 3.14.7, rdflib 7.6.0 (the SEK-08/09H rebuild venv).
#   python tools/verify-c1-exports.py <repo-root> [--also-v211]

import hashlib
import json
import sys
from pathlib import Path

from rdflib import Graph, URIRef

SE = 'https://github.com/kaicreator-mm/tally#'
RDF_TYPE = URIRef('http://www.w3.org/1999/02/22-rdf-syntax-ns#type')

REF_KEYS = {
    'evidence_refs', 'claim_refs', 'subject_ref', 'object_ref', 'source_ref',
    'source_locator_ref', 'target_scope_ref', 'domain_ref', 'view_ref', 'predicate',
}


def is_ref_pred(local):
    return local in REF_KEYS or local.endswith('_ref') or local.endswith('_refs')


def fail(msg):
    print('FAIL: ' + msg)
    sys.exit(1)


def nt_lines(g):
    return sorted(g.serialize(format='nt').strip().split('\n'))


def digest_of(lines):
    return hashlib.sha256(('\n'.join(lines) + '\n').encode('utf-8')).hexdigest()


def verify_pair(root, ttl_rel, jsonld_rel, manifest_rel, expected_root_id,
                expected_nodes, label):
    ttl_path = root / ttl_rel
    jsonld_path = root / jsonld_rel
    manifest_path = root / manifest_rel

    g_ttl = Graph()
    g_ttl.parse(ttl_path.as_posix(), format='turtle')
    g_jld = Graph()
    g_jld.parse(jsonld_path.as_posix(), format='json-ld')

    ttl_nt = nt_lines(g_ttl)
    jld_nt = nt_lines(g_jld)
    if ttl_nt != jld_nt:
        fail(f'{label}: TTL and JSON-LD triple sets differ '
             f'({len(ttl_nt)} vs {len(jld_nt)} lines)')
    triples = len(ttl_nt)

    digest = digest_of(ttl_nt)
    manifest = manifest_path.read_text(encoding='utf-8')
    if f'# canonical graph digest (sha256 over sorted N-Triples): {digest}' not in manifest:
        fail(f'{label}: canonical digest {digest} not in manifest')
    for fname in ('graph.ttl', 'graph.jsonld'):
        got = hashlib.sha256((root / manifest_path.parent / fname).read_bytes()).hexdigest()
        if f'{got}  {fname}' not in manifest:
            fail(f'{label}: file hash for {fname} missing/mismatched in manifest')

    subjects = {s for s in g_ttl.subjects()}
    if len(subjects) != expected_nodes:
        fail(f'{label}: {len(subjects)} subjects != {expected_nodes}')
    typed = {}
    for s, o in g_ttl.subject_objects(RDF_TYPE):
        typed.setdefault(s, o)
    roots = [s for s, o in typed.items() if o == URIRef(SE + 'KnowledgeGraph')]
    if len(roots) != 1 or str(roots[0]) != SE + expected_root_id:
        fail(f'{label}: graph root wrong: {roots}')

    # reference slots resolve to URIRefs and to known record ids
    ids = set()
    for s in subjects:
        if isinstance(s, URIRef) and str(s).startswith(SE):
            ids.add('se:' + str(s)[len(SE):])
    for pred in set(g_ttl.predicates(None, None)):
        local = str(pred)[len(SE):] if str(pred).startswith(SE) else None
        if local is None:
            continue
        if not is_ref_pred(local):
            continue
        for s, o in g_ttl.subject_objects(pred):
            if not isinstance(o, URIRef):
                fail(f'{label}: ref slot se:{local} on {s} has a literal object')
            if local == 'predicate':
                continue  # catalog meaning IRI space, not record-id space
            rid = 'se:' + str(o)[len(SE):] if str(o).startswith(SE) else str(o)
            if rid.startswith('se:') and rid not in ids:
                fail(f'{label}: dangling ref {s} se:{local} -> {rid}')

    claims = [s for s, o in g_ttl.subject_objects(RDF_TYPE)
              if o == URIRef(SE + 'KnowledgeClaim')]
    def claim_status(c):
        v = g_ttl.value(c, URIRef(SE + 'review_status'))
        return str(v) if v is not None else None
    pending = [c for c in claims if claim_status(c) == 'PENDING']
    if len(claims) != 106 or len(pending) != 106:
        fail(f'{label}: expected 106 PENDING KnowledgeClaims, got {len(pending)}/{len(claims)}')
    decisions = [s for s, o in g_ttl.subject_objects(RDF_TYPE)
                 if o == URIRef(SE + 'ReviewDecision')]
    if decisions:
        fail(f'{label}: ReviewDecision present: {decisions}')

    print(f'VERIFY PASS [{label}]: ttl_triples={triples} jsonld_triples={triples} '
          f'digest={digest} subjects={len(subjects)} claims=106/106 PENDING '
          f'review_decision=0 root=se:{expected_root_id}')


def main():
    root = Path(sys.argv[1]) if len(sys.argv) > 1 else Path('.')
    also_v211 = '--also-v211' in sys.argv
    verify_pair(
        root,
        'docs/versions/v2.1.2/knowledge/generated/graph.ttl',
        'docs/versions/v2.1.2/knowledge/generated/graph.jsonld',
        'docs/versions/v2.1.2/knowledge/generated/manifest.sha256',
        'GRAPH-V2.1.2-C1-OWNER-R13', 809, 'v2.1.2-c1')
    if also_v211:
        verify_pair(
            root,
            'docs/versions/v2.1.1/knowledge/generated/graph.ttl',
            'docs/versions/v2.1.1/knowledge/generated/graph.jsonld',
            'docs/versions/v2.1.1/knowledge/generated/manifest.sha256',
            'GRAPH-V2.1.1-SEK08', 807, 'frozen-v2.1.1')
    print('rdflib verification: ALL PASS')


if __name__ == '__main__':
    main()
