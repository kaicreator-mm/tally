import test from 'node:test';
import assert from 'node:assert/strict';
import { createHash } from 'node:crypto';
import { describeRisks, recordPieceApplications } from './route-guide.mjs';

const sha = value => createHash('sha256').update(value).digest('hex');
const seal = payload => ({ ...payload, receiptSha256: sha(JSON.stringify(payload)) });
function fixture() {
  const delivered = ['precision', 'abstraction', 'effects'].map(id => {
    const text = 'synthetic calibration ' + id;
    return { id, sha256: sha(text), bytes: Buffer.byteLength(text), text };
  });
  const selected = delivered.map(({ text, ...metadata }) => metadata);
  const receipt = seal({ decisionId: 'synthetic', status: 'DELIVERED', selected, delivered });
  const report = { receiptSha256: receipt.receiptSha256, pieces: [
    { id: 'precision', read: 'YES', outcome: 'ALREADY_ALIGNED', reason: 'existing test', before: 'boundary test', after: 'boundary test' },
    { id: 'abstraction', read: 'YES', outcome: 'ALREADY_ALIGNED', reason: 'private constant', before: 'private', after: 'private' },
    { id: 'effects', read: 'YES', outcome: 'REJECTED', reason: 'no persistent or external effect' }
  ] };
  return { receipt, report };
}
test('guide exposes eight scoped labels without loading excerpt text', () => {
  const guide = describeRisks();
  assert.deepEqual(guide.map(x => x.id), ['boundary','alias','abstraction','precision','oracle','maintenance','effects','cost']);
  assert.ok(guide.every(x => x.when && x.notWhen && !('text' in x)));
  assert.match(guide.find(x => x.id === 'precision').notWhen, /byte versus character/);
  assert.match(guide.find(x => x.id === 'effects').notWhen, /buffer reads/);
  assert.match(guide.find(x => x.id === 'alias').notWhen, /string aliases/);
});
test('mixed per-piece outcomes remain separate and self-reported', () => {
  const { receipt, report } = fixture();
  const result = recordPieceApplications(receipt, report);
  assert.deepEqual(result.pieces.map(p => p.outcome), ['ALREADY_ALIGNED','ALREADY_ALIGNED','REJECTED']);
  assert.equal(result.qualification, 'SELF_REPORTED_NOT_VERIFIED');
  assert.equal(result.qualityBenefit, 'NOT_ESTABLISHED');
});
test('missing, duplicate and foreign piece reports reject', () => {
  for (const change of [p => p.pop(), p => { p[1] = p[0]; }, p => { p[1].id = 'cost'; }]) {
    const { receipt, report } = fixture(); change(report.pieces);
    assert.throws(() => recordPieceApplications(receipt, report), /PIECE_REPORT/);
  }
});
test('unknown reading or rejection can be recorded without inventing a decision', () => {
  const { receipt, report } = fixture();
  report.pieces[2] = { id: 'effects', read: 'UNKNOWN', outcome: 'UNKNOWN', reason: 'not observed' };
  assert.equal(recordPieceApplications(receipt, report).pieces[2].before, null);
});
test('changed and unchanged decisions require consistent reported evidence', () => {
  const { receipt, report } = fixture();
  report.pieces[0].outcome = 'APPLIED';
  assert.throws(() => recordPieceApplications(receipt, report), /DECISION_OUTCOME_MISMATCH/);
  report.pieces[0].after = 'new test';
  assert.equal(recordPieceApplications(receipt, report).pieces[0].outcome, 'APPLIED');
  report.pieces[0].read = 'NO';
  assert.throws(() => recordPieceApplications(receipt, report), /MISSING_DECISION_EVIDENCE/);
});
test('receipt identity mismatch and modified text reject', () => {
  const { receipt, report } = fixture();
  assert.throws(() => recordPieceApplications(receipt, { ...report, receiptSha256: 'bad' }), /DELIVERY_MISMATCH/);
  receipt.delivered[0].text += '!';
  assert.throws(() => recordPieceApplications(receipt, report), /DELIVERY_MISMATCH/);
  const { receiptSha256, ...payload } = receipt;
  const resealed = seal(payload);
  assert.throws(() => recordPieceApplications(resealed, { ...report, receiptSha256: resealed.receiptSha256 }), /INVALID_DELIVERED_PIECE/);
});
test('receipt metadata inconsistency is rejected even after resealing', () => {
  const { receipt, report } = fixture();
  const { receiptSha256, ...payload } = receipt;
  payload.selected[0].bytes++;
  const resealed = seal(payload);
  assert.throws(() => recordPieceApplications(resealed, { ...report, receiptSha256: resealed.receiptSha256 }), /INVALID_DELIVERED_PIECE/);
});
test('NONE and ASK do not create an application receipt', () => {
  for (const status of ['NONE','ASK','UNKNOWN']) assert.throws(() => recordPieceApplications({ status }, {}), /INVALID_DELIVERY/);
});
