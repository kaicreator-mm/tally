// build-c1-canonical.test.mjs — native Node test suite for the Tally v2.1.2
// C1 canonical source-lane build (node --test, zero dependencies).
//
// Coverage:
//   positive: PyYAML parity of the Node YAML-subset parser over the whole
//     frozen corpus; overlay-free projection reproduces the frozen v2.1.1
//     exports byte-for-byte; candidate invariants (811/106/2/0, OD05 narrowing,
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
import { readFileSync, existsSync, copyFileSync, writeFileSync, mkdirSync, mkdtempSync, rmSync } from 'node:fs';
import { tmpdir } from 'node:os';
import { join, dirname } from 'node:path';
import { spawnSync } from 'node:child_process';
import { createHash } from 'node:crypto';
import {
  buildGraph, loadInputs, PATHS, REPO_ROOT, PINS, FROZEN_DIGEST,
  assertC1Candidate, assertV211Baseline, readPinnedInput, assertPinMatches,
  gitBlobSha1, parseTtl, triplesEqualSet, canonicalDigest, nodesToTriples,
} from './project-c1.mjs';
import { parseYaml, YamlError } from './yaml-min.mjs';
import { verify as verifyOwnerClaims } from './verify-c1-owner-claims-r100.mjs';
import { assertSourceOwnerDispositionEvolution, canonicalRecordDigest } from './source-owner-disposition.mjs';
import {
  build as materializeBuild,
  original as materializeOriginalPath,
  overlay as materializeOverlayPath,
} from './materialize-c1.mjs';
import {
  R137_IDS, R137_SHARED_SECTION_EVIDENCE, R137_SHARED_SECTION_DIRECT_CLAIMS,
  R140_REVIEWED_OUTCOME_EVIDENCE_DIGEST, R140_REVIEWED_CLAIM_SUPPORT_DIGEST,
  assertSelectedCqSourceBinding, readScbSchema,
} from './selected-cq-source-binding.mjs';

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
  assertC1Candidate(cand.nodes, inputs.overlay, cand.baselineNodes, inputs.sourceOwner, inputs.catalog);
  assert.equal(cand.nodes.length, 815);
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

test('R102 corrected CS2023 PDF locators reject old 174-176 range', () => {
  const cand=buildGraph('v212-c1',inputs),byId=new Map(cand.nodes.map(n=>[n['@id'],n]));
  for(const [id,page,printed] of [['se:LOC-CS2023-SE-REQ-KU',241,242],['se:LOC-CS2023-SE-VALIDATION-KU',245,246]]){
    const loc=byId.get(id);
    assert.equal(loc['se:record_version'],'0.1.1');
    assert.ok(loc['se:locator'].includes('verified PDF zero-based index '+page+', printed page '+printed));
    const fake=structuredClone(inputs);
    fake.overlay.changes[id]['se:locator']='wrong PDF page indices 174-176';
    // the R102 invariant or the earlier R137 SCB digest recomputation both fail closed
    assert.throws(()=>buildGraph('v212-c1',fake),/R102 PDF locator P2 drift|STALE SCB section locator digest/);
  }
});

