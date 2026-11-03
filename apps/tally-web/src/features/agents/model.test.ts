import test from 'node:test';
import assert from 'node:assert/strict';
import {durationLabel,tokenLabel,telemetryCoverage,byAgent,tokenTotals,
  sanitizeAttempts,UNKNOWN_TEXT}
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
test('attempt role is optional and never inferred from provider identity',()=>{
  assert.equal(attempts[0].role,undefined);
  assert.equal(make({role:'REVIEWER'}).role,'REVIEWER');
  assert.equal(make({role:null}).role,null);
});
test('grouping by agent keeps attempts explicit',()=>{
  const groups=byAgent(attempts);
  assert.deepEqual(groups.map(group=>group.agentId),['zcode','kimi']);
  assert.deepEqual(groups[0].attempts.map(attempt=>attempt.attemptId),['a1','a3']);
});
test('sanitizeAttempts redacts credential-shaped and malformed identifiers',()=>{
  const unsafe=sanitizeAttempts([make({
    attemptId:'ghp_12345678901234567890',
    agentId:'github_pat_abcdefghijklmnopqrstuvwxyz012345',
    provider:'Bearer abcdefghijklmnopqrstuvwxyz',
    model:'sk-proj-abcdefghijklmnopqrstuvwxyz123456',
    taskIds:['101','x'.repeat(64),'ok-2','101'],
    state:'SOMETHING_ELSE' as AttemptRecord['state'],
    role:'ROOT' as NonNullable<AttemptRecord['role']>,
    durationMs:-5,tokensIn:-1,tokensOut:Number.NaN})]);
  assert.equal(unsafe.length,1);
  const attempt=unsafe[0];
  assert.equal(attempt.attemptId,UNKNOWN_TEXT);
  assert.equal(attempt.agentId,UNKNOWN_TEXT);
  assert.equal(attempt.provider,null);
  assert.equal(attempt.model,null);
  assert.equal(attempt.state,'UNKNOWN');
  assert.equal(attempt.role,null);
  assert.equal(attempt.durationMs,null);
  assert.equal(attempt.tokensIn,null);
  assert.equal(attempt.tokensOut,null);
  assert.deepEqual(attempt.taskIds,['101','ok-2']);
  assert.equal(JSON.stringify(attempt).includes('ghp_'),false);
  assert.equal(JSON.stringify(attempt).includes('github_pat_'),false);
  assert.equal(JSON.stringify(attempt).includes('Bearer'),false);
  assert.equal(JSON.stringify(attempt).includes('sk-proj-'),false);
});
test('sanitizeAttempts keeps valid bounded records semantically intact',()=>{
  const safe=sanitizeAttempts([attempts[0],attempts[1]]);
  assert.deepEqual(safe.map(attempt=>attempt.attemptId),['a1','a2']);
  assert.equal(safe[0].provider,'zai');
  assert.equal(safe[0].model,'glm');
  assert.equal(safe[0].state,'EXITED_OK');
  assert.deepEqual(safe[0].taskIds,['101']);
  assert.equal(safe[0].durationMs,60000);
  assert.equal(safe[0].tokensIn,1000);
  assert.equal(safe[0].tokensOut,2000);
  assert.equal(safe[0].role,undefined);
  assert.equal(safe[1].durationMs,null);
  assert.equal(safe[1].state,'UNKNOWN');
  assert.equal(durationLabel(safe[0]),'1m0s');
  assert.equal(tokenLabel(safe[1],'tokensIn'),'UNKNOWN');
});
test('sanitizeAttempts bounds oversized identifiers to UNKNOWN',()=>{
  const unsafe=sanitizeAttempts([make({
    attemptId:'a'.repeat(200),agentId:'z'.repeat(200),
    provider:'p'.repeat(200),model:'m'.repeat(200),
    taskIds:Array.from({length:64},(_,i)=>`t${i}`)})]);
  assert.equal(unsafe[0].attemptId,UNKNOWN_TEXT);
  assert.equal(unsafe[0].agentId,UNKNOWN_TEXT);
  assert.equal(unsafe[0].provider,null);
  assert.equal(unsafe[0].model,null);
  assert.equal(unsafe[0].taskIds.length,32);
});
