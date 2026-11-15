import { readFileSync, writeFileSync } from 'node:fs';
import { createHash } from 'node:crypto';
import assert from 'node:assert/strict';
import { fileURLToPath } from 'node:url';
import { serializeJsonld } from './rdf-min.mjs';
import {
  readSourceOwnerInputs, materializeSourceOwnerDispositions,
  assertC1SourceOwnerDispositions,
} from './source-owner-disposition.mjs';
import {
  readScbSchema, materializeSelectedCqSourceBinding,
  assertSelectedCqSourceBinding,
} from './selected-cq-source-binding.mjs';
import { shapeOverlayValue, checkOverlayContract } from './project-c1.mjs';
export const original = fileURLToPath(new URL('../../../v2.1.1/knowledge/generated/graph.jsonld', import.meta.url));
export const overlay = fileURLToPath(new URL('../source/C1_OWNER_REVISION_R13.json', import.meta.url));
export const preview = fileURLToPath(new URL('../generated/c1-candidate.graph.jsonld', import.meta.url));
const expectedBlob = '0f6007315fd7352fe6fd218effcfecf672cb4961';
export function build(graph, patch, sourceOwner = readSourceOwnerInputs()) {
  // Reconstruction enforces the same Owner overlay qualification contract as
  // the canonical projection (project-c1 applyOverlay runs this on its path):
  // schema pin, OD05/GH2018, frozen-graph blob pin, qualification
  // conservation and originalsPreserved must all hold before anything is
  // materialized, so a reconstruction-only consumer cannot skip the
  // fail-closed Owner gates.
  checkOverlayContract(patch);
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
    'se:PAT-REQ-USER-STORY-SPECIFICATION', 'se:CASE-09F-GH-DB-FAILOVER',
    'se:LOC-CS2023-SE-REQ-KU', 'se:LOC-CS2023-SE-VALIDATION-KU'
  ].sort());
  assert.deepEqual(patch.additions.map(x => x['@id']).sort(), [
    'se:EVD-09F-GH-BACKLOG', 'se:EVD-TVV-0016', 'se:LOC-09F-GH-BACKLOG',
    'se:LOC-CS2023-SE-VALIDATION-LO2', 'se:REL-TVV-0024'
  ].sort());
  const out = structuredClone(graph);
  for (const [id, fields] of Object.entries(patch.changes)) {
    const node = out['@graph'].find(n => n['@id'] === id);
    assert(node, 'missing original node ' + id);
    // Consume change fields through the same value-shaping rule as the
    // canonical projection (project-c1 applyOverlay change loop: shaped
    // values, undefined→delete for emptied arrays, @id rename last), so both
    // construction paths materialize the identical record form for every
    // changed original — not only the R140 pinned additions.
    const finalId = fields['@id'] ?? node['@id'];
    for (const [k, v] of Object.entries(fields)) {
      if (k === '@id') continue;
      const shaped = shapeOverlayValue(k, v);
      if (shaped === undefined) delete node[k];
      else node[k] = shaped;
    }
    if (finalId !== node['@id']) node['@id'] = finalId;
  }
  for (const n of patch.additions) {
    assert(!originals.has(n['@id']), 'duplicate addition ' + n['@id']);
    // Consume additions through the same value-shaping rule as the canonical
    // projection (project-c1 shapeOverlayValue), so both construction paths
    // materialize the identical record form for the R140 fixed-anchor digests.
    const node = { '@id': n['@id'], '@type': n['@type'] };
    for (const [k, v] of Object.entries(n)) {
      if (k === '@id' || k === '@type') continue;
      const shaped = shapeOverlayValue(k, v);
      if (shaped !== undefined) node[k] = shaped;
    }
    out['@graph'].push(node);
  }
  for (const n of materializeSourceOwnerDispositions(sourceOwner.document, sourceOwner.schema)) {
    assert(!originals.has(n['@id']), 'duplicate disposition ' + n['@id']);
    out['@graph'].push(n);
  }
  const scbNode = materializeSelectedCqSourceBinding(sourceOwner.document, readScbSchema(), out['@graph'], patch);
  out['@graph'].push(scbNode);
  out['@graph'].sort((a,b) => a['@id'].localeCompare(b['@id'], 'en'));
  assert.equal(out['@graph'].length, 815);
  const ids = new Map(out['@graph'].map(n => [n['@id'], n]));
  assert.equal(ids.size, 815);
  const dm = ids.get('se:DM-REQ-FUNCTIONAL-DOC-METHOD');
  assert(dm['se:constraints'].some(x => x.includes('At least one')));
  assert(!dm['se:constraints'].some(x => x === 'The chosen method applies uniformly across the requirement set for consistency.'));
  assert(!ids.get('se:PAT-REQ-USER-STORY-SPECIFICATION')['se:solution'].includes('uniform across'));
  for(const [id,page,printed] of [['se:LOC-CS2023-SE-REQ-KU',241,242],['se:LOC-CS2023-SE-VALIDATION-KU',245,246]]){
    const loc=ids.get(id);
    assert.equal(loc['se:record_version'],'0.1.1');
    assert(loc['se:locator'].includes('verified PDF zero-based index '+page+', printed page '+printed));
  }
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
  // R137: the selected-CQ two-leg source binding must be consumed in this
  // reconstruction path as well (byte-convergence R4 additionally proves it)
  assertSelectedCqSourceBinding(sourceOwner.document, readScbSchema(), out['@graph'], patch);
  assertC1SourceOwnerDispositions(sourceOwner.document, sourceOwner.schema, graph['@graph'], out['@graph']);
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
    console.log('PASS: 815 nodes (R137 two-leg selected-CQ binding consumed), 106 pending claims, 2 SourceOwnerDispositions, 1 SelectedCqSourceBinding, 0 ReviewDecision; Oracle NOT_ATTESTED');
  } else if (process.argv.includes('--write')) {
    writeFileSync(preview, materialize());
    console.log('Updated v2.1.2 C1 candidate preview (not certified RDF graph)');
  } else {
    throw new Error('Provide --check or --write; no implicit mutation');
  }
}
