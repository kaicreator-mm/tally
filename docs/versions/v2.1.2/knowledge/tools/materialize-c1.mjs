import { readFileSync, writeFileSync } from 'node:fs';
import { createHash } from 'node:crypto';
import assert from 'node:assert/strict';
import { fileURLToPath } from 'node:url';
import { serializeJsonld } from './rdf-min.mjs';
export const original = fileURLToPath(new URL('../../../v2.1.1/knowledge/generated/graph.jsonld', import.meta.url));
export const overlay = fileURLToPath(new URL('../source/C1_OWNER_REVISION_R13.json', import.meta.url));
export const preview = fileURLToPath(new URL('../generated/c1-candidate.graph.jsonld', import.meta.url));
const expectedBlob = '0f6007315fd7352fe6fd218effcfecf672cb4961';
export function build(graph, patch) {
  assert.equal(patch.original.graphBlob, expectedBlob);
  assert.equal(patch.owner.OD05, 'NARROW_CLAIM');
  assert.equal(patch.owner.GH2018, 'ACCEPT_PATCH');
  assert.equal(patch.qualification.oracle, 'NOT_ATTESTED');
  assert.equal(graph['@graph'].length, 807);
  const originals = new Map(graph['@graph'].map(n => [n['@id'], n]));
  assert.equal(originals.size, 807);
  assert(originals.get('se:DM-REQ-FUNCTIONAL-DOC-METHOD')['se:constraints'].some(x => x.includes('uniformly across')));
  assert(originals.get('se:PAT-REQ-USER-STORY-SPECIFICATION')['se:solution'].includes('uniform'));
  assert(originals.get('se:LOC-09F-GH-REPLICATION')['se:locator'].includes('16:24'));
  assert(!originals.get('se:LOC-09F-GH-REPLICATION')['se:locator'].includes('23:03'));
  assert.deepEqual(Object.keys(patch.changes).sort(), [
    'se:GRAPH-V2.1.1-SEK08', 'se:DM-REQ-FUNCTIONAL-DOC-METHOD',
    'se:PAT-REQ-USER-STORY-SPECIFICATION', 'se:CASE-09F-GH-DB-FAILOVER'
  ].sort());
  assert.deepEqual(patch.additions.map(x => x['@id']).sort(), [
    'se:LOC-09F-GH-BACKLOG', 'se:EVD-09F-GH-BACKLOG'
  ].sort());
  const out = structuredClone(graph);
  for (const [id, fields] of Object.entries(patch.changes)) {
    const node = out['@graph'].find(n => n['@id'] === id);
    assert(node, 'missing original node ' + id);
    Object.assign(node, fields);
  }
  for (const n of patch.additions) {
    assert(!originals.has(n['@id']), 'duplicate addition ' + n['@id']);
    out['@graph'].push(structuredClone(n));
  }
  out['@graph'].sort((a,b) => a['@id'].localeCompare(b['@id'], 'en'));
  assert.equal(out['@graph'].length, 809);
  const ids = new Map(out['@graph'].map(n => [n['@id'], n]));
  assert.equal(ids.size, 809);
  const dm = ids.get('se:DM-REQ-FUNCTIONAL-DOC-METHOD');
  assert(dm['se:constraints'].some(x => x.includes('At least one')));
  assert(!dm['se:constraints'].some(x => x === 'The chosen method applies uniformly across the requirement set for consistency.'));
  assert(!ids.get('se:PAT-REQ-USER-STORY-SPECIFICATION')['se:solution'].includes('uniform across'));
  const c = ids.get('se:CASE-09F-GH-DB-FAILOVER');
  const e = ids.get('se:EVD-09F-GH-BACKLOG');
  const l = ids.get('se:LOC-09F-GH-BACKLOG');
  assert(c['se:evidence_refs'].some(x => x['@id'] === e['@id']));
  assert(c['se:outcome'].includes('200,000') && c['se:outcome'].includes('23:03'));
  assert.equal(e['se:source_locator_ref']['@id'], l['@id']);
  assert(l['se:locator'].includes('23:03'));
  assert.equal(e['se:lifecycle_status'], 'PENDING_REVIEW');
  assert.equal(e['se:extraction_status'], 'EXTRACTED_PENDING_REVIEW');
  assert(e['se:uncertainty'].includes('UNKNOWN'));
  assert.equal(ids.get('se:LOC-09F-GH-REPLICATION')['se:locator'], originals.get('se:LOC-09F-GH-REPLICATION')['se:locator']);
  assert.equal(out['@graph'].filter(n => n['@type'] === 'se:KnowledgeClaim').length, 106);
  assert.equal(out['@graph'].filter(n => n['@type'] === 'se:ReviewDecision').length, 0);
  return out;
}
export function materialize() {
  const bytes = readFileSync(original);
  const hash = createHash('sha1').update('blob ' + bytes.length + '\0').update(bytes).digest('hex');
  assert.equal(hash, expectedBlob, 'frozen original Git Blob changed');
  // canonical serialization (sorted keys and slot lists) — makes the overlay
  // product byte-identical with the canonical lane-integrated export
  return serializeJsonld(build(JSON.parse(bytes.toString('utf8')), JSON.parse(readFileSync(overlay, 'utf8')))['@graph']);
}
if (process.argv[1] && fileURLToPath(import.meta.url) === process.argv[1]) {
  if (process.argv.includes('--check')) {
    assert.equal(materialize(), readFileSync(preview, 'utf8'), 'preview must match authoritative source overlay');
    console.log('PASS: 809 nodes, 106 pending claims, 0 ReviewDecision; Oracle NOT_ATTESTED');
  } else if (process.argv.includes('--write')) {
    writeFileSync(preview, materialize());
    console.log('Updated v2.1.2 C1 candidate preview (not certified RDF graph)');
  } else {
    throw new Error('Provide --check or --write; no implicit mutation');
  }
}
