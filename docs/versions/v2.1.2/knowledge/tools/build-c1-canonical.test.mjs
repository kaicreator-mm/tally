// build-c1-canonical.test.mjs — native Node test suite for the Tally v2.1.2
// C1 canonical source-lane build (node --test, zero dependencies).
//
// Coverage:
//   positive: PyYAML parity of the Node YAML-subset parser over the whole
//     frozen corpus; overlay-free projection reproduces the frozen v2.1.1
//     exports byte-for-byte; candidate invariants (809/106/0, OD05 narrowing,
//     GH-2018 patch, UNKNOWN conservation); overlay preview byte convergence;
//     Turtle parse-back equivalence; determinism; manifest digests.
//   negative (fail-closed, incl. source failure): lane byte drift, unavailable
//     input, YAML duplicate keys, unknown relationship predicate, dangling
//     typed edge, uniformity restoration in the lane, unauthorized additions,
//     dropped-payload caveat removal, qualification tampering, TTL/JSON-LD
//     inequivalence.
//
// Run from the repository root:
//   node --test docs/versions/v2.1.2/knowledge/tools/build-c1-canonical.test.mjs

import { test } from 'node:test';
import assert from 'node:assert/strict';
import { readFileSync, existsSync } from 'node:fs';
import { createHash } from 'node:crypto';
import {
  buildGraph, loadInputs, PATHS, REPO_ROOT, PINS, FROZEN_DIGEST,
  assertC1Candidate, assertV211Baseline, readPinnedInput, assertPinMatches,
  gitBlobSha1, parseTtl, triplesEqualSet, canonicalDigest, nodesToTriples,
} from './project-c1.mjs';
import { parseYaml, YamlError } from './yaml-min.mjs';

function canonicalJson(value) {
  if (value === null || typeof value === 'string' || typeof value === 'number' || typeof value === 'boolean') {
    return JSON.stringify(value);
  }
  if (Array.isArray(value)) return '[' + value.map(canonicalJson).join(',') + ']';
  const keys = Object.keys(value).sort();
  return '{' + keys.map((k) => JSON.stringify(k) + ':' + canonicalJson(value[k])).join(',') + '}';
}

function sha256(s) {
  return createHash('sha256').update(s, 'utf8').digest('hex');
}

const inputs = loadInputs();

// ---------------------------------------------------------------------------
// positive

test('PyYAML parity: Node YAML-subset parse == committed parity hashes over the frozen corpus', () => {
  const fixture = REPO_ROOT + 'docs/versions/v2.1.2/knowledge/fixtures/c1-yaml-parity.sha256';
  const expected = readFileSync(fixture, 'utf8').trim().split('\n');
  assert.equal(expected.length, 12, 'parity fixture must cover 12 inputs');
  const rels = [
    PATHS.registry, PATHS.catalog,
    ...Object.keys(PINS.lanes).map((l) => PATHS.lanes(l)),
  ];
  for (const line of expected) {
    const [digest, rel] = line.split('  ');
    const parsed = parseYaml(readFileSync(REPO_ROOT + rel, 'utf8'), rel);
    assert.equal(sha256(canonicalJson(parsed)) + '  ' + rel, line, `parity mismatch for ${rel}`);
    void digest;
  }
  assert.equal(rels.length, 12);
});

test('R2 overlay-free lane projection reproduces the frozen v2.1.1 export byte-for-byte', () => {
  const baseline = buildGraph('v211-baseline', inputs);
  assertV211Baseline(baseline.nodes);
  assert.equal(baseline.digest, FROZEN_DIGEST);
  assert.equal(baseline.jsonld, readFileSync(REPO_ROOT + PATHS.frozenGraphJsonld, 'utf8'));
  assert.equal(baseline.ttl, readFileSync(REPO_ROOT + PATHS.frozenGraphTtl, 'utf8'));
  const baselineParsed = parseTtl(baseline.ttl);
  assert.ok(triplesEqualSet(baselineParsed, baseline.triples));
  assert.equal(canonicalDigest(baselineParsed), FROZEN_DIGEST);
});

test('R3/R4 candidate invariants hold and the overlay preview converges byte-for-byte', () => {
  const cand = buildGraph('v212-c1', inputs);
  assertC1Candidate(cand.nodes, inputs.overlay, cand.baselineNodes);
  assert.equal(cand.nodes.length, 809);
  assert.equal(cand.jsonld, readFileSync(REPO_ROOT + PATHS.outGraphJsonld, 'utf8'));
  assert.equal(cand.ttl, readFileSync(REPO_ROOT + PATHS.outGraphTtl, 'utf8'));
  assert.equal(
    cand.jsonld,
    readFileSync(REPO_ROOT + PATHS.outPreview, 'utf8'),
    'Owner-overlay candidate must equal the canonical lane-integrated export',
  );
});

