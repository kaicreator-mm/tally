import { test } from 'node:test';
import assert from 'node:assert/strict';
import { readFileSync } from 'node:fs';
import { build, materialize, original, overlay, preview } from './materialize-c1.mjs';
const get = (g,id) => g['@graph'].find(n=>n['@id']===id);
test('byte-pinned original reconstructs exact candidate without touching history', () => {
  assert.equal(materialize(), readFileSync(preview,'utf8'));
  assert.equal(JSON.parse(materialize())['@graph'].length, 809);
});
test('OD05 accepts mixed methods and rejects universal uniformity claim', () => {
  const g=JSON.parse(materialize()), dm=get(g,'se:DM-REQ-FUNCTIONAL-DOC-METHOD');
  assert(dm['se:procedure'].includes('different methods for different requirements'));
  assert(!dm['se:constraints'].some(x=>x==='The chosen method applies uniformly across the requirement set for consistency.'));
  const p=JSON.parse(readFileSync(overlay,'utf8'));
  assert(p.boundaries.OD05.negative.includes('Reject universal'));
  assert(p.boundaries.OD05.unknown.includes('No method superiority'));
});
test('GH 16:24 not backlog done, 23:03 not zero-loss; dropped evidence persists', () => {
  const g=JSON.parse(materialize()), originalLoc=get(g,'se:LOC-09F-GH-REPLICATION'), later=get(g,'se:LOC-09F-GH-BACKLOG'), e=get(g,'se:EVD-09F-GH-BACKLOG');
  assert(originalLoc['se:locator'].includes('16:24') && !originalLoc['se:locator'].includes('23:03'));
  assert(later['se:locator'].includes('23:03'));
  assert(e['se:statement'].includes('200,000'));
  assert.equal(e['se:source_locator_ref']['@id'], later['@id']);
  assert.equal(e['se:lifecycle_status'], 'PENDING_REVIEW');
});
test('negative original drift and unexpected patch additions fail closed', () => {
  const base=JSON.parse(readFileSync(original,'utf8')), patch=JSON.parse(readFileSync(overlay,'utf8'));
  get(base,'se:DM-REQ-FUNCTIONAL-DOC-METHOD')['se:constraints']=['forged'];
  assert.throws(()=>build(base,patch));
  const fresh=JSON.parse(readFileSync(original,'utf8'));
  patch.additions.push({'@id':'se:UNAUTHORIZED','@type':'se:ReviewDecision'});
  assert.throws(()=>build(fresh,patch));
});
test('negative missing dropped-payload caveat is rejected', () => {
  const base=JSON.parse(readFileSync(original,'utf8')), patch=JSON.parse(readFileSync(overlay,'utf8'));
  patch.changes['se:CASE-09F-GH-DB-FAILOVER']['se:outcome']='All events delivered, no loss';
  assert.throws(()=>build(base,patch));
});
test('no Owner-to-Oracle qualification laundering', () => {
  const p=JSON.parse(readFileSync(overlay,'utf8')), g=JSON.parse(materialize());
  assert.equal(p.qualification.sourceOwner,'2/2 CONFIRMED');
  assert.equal(p.qualification.oracle,'NOT_ATTESTED');
  assert.equal(p.qualification.sap,'0/6');
  assert.equal(p.qualification.dkeR8,'0/53');
  assert.equal(g['@graph'].filter(n=>n['@type']==='se:ReviewDecision').length,0);
});
