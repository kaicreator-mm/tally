import test from 'node:test';
import assert from 'node:assert/strict';
import { readFileSync } from 'node:fs';
import { createHash } from 'node:crypto';
import { KNOWLEDGE_SHA, verifyCatalog, select, deliver, recordApplication, consumeBatch } from './selector.mjs';
const catalog = JSON.parse(readFileSync(new URL('./catalog.json', import.meta.url), 'utf8'));
const risk = (id, applicability = 'YES') => ({ id, applicability, evidence: 'explicit caller-assessed source observation' });
const ctx = (overrides = {}) => ({ decisionId: 'd1', issue: '#561 synthetic calibration', sourceSha: 'a'.repeat(40), stage: 'L2',
  question: 'choose boundary', contract: 'CONFIRMED', contractEvidence: 'synthetic stated contract', risks: [], ...overrides });
const run = overrides => select(ctx(overrides), catalog);
test('all eight real excerpts have pinned identity and exact UTF-8 digest', () => {
  assert.equal(catalog.sourceSha, KNOWLEDGE_SHA); assert.equal(verifyCatalog(catalog).size, 8);
  for (const e of catalog.entries) assert.equal(createHash('sha256').update(e.text).digest('hex'), e.sha256);
});
test('risk routes select only their own excerpt at valid stage', () => {
  for (const [id, stage] of [['boundary','L2'],['alias','L2'],['abstraction','L3'],['precision','L3'],['oracle','QA'],['maintenance','MAINTENANCE'],['effects','L2'],['cost','L1']]) {
    const p = run({stage, risks:[risk(id)]}); assert.equal(p.status,'SELECTED'); assert.deepEqual(p.selected.map(x=>x.id),[id]); assert.equal('text' in p.selected[0],false);
  }
});
test('no risk or explicitly inapplicable risk returns NONE', () => { assert.equal(run({}).status,'NONE'); assert.equal(run({risks:[risk('alias','NO')]}).status,'NONE'); });
test('same question text, different facts select different knowledge', () => {
  assert.equal(run({question:'shared state',risks:[risk('alias')]}).selected[0].id,'alias');
  assert.equal(run({question:'shared state',risks:[risk('boundary')]}).selected[0].id,'boundary');
});
test('missing or conflicting authority blocks delivery', () => { for(const contract of ['MISSING','CONFLICT']) {const p=run({contract,risks:[risk('alias')]});assert.equal(p.status,'ASK');assert.deepEqual(p.selected,[]);} });
test('unknown or unsupported applicability cannot silently disappear', () => {
  for(const r of [risk('alias','UNKNOWN'),risk('unknown'),risk('oracle')]) {const p=run({risks:[risk('boundary'),r]});assert.equal(p.status,'UNKNOWN');assert.equal(p.selected.length,0);assert.equal(p.rejected.length,2);}
});
test('stale catalog SHA and altered excerpt reject', () => {
  assert.throws(()=>select(ctx(),{...catalog,sourceSha:'b'.repeat(40)}),/STALE_CATALOG/);
  const altered=structuredClone(catalog);altered.entries[0].text+='!';assert.throws(()=>select(ctx(),altered),/INVALID_EXCERPT/);
  altered.entries[0].sha256=createHash('sha256').update(altered.entries[0].text).digest('hex');assert.throws(()=>select(ctx(),altered),/CATALOG_IDENTITY_MISMATCH/);
});
test('budget exact UTF-8 boundary, soft expansion, and piece cap', () => {
  const n=Buffer.byteLength(catalog.entries.find(x=>x.id==='alias').text);
  assert.equal(run({risks:[risk('alias')],maxBytes:n-1}).reason,'BUDGET_INSUFFICIENT');
  assert.equal(run({risks:[risk('alias')],maxBytes:n}).usedBytes,n);
  assert.equal(run({risks:[risk('alias'),risk('boundary')],maxPieces:1}).reason,'BUDGET_INSUFFICIENT');
  assert.equal(run({risks:[risk('alias'),risk('boundary')],maxPieces:2}).selected.length,2);
});
test('unrelated declined knowledge is not delivered', () => {const p=deliver(ctx({risks:[risk('alias'),risk('cost','NO')]}),catalog);assert.equal(p.delivered.length,1);assert.equal(p.delivered[0].id,'alias');assert.equal(p.rejected[0].reason,'NOT_APPLICABLE');});
test('delivery has exact bytes, deterministic receipt, no claim of reading', () => {
  const c=ctx({risks:[risk('precision')]});const p=deliver(c,catalog);assert.deepEqual(p,deliver(c,catalog));assert.equal(p.deliveryMeaning,'MATERIALIZED_NOT_READ');assert.equal(p.delivered[0].text,catalog.entries.find(x=>x.id==='precision').text);
});
test('multiple decisions stay isolated; duplicate IDs reject', () => {
  const a=ctx({risks:[risk('alias')]}),b=ctx({decisionId:'d2',stage:'QA',risks:[risk('oracle')]});const p=consumeBatch([a,b],catalog);assert.deepEqual(p.map(x=>x.delivered[0].id),['alias','oracle']);assert.throws(()=>consumeBatch([a,a],catalog),/DUPLICATE_DECISION/);
});
test('correct existing decision may remain unchanged, not verified benefit', () => {
  const c=ctx({risks:[risk('alias')]});const p=deliver(c,catalog);const r=recordApplication(c,catalog,{receiptSha256:p.receiptSha256,outcome:'ALREADY_ALIGNED',before:'snapshot',after:'snapshot',reason:'contract already satisfied',probe:'mutate caller object'});assert.equal(r.qualification,'SELF_REPORTED_NOT_VERIFIED');assert.equal(r.qualityBenefit,'NOT_ESTABLISHED');
});
test('application requires correct receipt and real decision change for APPLIED', () => {
  const c=ctx({risks:[risk('alias')]});const p=deliver(c,catalog);const r={receiptSha256:p.receiptSha256,outcome:'APPLIED',before:'share',after:'snapshot',reason:'stated contract',probe:'mutate caller object'};assert.equal(recordApplication(c,catalog,r).outcome,'APPLIED');assert.throws(()=>recordApplication(c,catalog,{...r,receiptSha256:'bad'}),/DELIVERY_MISMATCH/);assert.throws(()=>recordApplication(c,catalog,{...r,after:r.before}),/NO_DECISION_CHANGE/);
});
test('invalid input, duplicate risks, prototype-like IDs do not bypass selection', () => {
  assert.throws(()=>run({sourceSha:'not-a-sha'}),/INVALID_CONTEXT/);assert.throws(()=>run({maxBytes:-1}),/INVALID_BUDGET/);assert.throws(()=>run({risks:[risk('alias'),risk('alias')]}),/INVALID_RISK/);assert.equal(run({risks:[risk('__proto__')]}).status,'UNKNOWN');
});