test('R137 positive: selected VV CQ consumes BOTH the p246 section leg and the p248 LO2 outcome leg', () => {
  const cand=buildGraph('v212-c1',inputs);
  const byId=new Map(cand.nodes.map(n=>[n['@id'],n]));
  const scb=byId.get(R137_IDS.scb);
  assert.ok(scb,'SelectedCqSourceBinding missing');
  assert.equal(scb['@type'],'se:SelectedCqSourceBinding');
  assert.equal(scb['se:positive_cq'],'CS2023_VV_TEACHING');
  assert.equal(scb['se:section_anchor_type'],'SECTION');
  assert.equal(scb['se:section_pdf_zero_based_page'],'245');
  assert.equal(scb['se:section_printed_page'],'246');
  assert.equal(scb['se:outcome_anchor_type'],'PAGE');
  assert.equal(scb['se:outcome_pdf_zero_based_page'],'247');
  assert.equal(scb['se:outcome_printed_page'],'248');
  assert.equal(scb['se:binding_scope'],'SELECTED_CQ_RESEARCH_CANDIDATE_ONLY_NOT_FROZEN_R8_CELL');
  assert.equal(scb['se:professional_oracle_status'],'NOT_ATTESTED');
  assert.equal(scb['se:claim_lifecycle_effect'],'NONE_ORIGINAL_AND_CANDIDATE_CLAIM_REMAIN_PENDING');
  // distinct legs, distinct locators, distinct pages
  assert.notEqual(scb['se:section_locator_ref']['@id'],scb['se:outcome_locator_ref']['@id']);
  const outLoc=byId.get(R137_IDS.outcomeLocator);
  assert.equal(outLoc['se:locator_type'],'PAGE');
  assert.ok(outLoc['se:locator'].includes('zero-based index 247, printed page 248'));
  assert.ok(outLoc['se:locator'].includes('Distinguish between program validation and verification'));
  const outEvd=byId.get(R137_IDS.outcomeEvidence);
  assert.equal(outEvd['se:source_locator_ref']['@id'],R137_IDS.outcomeLocator);
  assert.equal(outEvd['se:lifecycle_status'],'PENDING_REVIEW');
  const support=byId.get(R137_IDS.claimSupport);
  assert.equal(support['se:subject_ref']['@id'],R137_IDS.claim);
  assert.equal(support['se:object_ref']['@id'],R137_IDS.outcomeEvidence);
  assert.equal(support['se:predicate']['@id'],'se:SUPPORTED_BY');
  // locked originals untouched: the claim keeps exactly its original edge
  const claim=byId.get(R137_IDS.claim);
  assert.deepEqual([claim['se:SUPPORTED_BY']['@id']],[R137_IDS.sectionEvidence]);
});

test('R137 positive: shared SECTION locator fan-out stays exactly 8 Evidence / 5 direct Claims', () => {
  const cand=buildGraph('v212-c1',inputs);
  const evidenceNodes=cand.nodes.filter(n=>n['@type']==='se:Evidence');
  const bySection=evidenceNodes.filter(n=>n['se:source_locator_ref']?.['@id']===R137_IDS.sectionLocator).map(n=>n['@id']).sort();
  assert.deepEqual(bySection,[...R137_SHARED_SECTION_EVIDENCE].sort());
  const byOutcome=evidenceNodes.filter(n=>n['se:source_locator_ref']?.['@id']===R137_IDS.outcomeLocator).map(n=>n['@id']);
  assert.deepEqual(byOutcome,[R137_IDS.outcomeEvidence]);
  const directClaims=cand.nodes.filter(n=>n['@type']==='se:KnowledgeClaim')
    .filter(n=>(Array.isArray(n['se:evidence_refs'])?n['se:evidence_refs']:[n['se:evidence_refs']])
      .some(r=>R137_SHARED_SECTION_EVIDENCE.includes(r?.['@id']))).map(n=>n['@id']).sort();
  assert.deepEqual(directClaims,[...R137_SHARED_SECTION_DIRECT_CLAIMS].sort());
});

test('R137 negative: orphan outcome evidence (locator consumed by nobody) fails closed', () => {
  const tampered=structuredClone(inputs);
  tampered.overlay.additions=tampered.overlay.additions.filter(a=>a['@id']!==R137_IDS.outcomeEvidence);
  assert.throws(()=>buildGraph('v212-c1',tampered),/orphan R137 record/);
});

test('R137 negative: single-page disguise (p246 double as the p248 outcome proof) fails closed', () => {
  const tampered=structuredClone(inputs);
  tampered.sourceOwner.document.claims[1].elementAttribution.selectedCqSourceBinding.outcomeLeg.locatorRef=R137_IDS.sectionLocator;
  assert.throws(()=>buildGraph('v212-c1',tampered),/distinct locators|distinct PDF pages|STALE SCB outcome locator/);
  const tampered2=structuredClone(inputs);
  tampered2.sourceOwner.document.claims[1].elementAttribution.selectedCqSourceBinding.outcomeLeg.pdfZeroBasedPage='245';
  assert.throws(()=>buildGraph('v212-c1',tampered2),/distinct PDF pages|outcome page anchor drift/);
});

