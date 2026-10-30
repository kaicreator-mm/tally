import test from 'node:test';
import assert from 'node:assert/strict';
import {readFileSync} from 'node:fs';
import {replayCasebook,affectedCases} from '../src/casebook.mjs';

const fixture=JSON.parse(readFileSync(new URL('../../../experiments/v2/casebook/tally-v2-casebook-r0.json',import.meta.url),'utf8'));
const asOf='2026-10-31T07:00:00.000Z';
const projections=(book=fixture)=>Object.fromEntries(book.cases.map((row,index)=>[row.case_id,{
  decisions:[...row.expected_decisions],observedAt:row.permitted_inputs.observedAt,baselineErrors:index===1||index===2?1:0}]));
const replay=(mode='RETROSPECTIVE_DIAGNOSIS',book=fixture,inputs=projections(book))=>replayCasebook(book,{mode,asOf,projections:inputs});

test('real Casebook covers three versions cross-project and joint decisions without contradictions',()=>{
  const result=replay();
  assert.equal(result.status,'OBSERVED');assert.equal(result.cases.length,8);
  assert.ok(new Set(fixture.cases.map(row=>row.version)).size>=3);
  assert.ok(new Set(fixture.cases.map(row=>row.project)).size>=3);
  assert.equal(result.coverage.passCount,8);assert.equal(result.coverage.failCount,0);
  assert.equal(result.coverage.conflictSafetyCount,2);assert.equal(result.coverage.unknownHonestyCount,6);
  assert.equal(result.coverage.denominatorConsistencyCount,2);assert.equal(result.coverage.nextActionUsefulCount,7);
  assert.equal(result.coverage.naiveBaselineErrorCount,2);assert.equal(result.coverage.candidateErrorCount,0);
  for(const row of result.cases){
    assert.equal(row.status,'PASS');assert.ok(row.joint.timeline);assert.ok(row.joint.diagnosis);assert.ok(row.joint.planning);
    assert.deepEqual(row.forbiddenDecisions,[]);
  }
  assert.equal(result.coverage.activeHours,'NOT_MEASURED');assert.equal(result.coverage.productionRoi,'NOT_MEASURED');
});

test('blind replay excludes unrecoverable history rather than claiming prediction',()=>{
  const result=replay('HISTORICALLY_BLIND');
  assert.equal(result.coverage.notReconstructableCount,2);assert.equal(result.coverage.passCount,6);
  assert.equal(result.coverage.blindPredictiveCount,6);
  for(const id of ['dh-v010-1264-not-invoked','tm-v04-64-currentness'])
    assert.equal(result.cases.find(row=>row.case_id===id).status,'NOT_RECONSTRUCTABLE');
  const inputs=projections();inputs['dh-v0102-1354-convergence'].observedAt='2026-10-25T00:00:00.000Z';
  const later=replay('HISTORICALLY_BLIND',fixture,inputs);
  assert.equal(later.cases.find(row=>row.case_id==='dh-v0102-1354-convergence').status,'NOT_RECONSTRUCTABLE');
  assert.equal(later.coverage.notReconstructableCount,3);assert.equal(later.coverage.blindPredictiveCount,5);
});

test('forbidden security waiver and missing decisions become visible candidate failures',()=>{
  const inputs=projections();inputs['dh-v081-1100-security'].decisions.push('WAIVE_P0_FOR_COST');
  inputs['dh-v08-1013-collision'].decisions=['INTEGRATION_SEAM_GAP'];
  const result=replay('RETROSPECTIVE_DIAGNOSIS',fixture,inputs);
  assert.equal(result.cases.find(row=>row.case_id==='dh-v081-1100-security').status,'FAIL');
  assert.equal(result.cases.find(row=>row.case_id==='dh-v08-1013-collision').status,'FAIL');
  assert.equal(result.coverage.falseBlockerCount,1);assert.equal(result.coverage.candidateErrorCount,2);
});

test('blind replay rejects later-knowledge decisions even with forged timestamps',()=>{
  const inputs=projections();inputs['dh-v081-1100-security'].decisions.push('DECISION_FROM_LATER_KNOWLEDGE');
  const result=replay('HISTORICALLY_BLIND',fixture,inputs);
  const row=result.cases.find(value=>value.case_id==='dh-v081-1100-security');
  assert.equal(row.status,'FAIL');assert.deepEqual(row.leakedDecisions,['DECISION_FROM_LATER_KNOWLEDGE']);
});

test('affected-case selection limits regression replay to linked sources',()=>{
  assert.deepEqual(affectedCases(fixture,['https://github.com/kaicreator-mm/domain-harness/pull/1016']),['dh-v08-1013-collision']);
  assert.deepEqual(affectedCases(fixture,['https://github.com/kaicreator-mm/domain-harness/issues/586']),['dh-v07-586-campaign']);
  assert.deepEqual(affectedCases(fixture,['https://github.com/kaicreator-mm/domain-harness/pull/9999']),[]);
});

test('invalid schema projections or insufficient coverage cannot produce PASS or Release',()=>{
  assert.equal(replayCasebook(null,{mode:'RETROSPECTIVE_DIAGNOSIS',asOf,projections:{}}).status,'UNKNOWN');
  const missing=replay('RETROSPECTIVE_DIAGNOSIS',fixture,{});
  assert.equal(missing.status,'OBSERVED');assert.equal(missing.coverage.passCount,0);
  assert.equal(missing.cases[0].status,'UNKNOWN');assert.equal(missing.coverage.releasePass,false);
  const small={schema:'tally/casebook-v1',cases:fixture.cases.slice(0,2)};
  assert.equal(replayCasebook(small,{mode:'RETROSPECTIVE_DIAGNOSIS',asOf,projections:{}}).status,'UNKNOWN');
  const duplicate={...fixture,cases:fixture.cases.map((row,index)=>index===0?{...row,case_id:fixture.cases[1].case_id}:row)};
  assert.equal(replayCasebook(duplicate,{mode:'RETROSPECTIVE_DIAGNOSIS',asOf,projections:projections(duplicate)}).status,'UNKNOWN');
  const noBaseline=projections();delete noBaseline['dh-v07-530-false-serial'].baselineErrors;
  assert.equal(replay('RETROSPECTIVE_DIAGNOSIS',fixture,noBaseline).cases[0].status,'UNKNOWN');
});