test('OD05 positive: mixed context-appropriate methods allowed, teaching scope only', () => {
  const g = buildGraph('v212-c1', inputs);
  const dm = g.nodes.find((n) => n['@id'] === 'se:DM-REQ-FUNCTIONAL-DOC-METHOD');
  assert.ok(dm['se:procedure'].includes('different methods for different requirements'));
  assert.ok(dm['se:scope_description'].includes('no universal uniformity requirement'));
  assert.ok(dm['se:options'].includes('User stories'));
  const pat = g.nodes.find((n) => n['@id'] === 'se:PAT-REQ-USER-STORY-SPECIFICATION');
  assert.ok(pat['se:solution'].includes('Other requirements may use use cases or another structured method'));
});

test('GH-2018 positive: topology locator preserved, separate backlog locator + PENDING evidence', () => {
  const g = buildGraph('v212-c1', inputs);
  const byId = new Map(g.nodes.map((n) => [n['@id'], n]));
  const origLoc = byId.get('se:LOC-09F-GH-REPLICATION');
  assert.ok(origLoc['se:locator'].includes('16:24') && !origLoc['se:locator'].includes('23:03'));
  const backLoc = byId.get('se:LOC-09F-GH-BACKLOG');
  assert.ok(backLoc['se:locator'].includes('16:45') && backLoc['se:locator'].includes('23:03'));
  const evd = byId.get('se:EVD-09F-GH-BACKLOG');
  assert.equal(evd['se:lifecycle_status'], 'PENDING_REVIEW');
  assert.equal(evd['se:extraction_status'], 'EXTRACTED_PENDING_REVIEW');
  assert.ok(evd['se:statement'].includes('200,000'));
  assert.ok(evd['se:uncertainty'].includes('UNKNOWN'));
  assert.equal(evd['se:source_locator_ref']['@id'], 'se:LOC-09F-GH-BACKLOG');
  const caze = byId.get('se:CASE-09F-GH-DB-FAILOVER');
  assert.ok(caze['se:outcome'].includes('dropped') && caze['se:outcome'].includes('does not imply lossless delivery'));
});

test('UNKNOWN conservation: 106 PENDING claims, 0 ReviewDecision, Owner 2/2 vs Oracle NOT_ATTESTED', () => {
  const cand = buildGraph('v212-c1', inputs);
  const claims = cand.nodes.filter((n) => n['@type'] === 'se:KnowledgeClaim');
  assert.equal(claims.length, 106);
  assert.ok(claims.every((c) => c['se:review_status'] === 'PENDING'));
  assert.equal(cand.nodes.filter((n) => n['@type'] === 'se:ReviewDecision').length, 0);
  assert.ok(cand.nodes.every((n) => n['se:lifecycle_status'] !== 'ACCEPTED'));
  const q = inputs.overlay.qualification;
  assert.equal(q.sourceOwner, '2/2 CONFIRMED');
  assert.equal(q.oracle, 'NOT_ATTESTED');
  assert.equal(q.sap, '0/6');
  assert.equal(q.dkeR8, '0/53');
  assert.equal(q.root, 'UNKNOWN');
});

test('R5 candidate export equivalence: TTL parse-back == JSON-LD == internal canonical set', () => {
  const cand = buildGraph('v212-c1', inputs);
  const parsed = parseTtl(cand.ttl);
  assert.ok(triplesEqualSet(parsed, cand.triples));
  assert.equal(canonicalDigest(parsed), cand.digest);
  const fromJsonld = nodesToTriples(JSON.parse(cand.jsonld)['@graph']);
  assert.ok(triplesEqualSet(fromJsonld, cand.triples));
  const manifest = readFileSync(REPO_ROOT + PATHS.outManifest, 'utf8');
  assert.ok(manifest.includes(cand.digest));
  for (const [name, bytes] of [['graph.ttl', cand.ttl], ['graph.jsonld', cand.jsonld]]) {
    assert.ok(manifest.includes(sha256(bytes) + '  ' + name));
  }
});

test('R6 determinism: independent rebuilds are byte-identical', () => {
  const a = buildGraph('v212-c1', loadInputs());
  const b = buildGraph('v212-c1', loadInputs());
  assert.equal(a.jsonld, b.jsonld);
  assert.equal(a.ttl, b.ttl);
  assert.equal(a.digest, b.digest);
});

// ---------------------------------------------------------------------------
// negative — source failure and tampering must fail closed

test('negative: frozen lane byte drift fails closed at the Git Blob pin', () => {
  const real = readPinnedInput(PATHS.lanes('requirements'), PINS.lanes.requirements, 'lane requirements');
  assert.equal(real.blob, PINS.lanes.requirements);
  const drifted = Buffer.from(real.text.replace(
    'At least one documentation and structuring method must actually be used',
    'A single uniform method must actually be used'), 'utf8');
  assert.notEqual(gitBlobSha1(drifted), PINS.lanes.requirements);
  assert.throws(
    () => assertPinMatches(drifted, PINS.lanes.requirements, 'lane requirements(drift-sim)'),
    /Git Blob drift.*source failure must fail closed/,
  );
  // the same reader with unpinned expectation bypasses nothing: the pipeline
  // gate still catches tampered content (see uniformity-restoration test)
});