test('R137 negative: shared-locator fan-out damage fails closed', () => {
  const tampered=structuredClone(inputs);
  const lane=tampered.lanes['testing-verification'];
  const tvd=lane.collections.evidence_records.find(e=>e.id==='se:EVD-TVV-0004');
  tvd.source_locator_ref='se:LOC-CS2023-SE-VALIDATION-LO2';
  assert.throws(()=>buildGraph('v212-c1',tampered),/fan-out changed/);
});

test('R137 negative: stale outcome locator digest fails closed', () => {
  const tampered=structuredClone(inputs);
  tampered.sourceOwner.document.claims[1].elementAttribution.selectedCqSourceBinding.outcomeLeg.digestSha256='0'.repeat(64);
  assert.throws(()=>buildGraph('v212-c1',tampered),/STALE SCB outcome locator digest/);
});

test('R137 negative: unauthorized overlay relationship predicate fails closed', () => {
  const tampered=structuredClone(inputs);
  const rel=tampered.overlay.additions.find(a=>a['@id']===R137_IDS.claimSupport);
  rel['se:predicate']={'@id':'se:NOT_IN_CATALOG'};
  assert.throws(()=>buildGraph('v212-c1',tampered),/not declared in RELATIONSHIP_CATALOG/);
});

// R140 P1: independently fixed, publisher-anchored semantic expectation.
test('R140 positive: outcome Evidence and support relation match independent fixed semantic digests', () => {
  const cand=buildGraph('v212-c1',inputs);
  const byId=new Map(cand.nodes.map(n=>[n['@id'],n]));
  assert.equal(sha256(canonicalJson(byId.get(R137_IDS.outcomeEvidence))),R140_REVIEWED_OUTCOME_EVIDENCE_DIGEST);
  assert.equal(sha256(canonicalJson(byId.get(R137_IDS.claimSupport))),R140_REVIEWED_CLAIM_SUPPORT_DIGEST);
  assert.equal(byId.get(R137_IDS.outcomeEvidence)['se:record_version'],'0.1.0');
  assert.equal(byId.get(R137_IDS.claimSupport)['se:record_version'],'0.1.0');
  assertSelectedCqSourceBinding(inputs.sourceOwner.document,readScbSchema(),cand.nodes,inputs.overlay);
});

test('R140 negative: Evidence statement inverted while both pages/IDs/status stay valid', () => {
  const tampered=structuredClone(inputs);
  const ev=tampered.overlay.additions.find(n=>n['@id']===R137_IDS.outcomeEvidence);
  ev['se:statement']='CS2023 demands verification and validation be identical; green tests automatically establish customer acceptance.';
  assert.equal(ev['se:source_locator_ref']['@id'],R137_IDS.outcomeLocator);
  assert.equal(ev['se:record_version'],'0.1.0');
  assert.throws(()=>buildGraph('v212-c1',tampered),/R140 LO2 Evidence semantic inversion/);
});

test('R140 negative: outcome Evidence metadata and support rationale each fail semantic drift', () => {
  for(const field of ['se:context','se:result','se:limitations','se:uncertainty']){
    const tampered=structuredClone(inputs);
    const ev=tampered.overlay.additions.find(n=>n['@id']===R137_IDS.outcomeEvidence);
    ev[field]='Fabricated publisher authority grants customer acceptance from passing tests.';
    assert.throws(()=>buildGraph('v212-c1',tampered),/R140 LO2 Evidence semantic\/digest drift/,field);
  }
  for(const field of ['se:context','se:rationale']){
    const tampered=structuredClone(inputs);
    const rel=tampered.overlay.additions.find(n=>n['@id']===R137_IDS.claimSupport);
    rel[field]='Publisher mandates identical verification and validation; green tests imply release acceptance.';
    assert.throws(()=>buildGraph('v212-c1',tampered),/R140 claim-support (rationale semantic inversion|semantic\/digest drift)/,field);
  }
});

