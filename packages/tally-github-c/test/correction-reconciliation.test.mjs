import test from 'node:test';
import assert from 'node:assert/strict';
import {readFileSync} from 'node:fs';
import {reconcileCorrectionNotices} from '../src/correction-reconciliation.mjs';
import {STATES} from '../src/report-projection.mjs';

const summary=JSON.parse(readFileSync(new URL('../../tally-ads-v5-validator/test/fixtures/summary.json',import.meta.url)));
const correction=JSON.parse(readFileSync(new URL('../../tally-ads-v5-validator/test/fixtures/correction.json',import.meta.url)));
const repo='example-org/example-repo',issueNumber=42,sha='a'.repeat(40);
// JCS hash is verified by native transport; the pure reconciler compares it exactly.
const dig=()=>'e'.repeat(64);
function pair() {
  const sum=structuredClone(summary);
  sum.data.limits=[];sum.data.capture_coverage='COMPLETE';
  const corr=structuredClone(correction);
  corr.data.limits=[];corr.data.target.data_digest=dig(sum);
  const subject=sum.data.subject;
  Object.assign(corr.data.target,{
    comment_id:123,comment_ref:'https://github.com/example-org/example-repo/issues/42#issuecomment-123',
    target_source_sha:subject.source_sha,
  });
  return [
    {envelope:sum,validation:{schema:'ads/github/v5',passed:true},
      native:{repository:repo,issueNumber,commentId:123,sourceSha:sha,
        dataDigest:dig(sum),permission:'GRANTED',readback:'VERIFIED',
        pagination:'COMPLETE',currentness:'CURRENT'}},
    {envelope:corr,validation:{schema:'ads/github/v5',passed:true},
      native:{repository:repo,issueNumber,commentId:124,sourceSha:sha,
        dataDigest:dig(corr),permission:'GRANTED',readback:'VERIFIED',
        pagination:'COMPLETE',currentness:'CURRENT'}}
  ];
}
test('ordinary valid summary keeps CURRENT_OBSERVED advisory, never Task PASS',()=>{
  const result=reconcileCorrectionNotices(pair().slice(0,1));
  assert.equal(result.status,STATES.CURRENT);
  assert.equal(result.reconciliation,'NO_NOTICES');
  assert.equal(result.effectiveRecords[0].status,'OBSERVED_ONLY');
  assert.equal(result.taskPass,false);
  assert.equal(result.authorized,false);
});
for(const [notice,expected] of [['REVOKE','REVOKED'],['REDACT','REDACTED'],['DISPUTE','DISPUTED']]) {
  test('actual schema v5 '+notice+' applies only exact target, never restores CURRENT',()=>{
    const records=pair();records[1].envelope.data.notice=notice;
    const result=reconcileCorrectionNotices(records);
    assert.equal(result.status,STATES.UNKNOWN);
    assert.equal(result.reconciliation,'NOTICES_APPLIED_ADVISORY');
    assert.equal(result.corrections[0].status,'MATCHED');
    assert.equal(result.effectiveRecords[0].status,expected);
    assert.equal(result.taskPass,false);assert.equal(result.releasePass,false);
  });
}
test('wrong JCS digest must not apply to target even when comment id matches',()=>{
  const records=pair();
  records[1].envelope.data.target.data_digest='f'.repeat(64);
  const r=reconcileCorrectionNotices(records);
  assert.equal(r.status,STATES.UNKNOWN);
  assert.equal(r.reconciliation,'UNRESOLVED');
  assert.equal(r.corrections[0].reason,'NOTICE_TARGET_UNVERIFIED');
  assert.equal(r.effectiveRecords[0].status,'NOT_CURRENT');
});
test('cross-publisher target and cross-Issue scope fail closed',()=>{
  for(const mutate of [
    x=>{x.envelope.data.target.publisher_namespace='evil:other';},
    x=>{x.envelope.data.target.repository='evil/other';},
    x=>{x.envelope.data.target.issue_ref='https://github.com/evil/other/issues/42';},
    x=>{x.envelope.data.target.task_ref='task:another';},
    x=>{x.envelope.data.target.target_source_sha='b'.repeat(40);},
    x=>{x.envelope.data.target.revision=2;},
  ]) {
    const records=pair();mutate(records[1]);
    const r=reconcileCorrectionNotices(records);
    assert.equal(r.status,STATES.UNKNOWN);
    assert.equal(r.reconciliation,'UNRESOLVED');
    assert.equal(r.corrections[0].status,'UNRESOLVED');
    assert.equal(r.taskPass,false);
  }
});
test('edited or not-current notice may not be used as an applied correction',()=>{
  const records=pair();records[1].native.currentness='STALE';
  const r=reconcileCorrectionNotices(records);
  assert.equal(r.status,STATES.UNKNOWN);
  assert.equal(r.corrections[0].reason,'NOTICE_NOT_CURRENT_OR_INVALID');
});
test('duplicate apparently valid notices cannot become one authoritative correction',()=>{
  const records=pair();
  const second=structuredClone(records[1]);second.native.commentId=125;
  records.push(second);
  const r=reconcileCorrectionNotices(records);
  assert.notEqual(r.status,STATES.CURRENT);
  assert.equal(r.reconciliation,'UNRESOLVED');
  assert.equal(r.corrections.at(-1).reason,'DUPLICATE_OR_CONFLICTING_NOTICE');
  assert.equal(r.effectiveRecords[0].status,'NOT_CURRENT');
});
test('missing referenced source denies, and non-array input never throws',()=>{
  const records=pair().slice(1);
  const r=reconcileCorrectionNotices(records);
  assert.equal(r.reconciliation,'UNRESOLVED');
  assert.equal(r.corrections[0].reason,'NOTICE_TARGET_UNVERIFIED');
  assert.equal(reconcileCorrectionNotices(null).status,STATES.INVALID);
  assert.equal(reconcileCorrectionNotices(Array(1001).fill(null)).status,STATES.INVALID);
});

test('ADS v4.18 rejects correction whose OWN subject Task or Attempt differs from target',()=>{
  for(const field of ['task_ref','attempt_ref']){
    const records=pair();
    records[1].envelope.data.subject[field]='fixture:cross-boundary';
    const r=reconcileCorrectionNotices(records);
    assert.equal(r.status,STATES.UNKNOWN);
    assert.equal(r.reconciliation,'UNRESOLVED');
    assert.equal(r.corrections[0].reason,'NOTICE_CROSS_SCOPE_OR_PUBLISHER');
    assert.equal(r.effectiveRecords[0].status,'NOT_CURRENT');
    assert.equal(r.taskPass,false);
  }
});
test('ADS v4.18 rejects a correction whose own publisher source sha mismatches',()=>{
  const records=pair();
  records[1].envelope.data.notice_publisher_source_sha='b'.repeat(40);
  const r=reconcileCorrectionNotices(records);
  assert.equal(r.status,STATES.UNKNOWN);
  assert.equal(r.reconciliation,'UNRESOLVED');
  assert.equal(r.corrections[0].reason,'NOTICE_CROSS_SCOPE_OR_PUBLISHER');
  assert.equal(r.taskPass,false);
});
test('an edited target is not a validated correction target',()=>{
  const records=pair();records[0].native.currentness='STALE';
  const r=reconcileCorrectionNotices(records);
  assert.equal(r.status,STATES.UNKNOWN);
  assert.equal(r.reconciliation,'UNRESOLVED');
  assert.equal(r.corrections[0].reason,'NOTICE_TARGET_NOT_CURRENT');
  assert.equal(r.effectiveRecords[0].status,'NOT_CURRENT');
});