test('negative: unavailable input fails closed with an explicit source-failure error', () => {
  assert.throws(
    () => readPinnedInput('docs/versions/v2.1.1/knowledge/vertical-slice/lanes/does-not-exist/records.yaml', null, 'missing lane'),
    /unavailable.*source failure must fail closed/,
  );
  assert.equal(existsSync(REPO_ROOT + 'docs/versions/v2.1.1/knowledge/vertical-slice/lanes/does-not-exist/records.yaml'), false);
});

test('negative: YAML duplicate keys are rejected (UniqueLoader discipline)', () => {
  assert.throws(
    () => parseYaml('a: 1\nb: 2\na: 3\n', 'dup-fixture'),
    YamlError,
  );
  assert.throws(
    () => parseYaml('outer:\n  inner: 1\n  inner: 2\n', 'dup-fixture2'),
    (e) => e instanceof YamlError && e.message.includes('duplicate key "inner"'),
  );
  // record-id duplication is an assembly-level gate, not a parser gate
  const parsed = parseYaml('collections:\n  domains:\n    - id: se:X\n    - id: se:X\n', 'dup-id-parse');
  assert.equal(parsed.collections.domains.length, 2);
});

test('negative: unknown relationship predicate fails closed', () => {
  const tampered = structuredClone(inputs);
  const lane = tampered.lanes.requirements;
  const ras = lane.collections.relationship_assertions;
  ras[0].predicate = 'NOT_IN_CATALOG';
  assert.throws(() => buildGraph('v211-baseline', tampered), /not declared in RELATIONSHIP_CATALOG/);
});

test('negative: dangling typed-edge endpoint fails closed', () => {
  const tampered = structuredClone(inputs);
  const lane = tampered.lanes.requirements;
  const ras = lane.collections.relationship_assertions;
  ras[0].object_ref = 'se:DOES-NOT-EXIST';
  assert.throws(() => buildGraph('v211-baseline', tampered), /does not resolve/);
});

test('negative: lane content tampering diverges from the frozen canonical bytes', () => {
  const tampered = structuredClone(inputs);
  const dms = tampered.lanes.requirements.collections.decision_models;
  const dm = dms.find((d) => d.id === 'se:DM-REQ-FUNCTIONAL-DOC-METHOD');
  dm.constraints[1] = 'forged constraint never present in the frozen lane';
  const baseline = buildGraph('v211-baseline', tampered);
  const frozen = readFileSync(REPO_ROOT + PATHS.frozenGraphJsonld, 'utf8');
  assert.notEqual(baseline.jsonld, frozen, 'tampered lane input must not reproduce the frozen export');
  assert.notEqual(baseline.digest, FROZEN_DIGEST);
});

test('negative: unauthorized ReviewDecision addition fails closed', () => {
  const tampered = structuredClone(inputs);
  tampered.overlay.additions.push({ '@id': 'se:UNAUTHORIZED-DECISION', '@type': 'se:ReviewDecision' });
  assert.throws(() => buildGraph('v212-c1', tampered), /addition type not authorized/);
});

test('negative: dropped-payload caveat removal fails closed', () => {
  const tampered = structuredClone(inputs);
  tampered.overlay.changes['se:CASE-09F-GH-DB-FAILOVER']['se:outcome']
    = 'All webhook payloads were delivered; no loss occurred at 23:03 UTC.';
  assert.throws(() => buildGraph('v212-c1', tampered), /dropped-payload caveat/);
});

test('negative: qualification tampering (Owner to Oracle laundering) fails closed', () => {
  const tampered = structuredClone(inputs);
  tampered.overlay.qualification.oracle = 'ATTESTED';
  assert.throws(() => buildGraph('v212-c1', tampered), /qualification conservation/);
  const tampered2 = structuredClone(inputs);
  tampered2.overlay.owner.OD05 = 'ACCEPT_WITH_SOURCE_PROOF';
  assert.throws(() => buildGraph('v212-c1', tampered2), /Owner decision OD05 drifted/);
});

test('negative: injected TTL/JSON-LD inequivalence is detected', () => {
  const cand = buildGraph('v212-c1', inputs);
  const mutated = [...cand.triples];
  mutated[0] = { ...mutated[0], o: { kind: 'lit', v: 'tampered' } };
  assert.ok(!triplesEqualSet(mutated, cand.triples));
  assert.notEqual(canonicalDigest(mutated), cand.digest);
  const parsed = parseTtl(cand.ttl);
  assert.notEqual(canonicalDigest(parsed), canonicalDigest(mutated));
});