test('R140 negative: coordinated Overlay + graph + Sidecar + simulated manifest repin cannot launder wrong Evidence meaning', () => {
  const baseline=buildGraph('v212-c1',inputs);
  const candidate=structuredClone(baseline.nodes);
  const patchedOverlay=structuredClone(inputs.overlay);
  const alteredText='CS2023 requires treating verification and validation as identical; all tests passed establishes customer acceptance.';
  candidate.find(n=>n['@id']===R137_IDS.outcomeEvidence)['se:statement']=alteredText;
  patchedOverlay.additions.find(n=>n['@id']===R137_IDS.outcomeEvidence)['se:statement']=alteredText;
  const authorSidecar=structuredClone(inputs.sourceOwner.document);
  const repinnedCandidateBytes=Buffer.from(JSON.stringify(candidate),'utf8');
  authorSidecar.sources.candidateGraphBlob=gitBlobSha1(repinnedCandidateBytes);
  const repinnedManifestSha256=createHash('sha256').update(repinnedCandidateBytes).digest('hex');
  assert.equal(authorSidecar.sources.candidateGraphBlob,gitBlobSha1(repinnedCandidateBytes));
  assert.equal(repinnedManifestSha256,createHash('sha256').update(repinnedCandidateBytes).digest('hex'));
  // The input pins now agree with each other, but the source-anchored
  // verification code DOES NOT derive its expected truth from these pins.
  assert.throws(()=>assertSelectedCqSourceBinding(authorSidecar,readScbSchema(),candidate,patchedOverlay),
    /R140 LO2 Evidence semantic inversion/);
});

