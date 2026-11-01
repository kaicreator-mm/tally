import test from 'node:test';
import assert from 'node:assert/strict';
import {durationLabel,tokenLabel,telemetryCoverage,byAgent,tokenTotals}
  from './model.ts';
import type {AttemptRecord} from './model.ts';

const make=(over:Partial<AttemptRecord>):AttemptRecord=>({
  attemptId:'a1',taskIds:['101'],agentId:'zcode',state:'EXITED_OK',
  startedAt:'2026-11-01T00:00:00Z',terminatedAt:'2026-11-01T00:01:00Z',
  durationMs:60000,tokensIn:1000,tokensOut:2000,costUsd:null,
  model:'glm',provider:'zai',...over});
const attempts:AttemptRecord[]=[
  make({}),
  make({attemptId:'a2',agentId:'kimi',durationMs:null,tokensIn:null,tokensOut:null,
    state:'UNKNOWN'}),
  make({attemptId:'a3',agentId:'zcode',durationMs:125000,tokensIn:500,
    tokensOut:null,state:'RUNNING'}),
];
test('UNKNOWN telemetry stays UNKNOWN, never zero or inferred',()=>{
  assert.equal(durationLabel(attempts[1]),'UNKNOWN');
  assert.equal(tokenLabel(attempts[1],'tokensIn'),'UNKNOWN');
  assert.equal(durationLabel(attempts[0]),'1m0s');
  assert.equal(durationLabel(attempts[2]),'2m5s');
});
test('coverage denominators distinguish missing from measured',()=>{
  const coverage=telemetryCoverage(attempts);
  assert.deepEqual(coverage,{measured:2,total:3,coverage:2/3});
  assert.deepEqual(telemetryCoverage([]),{measured:0,total:0,coverage:null});
});
test('token totals only over measured attempts with coverage attached',()=>{
  const totals=tokenTotals(attempts);
  assert.equal(totals.tokensIn,1500); // 1000 + 500, UNKNOWN a2 excluded
  assert.equal(totals.tokensOut,2000);
  assert.equal(totals.inCoverage,2/3);
  assert.equal(totals.outCoverage,1/3);
});
test('grouping by agent keeps attempts explicit',()=>{
  const groups=byAgent(attempts);
  assert.deepEqual(groups.map(group=>group.agentId),['zcode','kimi']);
  assert.deepEqual(groups[0].attempts.map(attempt=>attempt.attemptId),['a1','a3']);
});