test('R144: materialize reconstruction and canonical projection materialize the identical record form for every changed original and addition (fixed anchors + deep/canonical-digest parity + shaped ordering)', () => {
  const cand = buildGraph('v212-c1', inputs);
  const projected = new Map(cand.nodes.map((n) => [n['@id'], n]));
  const frozenGraph = JSON.parse(readFileSync(materializeOriginalPath, 'utf8'));
  const overlayDoc = JSON.parse(readFileSync(materializeOverlayPath, 'utf8'));
  const rebuilt = materializeBuild(frozenGraph, overlayDoc);
  const reconstructed = new Map(rebuilt['@graph'].map((n) => [n['@id'], n]));
  // R140 fixed semantic anchors hold on BOTH construction paths.
  for (const [id, digest] of [
    [R137_IDS.outcomeEvidence, R140_REVIEWED_OUTCOME_EVIDENCE_DIGEST],
    [R137_IDS.claimSupport, R140_REVIEWED_CLAIM_SUPPORT_DIGEST],
  ]) {
    assert.equal(canonicalRecordDigest(projected.get(id)), digest,
      'R144 projection-path fixed-anchor drift for ' + id);
    assert.equal(canonicalRecordDigest(reconstructed.get(id)), digest,
      'R144 reconstruction-path fixed-anchor drift for ' + id);
  }
  // The co-shape class is closed mechanically: EVERY overlay-changed original
  // (including the renamed graph root, resolved through its new @id) and
  // EVERY addition must be deep-equal and canonical-digest-equal across the
  // projection and reconstruction construction paths.
  for (const [id, fields] of Object.entries(overlayDoc.changes)) {
    const targetId = fields['@id'] ?? id;
    const a = projected.get(targetId);
    const b = reconstructed.get(targetId);
    assert.ok(a && b, 'R144 both paths must materialize changed original ' + targetId);
    assert.deepEqual(b, a, 'R144 changed-original deep parity failed for ' + targetId);
    assert.equal(canonicalRecordDigest(b), canonicalRecordDigest(a),
      'R144 changed-original canonical-digest parity failed for ' + targetId);
  }
  for (const n of overlayDoc.additions) {
    const a = projected.get(n['@id']);
    const b = reconstructed.get(n['@id']);
    assert.ok(a && b, 'R144 both paths must materialize addition ' + n['@id']);
    assert.deepEqual(b, a, 'R144 addition deep parity failed for ' + n['@id']);
    assert.equal(canonicalRecordDigest(b), canonicalRecordDigest(a),
      'R144 addition canonical-digest parity failed for ' + n['@id']);
    // both paths assemble additions with @id/@type pinned first
    assert.deepEqual(Object.keys(a).slice(0, 2), ['@id', '@type'],
      'R144 projection addition field order drifted for ' + n['@id']);
    assert.deepEqual(Object.keys(b).slice(0, 2), ['@id', '@type'],
      'R144 reconstruction addition field order drifted for ' + n['@id']);
  }
  // Shaped value ordering survives verbatim on both paths: string arrays sort
  // lexicographically (field order) and @id ref arrays sort by id (reference
  // order) — the two live co-shape falsifiers measured by the R142 review.
  assert.ok(projected.get('se:DM-REQ-FUNCTIONAL-DOC-METHOD')['se:options']
    .every((x, i, arr) => i === 0 || arr[i - 1] <= arr[i]),
    'R144 shaped se:options must stay lexicographically sorted');
  assert.deepEqual(
    reconstructed.get('se:DM-REQ-FUNCTIONAL-DOC-METHOD')['se:options'],
    projected.get('se:DM-REQ-FUNCTIONAL-DOC-METHOD')['se:options']);
  const refIds = (rec) => rec['se:evidence_refs'].map((x) => x['@id']);
  assert.deepEqual(refIds(projected.get('se:CASE-09F-GH-DB-FAILOVER')),
    ['se:EVD-09F-GH-BACKLOG', 'se:EVD-09F-GH-RECOVERY', 'se:EVD-09F-GH-REPLICATION'],
    'R144 shaped evidence_refs must stay @id-sorted');
  assert.deepEqual(refIds(reconstructed.get('se:CASE-09F-GH-DB-FAILOVER')),
    refIds(projected.get('se:CASE-09F-GH-DB-FAILOVER')));
});

test('R144 negative: an overlay change with an emptied array field drops the key identically on both construction paths (undefined→delete parity)', () => {
  const tampered = structuredClone(inputs);
  tampered.overlay.changes['se:DM-REQ-FUNCTIONAL-DOC-METHOD']['se:options'] = [];
  const cand = buildGraph('v212-c1', tampered);
  const projected = new Map(cand.nodes.map((n) => [n['@id'], n]));
  const frozenGraph = JSON.parse(readFileSync(materializeOriginalPath, 'utf8'));
  const tamperedOverlay = structuredClone(JSON.parse(readFileSync(materializeOverlayPath, 'utf8')));
  tamperedOverlay.changes['se:DM-REQ-FUNCTIONAL-DOC-METHOD']['se:options'] = [];
  const rebuilt = materializeBuild(frozenGraph, tamperedOverlay);
  const reconstructed = new Map(rebuilt['@graph'].map((n) => [n['@id'], n]));
  for (const [label, rec] of [
    ['projection', projected.get('se:DM-REQ-FUNCTIONAL-DOC-METHOD')],
    ['reconstruction', reconstructed.get('se:DM-REQ-FUNCTIONAL-DOC-METHOD')],
  ]) {
    assert.ok(rec && !('se:options' in rec),
      'R144 ' + label + ' path must delete an emptied change key instead of assigning []');
  }
  assert.deepEqual(reconstructed.get('se:DM-REQ-FUNCTIONAL-DOC-METHOD'),
    projected.get('se:DM-REQ-FUNCTIONAL-DOC-METHOD'),
    'R144 emptied-key records must stay identical across both paths');
  assert.equal(
    canonicalRecordDigest(reconstructed.get('se:DM-REQ-FUNCTIONAL-DOC-METHOD')),
    canonicalRecordDigest(projected.get('se:DM-REQ-FUNCTIONAL-DOC-METHOD')),
    'R144 emptied-key canonical-digest parity failed');
});

test('R144 negative: reconstruction path enforces the Owner overlay qualification contract before materializing', () => {
  const frozenGraph = JSON.parse(readFileSync(materializeOriginalPath, 'utf8'));
  const overlayDoc = JSON.parse(readFileSync(materializeOverlayPath, 'utf8'));
  const laundered = structuredClone(overlayDoc);
  laundered.qualification.oracle = 'ATTESTED';
  assert.throws(() => materializeBuild(frozenGraph, laundered), /qualification conservation/);
  const drifted = structuredClone(overlayDoc);
  drifted.owner.OD05 = 'ACCEPT_WITH_SOURCE_PROOF';
  assert.throws(() => materializeBuild(frozenGraph, drifted), /Owner decision OD05 drifted/);
  const unpreserved = structuredClone(overlayDoc);
  unpreserved.originalsPreserved = false;
  assert.throws(() => materializeBuild(frozenGraph, unpreserved), /originalsPreserved/);
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

test('R107 SourceOwnerDisposition is canonical, exact-bound, and separate from Claim review', () => {
  const cand=buildGraph('v212-c1',inputs);
  const dispositions=cand.nodes.filter(n=>n['@type']==='se:SourceOwnerDisposition');
  assert.deepEqual(dispositions.map(n=>n['@id']).sort(),['se:SOD-CS2023-REQ-R1','se:SOD-CS2023-VV-R1']);
  for(const disposition of dispositions){
    assert.equal(disposition['se:disposition'],'ACCEPT_SCOPED_FOR_VERSIONED_CANDIDATE');
    assert.equal(disposition['se:lifecycle_status'],'ACTIVE');
    assert.equal(disposition['se:professional_oracle_status'],'NOT_ATTESTED');
    assert.equal(disposition['se:publication_permission'],'NOT_GRANTED_BY_THIS_OWNER_DECISION');
    assert.equal(disposition['se:claim_lifecycle_effect'],'NONE_ORIGINAL_AND_CANDIDATE_CLAIM_REMAIN_PENDING');
    assert.ok(disposition['se:unknown'].length>0);
    const claim=cand.nodes.find(n=>n['@id']===disposition['se:claim_ref']['@id']);
    assert.equal(claim['se:review_status'],'PENDING');
    assert.equal(claim['se:lifecycle_status'],'PENDING_REVIEW');
  }
});

test('SourceOwnerDisposition evolution permits non-ACTIVE history and rejects duplicate ACTIVE lineage', () => {
  const cand=buildGraph('v212-c1',inputs);
  const current=structuredClone(cand.nodes.find(n=>n['@id']==='se:SOD-CS2023-REQ-R1'));
  const previous=structuredClone(current);
  previous['@id']='se:SOD-CS2023-REQ-R0';
  previous['se:record_version']='0.0.9';
  previous['se:lifecycle_status']='SUPERSEDED';
  current['se:supersedes_ref']={'@id':previous['@id']};
  assert.doesNotThrow(()=>assertSourceOwnerDispositionEvolution([previous,current]));
  const duplicateActive=structuredClone(previous);
  duplicateActive['se:lifecycle_status']='ACTIVE';
  assert.throws(()=>assertSourceOwnerDispositionEvolution([duplicateActive,current]),/cannot remain ACTIVE|multiple ACTIVE/);
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

test('negative: ACTIVE SourceOwnerDisposition stale exact binding fails closed', () => {
  const tampered=structuredClone(inputs);
  tampered.sourceOwner.document.claims[0].recordBindings.candidateLocator.digestSha256='0'.repeat(64);
  assert.throws(()=>buildGraph('v212-c1',tampered),/STALE candidate_locator digest/);
});

test('negative: source+projection co-drift cannot bypass independent Owner expectations', () => {
  const tampered=structuredClone(inputs);
  tampered.sourceOwner.document.claims[0].scope='universal professional mandate with unrestricted rights';
  const cand=buildGraph('v212-c1',tampered);
  const frozen=JSON.parse(readFileSync(REPO_ROOT+PATHS.frozenGraphJsonld,'utf8'));
  assert.throws(
    ()=>verifyOwnerClaims(tampered.sourceOwner.document,frozen,JSON.parse(cand.jsonld),tampered.sourceOwner.schema),
    assert.AssertionError,
  );
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

test('regression: --write repairs stale or missing Owner preview in an isolated native checkout', () => {
  // Do not mutate the real repository's immutable source or committed exports.
  const tempRoot = mkdtempSync(join(tmpdir(), 'tally-c1-canonical-write-'));
  const toolPrefix = 'docs/versions/v2.1.2/knowledge/tools/';
  const sourcePaths = [
    ...Object.keys(PINS.lanes).map(PATHS.lanes),
    PATHS.registry, PATHS.catalog,
    PATHS.frozenGraphJsonld, PATHS.frozenGraphTtl, PATHS.overlay,
    PATHS.sourceOwnerDispositions, PATHS.sourceOwnerSchema,
    'docs/versions/v2.1.2/knowledge/model/selected-cq-source-binding.schema.json',
    PATHS.outGraphJsonld, PATHS.outGraphTtl, PATHS.outManifest, PATHS.outPreview,
    ...['build-c1-canonical.mjs', 'project-c1.mjs', 'materialize-c1.mjs',
         'source-owner-disposition.mjs', 'selected-cq-source-binding.mjs',
         'yaml-min.mjs', 'rdf-min.mjs'].map(name => toolPrefix + name),
  ];
  const run = mode => spawnSync(process.execPath,
    [join(tempRoot, toolPrefix + 'build-c1-canonical.mjs'), mode],
    {cwd: tempRoot, encoding: 'utf8', timeout: 120000});
  try {
    for (const rel of sourcePaths) {
      const destination = join(tempRoot, rel);
      mkdirSync(dirname(destination), {recursive: true});
      copyFileSync(REPO_ROOT + rel, destination);
    }
    const previewPath = join(tempRoot, PATHS.outPreview);
    const candidatePath = join(tempRoot, PATHS.outGraphJsonld);
    const frozenPath = join(tempRoot, PATHS.frozenGraphJsonld);
    const frozenBefore = readFileSync(frozenPath);
    const canonicalBefore = readFileSync(candidatePath);

    writeFileSync(previewPath, '{"@graph":[]}\n');
    const red = run('--check');
    assert.notEqual(red.status, 0, 'stale preview must fail --check');
    const greenWrite = run('--write');
    assert.equal(greenWrite.status, 0, greenWrite.stderr || greenWrite.stdout);
    assert.deepEqual(readFileSync(previewPath), readFileSync(candidatePath),
      '--write must regenerate a byte-identical preview');
    assert.deepEqual(readFileSync(candidatePath), canonicalBefore);
    const greenCheck = run('--check');
    assert.equal(greenCheck.status, 0, greenCheck.stderr || greenCheck.stdout);

    rmSync(previewPath);
    const missingRed = run('--check');
    assert.notEqual(missingRed.status, 0, 'missing preview must fail --check');
    const missingGreen = run('--write');
    assert.equal(missingGreen.status, 0, missingGreen.stderr || missingGreen.stdout);
    assert.deepEqual(readFileSync(previewPath), readFileSync(candidatePath));
    const finalCheck = run('--check');
    assert.equal(finalCheck.status, 0, finalCheck.stderr || finalCheck.stdout);
    assert.deepEqual(readFileSync(frozenPath), frozenBefore,
      'frozen v2.1.1 source must remain unchanged');
  } finally {
    rmSync(tempRoot, {recursive: true, force: true});
  }
});
